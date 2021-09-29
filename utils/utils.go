package utils

import (
	"errors"
	"math/big"

	"github.com/btcsuite/btcutil/bech32"
)

const (
	// BaseVon = 1
	BaseVon = 1000000000000000000
	// TBaseVon = 10
	TBaseVon = 10000000000000000000
	// HBaseVon = 100
	HBaseVon = 100000000000000000000
	// KBaseVon = 100
	KBaseVon = 100000000000000000000
)

// ContractAddr ...
func ContractAddr(fnType int64) (address string) {
	switch {
	case fnType >= 1000 && fnType < 2000:
		address = "0x1000000000000000000000000000000000000002"
	case fnType >= 2000 && fnType < 3000:
		address = "0x1000000000000000000000000000000000000005"
	case fnType >= 3000 && fnType < 4000:
		address = "0x1000000000000000000000000000000000000004"
	case fnType >= 4000 && fnType < 5000:
		address = "0x1000000000000000000000000000000000000001"
	case fnType >= 5000 && fnType < 6000:
		address = "0x1000000000000000000000000000000000000006"
	default:
	}
	return
}

//ConvertAndEncode converts from a base64 encoded byte string to base32 encoded byte string and then to bech32
func ConvertAndEncode(hrp string, data []byte) (string, error) {
	//this is base32
	converted, err := bech32.ConvertBits(data, 8, 5, true)

	if err != nil {
		return "", errors.New("encoding bech32 failed")
	}
	return bech32.Encode(hrp, converted)
}

func HumReadBalance(balance *big.Int) string {
	baseVon := big.NewFloat(0).SetFloat64(BaseVon)
	balanceVon := big.NewFloat(0).SetInt(balance)
	return big.NewFloat(0).Quo(balanceVon, baseVon).String()
}
