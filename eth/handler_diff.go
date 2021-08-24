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

package eth

import (
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth/protocols/diff"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

// diffHandler implements the diff.Backend interface to handle the various network
// packets that are sent as replies or broadcasts.
type diffHandler handler

func (h *diffHandler) Chain() *core.BlockChain { return h.chain }

// RunPeer is invoked when a peer joins on the `diff` protocol.
func (h *diffHandler) RunPeer(peer *diff.Peer, hand diff.Handler) error {
	if err := peer.Handshake(h.lightSync); err != nil {
		return err
	}
	return hand(peer)
}

// PeerInfo retrieves all known `diff` information about a peer.
func (h *diffHandler) PeerInfo(id enode.ID) interface{} {
	return nil
}

// Handle is invoked from a peer's message handler when it receives a new remote
// message that the handler couldn't consume and serve itself.
func (h *diffHandler) Handle(peer *diff.Peer, packet diff.Packet) error {
	// TODO implement package handler here
	return nil
}
