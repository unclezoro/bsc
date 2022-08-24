package core

import (
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
)

// NewMevGasPricePool creates a new MevGasPricePool.
func NewMevGasPricePool(expire time.Duration) *MevGasPricePool {
	return &MevGasPricePool{
		expire: expire,
		queue:  prque.New(nil),
		latest: common.Big0,
	}
}

// MevGasPricePool is a limited number of queues.
// In order to avoid too drastic gas price changes, the latest n gas prices are cached.
// Allowed as long as the user's Gas Price matches this range.
type MevGasPricePool struct {
	mu     sync.RWMutex
	expire time.Duration
	queue  *prque.Prque
	latest *big.Int
}

type gasPriceInfo struct {
	val  *big.Int
	time time.Time
}

// Push is a method to cache a new gas price.
func (pool *MevGasPricePool) Push(gasPrice *big.Int) {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.retire()
	index := -gasPrice.Int64()
	pool.queue.Push(&gasPriceInfo{val: gasPrice, time: time.Now()}, index)
	pool.latest = gasPrice
}

func (pool *MevGasPricePool) retire() {
	now := time.Now()
	for !pool.queue.Empty() {
		v, _ := pool.queue.Peek()
		info := v.(*gasPriceInfo)
		if info.time.Add(pool.expire).After(now) {
			break
		}
		pool.queue.Pop()
	}
}

// LatestGasPrice is a method to get latest cached gas price.
func (pool *MevGasPricePool) LatestGasPrice() *big.Int {
	pool.mu.RLock()
	defer pool.mu.RUnlock()
	return pool.latest
}

// MinimalGasPrice is a method to get minimal cached gas price.
func (pool *MevGasPricePool) MinimalGasPrice() *big.Int {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	if pool.queue.Empty() {
		return common.Big0
	}
	pool.retire()
	v, _ := pool.queue.Peek()
	gasPriceInfo := v.(*gasPriceInfo)
	return gasPriceInfo.val
}

// Clear is a method to clear all caches.
func (pool *MevGasPricePool) Clear() {
	pool.mu.Lock()
	defer pool.mu.Unlock()
	pool.queue.Reset()
}
