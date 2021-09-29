package internal

import (
	"bytes"
	"context"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"gitee.com/zonzpoo/platonjob/utils"
	"github.com/ethereum/go-ethereum/common"
	tp "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/rlp"
	"k8s.io/klog"
)

// https://devdocs.platon.network/docs/zh-CN/Economic_Model  Gas calculation rules for built-in transactions
const (
	delegateCode = int64(1004)
)

type Delegate struct {
	SvcImpl

	ctx   context.Context
	addrs []*Addr

	send    chan *Addr
	receipt chan *Receipt

	receipts int32
	total    int32

	exit chan struct{}
}

func (d *Delegate) Start() {
	go d.report()
	go d.run()

	for _, addr := range d.addrs {
		d.send <- addr
		time.Sleep(100 * time.Millisecond)
	}
}

func (d *Delegate) report() {
	timeout := time.After(time.Duration(d.total) * 2 * time.Second)
	t := time.NewTicker(time.Millisecond * 500)
	for {
		select {
		case <-t.C:
			klog.Infof("[Delegate report] current total: %d, receipts: %d", d.total, d.receipts)
			if d.total == d.receipts {
				close(d.exit)
			}
		case <-timeout:
			klog.Infof("[Delegate report] timeout, total: %d", d.total)
			close(d.exit)
		case <-d.ctx.Done():
			close(d.exit)
		case <-d.exit:
			return
		}
	}
}

func (d *Delegate) add() {
	atomic.AddInt32(&d.receipts, 1)
}

func (d *Delegate) run() {
	for {
		select {
		case addr := <-d.send:
			klog.Infof("[Delegate run] receive address: %s, begin send transaction", addr.ArpStr)
			go d.sendTransaction(addr)
		case receipt := <-d.receipt:
			if receipt.err != nil {
				klog.Errorf("[Reward run] current address: %s, get initiate delegate err: %s", receipt.addr.ArpStr, receipt.err)
			} else {
				if d.IsAsync() {
					klog.Infof("[Delegate run] current address: %s", receipt.addr.ArpStr)
				} else {
					klog.Infof("[Delegate run] current address: %s, get initiate delegate hash tx: %s", receipt.addr.ArpStr, receipt.tx.Hash().Hex())
				}
			}
			go d.add()
		case <-d.exit:
			return
		}
	}
}

func (d *Delegate) sendTransaction(addr *Addr) {
	var (
		tx  *tp.Transaction
		err error
	)

	defer func() {
		<-time.After(time.Second)
		d.receipt <- &Receipt{
			addr: addr,
			tx:   tx,
			err:  err,
		}
	}()

	nonce, err := d.GetNonce(d.ctx, addr.ArpStr)
	if err != nil {
		err = fmt.Errorf("[Delegate sendTransaction] current address: %s, get nonce error: %s", addr.ArpStr, err)
		return
	}

	baseVon := big.NewFloat(0).SetFloat64(utils.BaseVon)
	delegateValue, err := d.GetDelegateValue(d.ctx, addr.ArpStr)
	if err != nil {
		err = fmt.Errorf("[Delegate sendTransaction] current address: %s, get delegate value error: %s", addr.ArpStr, err)
		return
	}
	if delegateValue.Cmp(d.MinVon()) == -1 {
		err = fmt.Errorf("[Delegate sendTransaction] current address: %s, delegate value: %s", addr.ArpStr, big.NewFloat(0).Quo(delegateValue, baseVon).String())
		return
	}
	realdelegateValue, _ := delegateValue.Int(big.NewInt(0))
	tx, err = d.RunDelegate(d.ctx, addr.NodeId, realdelegateValue, addr, nonce)
	if err != nil {
		err = fmt.Errorf("[Delegate sendTransaction] current address %s run delegate failed %s", addr.ArpStr, err)
		return
	}
	klog.Infof("[Delegate sendTransaction] finished send delegate, current address: %s, nonce: %d", addr.ArpStr, nonce)
}

func (s *Service) InitDelegate(ctx context.Context) {
	addrs := []*Addr{}
	for _, address := range s.Addrs {
		addr, err := NewAddr(address.PrivateKey, s.Arp, address.NodeID)
		if err != nil {
			panic(err)
		}
		addrs = append(addrs, addr)
	}
	delegate := &Delegate{
		SvcImpl: s,
		ctx:     ctx,
		addrs:   addrs,

		send:    make(chan *Addr, len(addrs)),
		receipt: make(chan *Receipt, len(addrs)),

		receipts: 0,
		total:    int32(len(addrs)),
		exit:     make(chan struct{}),
	}
	go delegate.Start()
}

func (s *Service) RunDelegate(ctx context.Context, nodeID discv5.NodeID, amount *big.Int, addr *Addr, nonce uint64) (tx *tp.Transaction, err error) {
	var (
		gasPrice *big.Int
	)
	address := utils.ContractAddr(delegateCode)
	if address == "" {
		err = fmt.Errorf("invalid contract code: %d", delegateCode)
		return
	}

	buf, err := s.delegateBufData(nodeID, amount)
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
			big.NewInt(1),
			s.DelegateGasLimit,
			gasPrice,
			buf),
		s.signer,
		addr.PrivateKey)
	if err != nil {
		return
	}
	err = s.client.SendTransaction(ctx, tx)
	return
}

func (s *Service) delegateBufData(node discv5.NodeID, amount *big.Int) (buf []byte, err error) {
	fnType, err := rlp.EncodeToBytes(uint16(delegateCode))
	if err != nil {
		return
	}
	typ, err := rlp.EncodeToBytes(uint16(0))
	if err != nil {
		return
	}
	nodeID, err := rlp.EncodeToBytes(node)
	if err != nil {
		return
	}
	amountBytes, err := rlp.EncodeToBytes(amount)
	if err != nil {
		return
	}

	params := make([][]byte, 0)
	params = append(params, fnType)
	params = append(params, typ)
	params = append(params, nodeID)
	params = append(params, amountBytes)

	byteBuf := new(bytes.Buffer)
	err = rlp.Encode(byteBuf, params)
	if err != nil {
		return
	}
	buf = byteBuf.Bytes()
	return
}
