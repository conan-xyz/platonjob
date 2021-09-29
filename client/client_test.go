package client

import (
	"context"
	"testing"
)

func TestGasPrice(t *testing.T) {
	c, err := DialContext(context.TODO(), "https://openapi.alaya.network/rpc")
	if err != nil {
		t.Error(err)
	}
	n, err := c.GasPrice(context.TODO())
	if err != nil {
		t.Error(err)
	}
	t.Log(n.Int64())
}
