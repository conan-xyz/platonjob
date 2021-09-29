package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	tp "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"k8s.io/klog"

	"gitee.com/zonzpoo/platonjob/utils"
	"gitee.com/zonzpoo/platonjob/utils/types"
)

const (
	rewardCode     = int64(5000)
	rewardGasLimit = uint64(35040)
)

// Reward ...
type Reward struct {
	SvcImpl

	ctx   context.Context
	addrs []*Addr

	send    chan *Addr
	receipt chan *Receipt

	receipts int32
	total    int32

	exit chan struct{}
}

func (r *Reward) Start() {
	go r.report()
	go r.run()

	for _, addr := range r.addrs {
		r.send <- addr
		time.Sleep(100 * time.Millisecond)
	}
}

func (r *Reward) report() {
	timeout := time.After(time.Duration(r.total) * 2 * time.Second)
	t := time.NewTicker(time.Millisecond * 500)
	for {
		select {
		case <-t.C:
			klog.Infof("[Reward report] current total: %d, receipts: %d", r.total, r.receipts)
			if r.total == r.receipts {
				close(r.exit)
			}
		case <-timeout:
			klog.Infof("[Reward report] timeout, total: %d", r.total)
			close(r.exit)
		case <-r.ctx.Done():
			close(r.exit)
		case <-r.exit:
			return
		}
	}
}

func (r *Reward) run() {
	for {
		select {
		case addr := <-r.send:
			klog.Infof("[Reward run] receive address: %s, begin send transaction", addr.ArpStr)
			go r.sendTransaction(addr)
		case receipt := <-r.receipt:
			err := receipt.err
			if err != nil {
				klog.Errorf("[Reward run] current address: %s, get reward err: %s", receipt.addr.ArpStr, err)
			} else {
				if r.IsAsync() {
					klog.Infof("[Reward run] current address: %s", receipt.addr.ArpStr)
				} else {
					klog.Infof("[Reward run] current address: %s, get reward hash tx: %s", receipt.addr.ArpStr, receipt.tx.Hash().Hex())
				}
			}
			go r.add()
		case <-r.exit:
			return
		}
	}
}

func (r *Reward) add() {
	atomic.AddInt32(&r.receipts, 1)
}

func (r *Reward) sendTransaction(addr *Addr) {
	var (
		tx  *tp.Transaction
		err error
	)
	defer func() {
		<-time.After(time.Second)
		r.receipt <- &Receipt{
			addr: addr,
			tx:   tx,
			err:  err,
		}
	}()

	reward, err := r.ListRewards(r.ctx, addr)
	if err != nil {
		err = fmt.Errorf("[Reward sendTransaction] current address: %s, list reward error: %s", addr.ArpStr, err)
		return
	}
	if reward.Cmp(big.NewInt(0).SetInt64(utils.BaseVon)) == -1 {
		err = fmt.Errorf("[Reward sendTransaction] current address: %s reward less then 1: %s", addr.ArpStr, utils.HumReadBalance(reward))
		return
	}
	nonce, err := r.GetNonce(r.ctx, addr.ArpStr)
	if err != nil {
		err = fmt.Errorf("[Reward sendTransaction] current address: %s get nonce err: %s", addr.ArpStr, err)
		return
	}
	tx, err = r.RunReward(r.ctx, addr, nonce)
	if err != nil {
		err = fmt.Errorf("[Reward sendTransaction] current address %s get reward failed %s", addr.ArpStr, err)
		return
	}
	klog.Infof("[Reward sendTransaction] finished send get_reward, current address: %s, nonce: %d", addr.ArpStr, nonce)
}

// ListRewards list address rewards
func (s *Service) ListRewards(ctx context.Context, addr *Addr) (reward *big.Int, err error) {
	reward = big.NewInt(0)
	msg, err := addr.RewardMsg(context.TODO(), s.Arp)
	if err != nil {
		return
	}

	rewardByte, err := s.client.CallContract(ctx, msg, nil)
	if err != nil {
		return
	}

	var can types.RewardResponse
	err = json.Unmarshal(rewardByte, &can)
	if err != nil {
		return
	}
	ret := can.Ret
	for _, info := range ret {
		i, err := hexutil.DecodeBig(info.Reward)
		if err != nil {
			continue
		}
		reward.Add(reward, i)
	}
	return
}

func (s *Service) WithdrawReward(ctx context.Context) {
	addrs := []*Addr{}
	for _, address := range s.Addrs {
		addr, err := NewAddr(address.PrivateKey, s.Arp, address.NodeID)
		if err != nil {
			panic(err)
		}
		addrs = append(addrs, addr)
	}
	reward := &Reward{
		SvcImpl: s,
		ctx:     ctx,
		addrs:   addrs,

		send:    make(chan *Addr, len(addrs)),
		receipt: make(chan *Receipt, len(addrs)),

		receipts: 0,
		total:    int32(len(addrs)),
		exit:     make(chan struct{}),
	}
	go reward.Start()
}

func (s *Service) RunReward(ctx context.Context, addr *Addr, nonce uint64) (tx *tp.Transaction, err error) {
	var (
		gasPrice *big.Int
	)
	address := utils.ContractAddr(rewardCode)
	if address == "" {
		err = fmt.Errorf("invalid contract code: %d", rewardCode)
		return
	}
	buf, err := rewardBufData()
	if err != nil {
		return
	}
	gasPrice, err = s.client.GasPrice(ctx)
	if err != nil {
		return
	}
	if s.IsAsync() {
		gasPrice = big.NewInt(0)
	}

	tx, err = tp.SignTx(
		tp.NewTransaction(
			nonce,
			common.HexToAddress(address),
			big.NewInt(0),
			s.RewardGasLimit,
			gasPrice,
			buf),
		s.signer,
		addr.PrivateKey)
	if err != nil {
		return
	}
	err = s.client.SendTransaction(context.Background(), tx)
	return
}

func rewardBufData() (buf []byte, err error) {
	fnType, err := rlp.EncodeToBytes(uint16(rewardCode))
	if err != nil {
		return
	}

	params := make([][]byte, 0)
	params = append(params, fnType)

	byteBuf := new(bytes.Buffer)
	err = rlp.Encode(byteBuf, params)
	if err != nil {
		return
	}
	buf = byteBuf.Bytes()
	return
}
