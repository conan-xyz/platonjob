package internal

import (
	"context"
	"math/big"

	tp "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/p2p/discv5"

	"gitee.com/zonzpoo/platonjob/client"
	"gitee.com/zonzpoo/platonjob/conf"
	"gitee.com/zonzpoo/platonjob/utils"
)

type SvcImpl interface {
	IsAsync() bool

	CurrentBlockNumber(ctx context.Context) (number int64)
	GetNonce(ctx context.Context, arpStr string) (uint64, error)
	GetBalance(ctx context.Context, arpStr string) (*big.Int, error)

	// award
	ListRewards(ctx context.Context, addr *Addr) (*big.Int, error)
	RunReward(ctx context.Context, addr *Addr, nonce uint64) (*tp.Transaction, error)
	WithdrawReward(ctx context.Context)

	// delegate
	MinVon() *big.Float
	GetDelegateValue(ctx context.Context, arpStr string) (*big.Float, error)
	RunDelegate(ctx context.Context, nodeID discv5.NodeID, amount *big.Int, addr *Addr, nonce uint64) (*tp.Transaction, error)
	InitDelegate(ctx context.Context)
}

type Service struct {
	*conf.Config

	client *client.Client
	signer tp.EIP155Signer
	async  *bool
}

type Receipt struct {
	addr *Addr
	tx   *tp.Transaction

	err error
}

func New(ctx context.Context, ac *conf.Config) (svc SvcImpl, err error) {
	client, err := client.DialContext(ctx, ac.RawURL)
	if err != nil {
		return
	}
	svc = &Service{Config: ac, client: client, async: ac.Async, signer: tp.NewEIP155Signer(big.NewInt(ac.ChainID))}
	return
}

func (s *Service) MinVon() *big.Float {
	return big.NewFloat(0).SetFloat64(s.MinDelegate)
}

func (s *Service) GetNonce(ctx context.Context, arpStr string) (nonce uint64, err error) {
	return s.client.NonceAt(ctx, arpStr, nil)
}

func (s *Service) GetBalance(ctx context.Context, arpStr string) (balance *big.Int, err error) {
	return s.client.BalanceAt(ctx, arpStr, nil)
}

func (s *Service) GetDelegateValue(ctx context.Context, arpStr string) (value *big.Float, err error) {
	balance, err := s.client.BalanceAt(ctx, arpStr, nil)
	if err != nil {
		return
	}
	baseVon := big.NewFloat(0).SetFloat64(utils.BaseVon)
	balanceVon := big.NewFloat(0).SetInt(balance)
	value = balanceVon.Sub(balanceVon, baseVon.Mul(baseVon, big.NewFloat(0).SetFloat64(0.1)))
	return
}

func (s *Service) IsAsync() (async bool) {
	as := s.Async
	if as == nil {
		// default
		async = false
	}
	return *as
}

func (s *Service) CurrentBlockNumber(ctx context.Context) (number int64) {
	bInt, err := s.client.BlockNumberAt(ctx)
	if err != nil {
		return
	}
	number = bInt.Int64()
	return
}
