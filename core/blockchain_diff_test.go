// Copyright 2020 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Tests that abnormal program termination (i.e.crash) and restart doesn't leave
// the database in some strange state with gaps in the chain, nor with block data
// dangling in the future.

package core

import (
	"math/big"
	"testing"

	"golang.org/x/crypto/sha3"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	// testKey is a private key to use for funding a tester account.
	testKey, _ = crypto.HexToECDSA("b71c71a67e1177ad4e901695e1b4b9ee17ae16c6668d313eac2f96dbcda3f291")
	// testAddr is the Ethereum address of the tester account.
	testAddr = crypto.PubkeyToAddress(testKey.PublicKey)
)

// testBackend is a mock implementation of the live Ethereum message handler. Its
// purpose is to allow testing the request/reply workflows and wire serialization
// in the `eth` protocol without actually doing any data processing.
type testBackend struct {
	db    ethdb.Database
	chain *BlockChain
}

// newTestBackend creates an empty chain and wraps it into a mock backend.
func newTestBackend(blocks int, light bool) *testBackend {
	return newTestBackendWithGenerator(blocks, light)
}

// newTestBackend creates a chain with a number of explicitly defined blocks and
// wraps it into a mock backend.
func newTestBackendWithGenerator(blocks int, lightProcess bool) *testBackend {
	signer := types.HomesteadSigner{}
	// Create a database pre-initialize with a genesis block
	db := rawdb.NewMemoryDatabase()
	(&Genesis{
		Config: params.TestChainConfig,
		Alloc:  GenesisAlloc{testAddr: {Balance: big.NewInt(100000000000000000)}},
	}).MustCommit(db)

	chain, _ := NewBlockChain(db, nil, params.TestChainConfig, ethash.NewFaker(), vm.Config{}, nil, nil)
	generator := func(i int, block *BlockGen) {
		// The chain maker doesn't have access to a chain, so the difficulty will be
		// lets unset (nil). Set it here to the correct value.
		block.SetCoinbase(testAddr)

		// We want to simulate an empty middle block, having the same state as the
		// first one. The last is needs a state change again to force a reorg.
		tx, err := types.SignTx(types.NewTransaction(block.TxNonce(testAddr), common.Address{0x01}, big.NewInt(1), params.TxGas, big.NewInt(1), nil), signer, testKey)
		if err != nil {
			panic(err)
		}
		block.AddTxWithChain(chain, tx)
	}
	bs, _ := GenerateChain(params.TestChainConfig, chain.Genesis(), ethash.NewFaker(), db, blocks, generator)
	if _, err := chain.InsertChain(bs); err != nil {
		panic(err)
	}
	if lightProcess {
		EnableLightProcessor(chain)
	}

	return &testBackend{
		db:    db,
		chain: chain,
	}
}

// close tears down the transaction pool and chain behind the mock backend.
func (b *testBackend) close() {
	b.chain.Stop()
}

func (b *testBackend) Chain() *BlockChain { return b.chain }

func TestHandleDiffLayer(t *testing.T) {
	t.Parallel()

	blockNum := maxDiffLimit - 1
	fullBackend := newTestBackend(blockNum, false)
	defer fullBackend.close()

	lightBackend := newTestBackend(0, true)
	for i := 1; i <= blockNum; i++ {
		block := fullBackend.chain.GetBlockByNumber(uint64(i))
		if block == nil {
			t.Fatal("block should not be nil")
		}
		blockHash := block.Hash()
		rawDiff := fullBackend.chain.GetDiffLayerRLP(blockHash)
		var diff types.DiffLayer
		hasher := sha3.NewLegacyKeccak256()
		err := rlp.DecodeBytes(rawDiff, &diff)
		if err != nil {
			t.Fatal("decode raw data failed")
		}
		hasher.Write(rawDiff)
		var diffHash common.Hash
		hasher.Sum(diffHash[:0])
		hasher.Reset()
		diff.DiffHash = diffHash
		lightBackend.Chain().HandleDiffLayer(&diff, "testpid")
		_, err = lightBackend.chain.insertChain([]*types.Block{block}, true)
		if err != nil {
			t.Errorf("failed to insert block %v", err)
		}
	}
}
