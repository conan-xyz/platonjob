package internal

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"gitee.com/zonzpoo/platonjob/client"
	"gitee.com/zonzpoo/platonjob/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p/discv5"
	"github.com/ethereum/go-ethereum/rlp"
)

// Addr ...
type Addr struct {
	PrivateKey *ecdsa.PrivateKey
	Address    common.Address
	ArpStr     string
	NodeId     discv5.NodeID
}

func NewAddr(privateKey, hrp, nodeId string) (addr *Addr, err error) {
	var (
		ok bool
	)
	addr = &Addr{}
	addr.PrivateKey, err = crypto.HexToECDSA(privateKey)
	if err != nil {
		return
	}
	publicKey := addr.PrivateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		err = fmt.Errorf("publicKey is not vaild")
		return
	}
	addr.Address = crypto.PubkeyToAddress(*publicKeyECDSA)
	addr.ArpStr, err = utils.ConvertAndEncode(hrp, addr.Address.Bytes())
	if err != nil {
		return
	}
	addr.NodeId, err = discv5.HexID(nodeId)
	if err != nil {
		return
	}
	return
}

func (d *Addr) RewardMsg(ctx context.Context, arp string) (msg client.CallMsg, err error) {
	rewardCode := int64(5100)
	address := utils.ContractAddr(rewardCode)
	if address == "" {
		err = fmt.Errorf("invalid contract code: %d", rewardCode)
		return
	}
	arpStr, err := d.toArp(ctx, arp)
	if address == "" {
		err = fmt.Errorf("invalid address: %s", err)
		return
	}
	contractAddrByte := common.HexToAddress(address)
	contractAddr, err := utils.ConvertAndEncode(arp, contractAddrByte.Bytes())
	if address == "" {
		err = fmt.Errorf("invalid contract address: %s", err)
		return
	}
	nodes := new([]discv5.NodeID)
	buf, err := d.bufData(rewardCode, nodes, d.Address)
	if err != nil {
		return
	}
	msg = client.CallMsg{
		From:     arpStr,
		To:       contractAddr,
		Gas:      103496,
		GasPrice: big.NewInt(500000000000),
		Data:     buf,
	}
	return
}

func (d *Addr) bufData(rewardCode int64, nodes *[]discv5.NodeID, address common.Address) (buf []byte, err error) {
	fnType, err := rlp.EncodeToBytes(uint16(rewardCode))
	if err != nil {
		return
	}
	nByte, err := rlp.EncodeToBytes(nodes)
	if err != nil {
		return
	}
	addressByte, err := rlp.EncodeToBytes(address)
	if err != nil {
		return
	}

	params := make([][]byte, 0)
	params = append(params, fnType)
	params = append(params, addressByte)
	params = append(params, nByte)

	byteBuf := new(bytes.Buffer)
	err = rlp.Encode(byteBuf, params)
	if err != nil {
		return
	}
	buf = byteBuf.Bytes()
	return
}

func (d *Addr) toArp(ctx context.Context, arp string) (arpStr string, err error) {
	return utils.ConvertAndEncode(arp, d.Address.Bytes())
}
