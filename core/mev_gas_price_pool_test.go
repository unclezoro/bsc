package core

import (
	"math/big"
	"testing"
	"time"
)

func TestMevGasPricePool(t *testing.T) {
	pool := NewMevGasPricePool(1 * time.Second)
	tests := []struct {
		name    string
		args    []*big.Int
		latest  *big.Int
		minimal *big.Int
	}{
		{
			name:    "TestMevGasPricePool Case 1",
			args:    []*big.Int{big.NewInt(5), big.NewInt(4), big.NewInt(3), big.NewInt(2), big.NewInt(1)},
			latest:  big.NewInt(1),
			minimal: big.NewInt(1),
		},
		{
			name:    "TestMevGasPricePool Case 2",
			args:    []*big.Int{big.NewInt(1), big.NewInt(2), big.NewInt(3), big.NewInt(4), big.NewInt(5), big.NewInt(6)},
			latest:  big.NewInt(6),
			minimal: big.NewInt(5),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for i := range tt.args {
				pool.Push(tt.args[i])
				time.Sleep(400 * time.Millisecond)
			}
			if minimal := pool.MinimalGasPrice(); minimal.Uint64() != tt.minimal.Uint64() {
				t.Errorf("MevGasPricePool.MinimalGasPrice() = %d, want: %d", minimal, tt.minimal)
				return
			}
			if latest := pool.LatestGasPrice(); latest.Uint64() != tt.latest.Uint64() {
				t.Errorf("MevGasPricePool.LatestGasPrice() = %d, want: %d", latest, tt.latest)
				return
			}
			pool.Clear()
		})
	}
}
