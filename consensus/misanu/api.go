// Copyright 2018 The go-ethereum Authors
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

package misanu

import (
	"errors"

	"github.com/frostymuaddib/go-ethereum-poic/common"
	"github.com/frostymuaddib/go-ethereum-poic/common/hexutil"
	"github.com/frostymuaddib/go-ethereum-poic/consensus"
	"github.com/frostymuaddib/go-ethereum-poic/core/types"
	"github.com/frostymuaddib/go-ethereum-poic/rpc"
)

var errEthashStopped = errors.New("ethash stopped")

// API exposes ethash related methods for the RPC interface.
type API struct {
	p *PoIC // Make sure the mode of ethash is normal.
}

// GetWork returns a work package for external miner.
//
// The work package consists of 3 strings:
//
//	result[0] - 32 bytes hex encoded current block header pow-hash
//	result[1] - 32 bytes hex encoded seed hash used for DAG
//	result[2] - 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
func (api *API) GetWork() ([3]string, error) {
	// if api.ethash.config.PowMode != ModeNormal && api.ethash.config.PowMode != ModeTest {
	// 	return [3]string{}, errors.New("not supported")
	// }

	var (
		workCh = make(chan [3]string, 1)
		errc   = make(chan error, 1)
	)

	select {
	case api.p.fetchWorkCh <- &sealWork{errc: errc, res: workCh}:
	case <-api.p.exitCh:
		return [3]string{}, errEthashStopped
	}

	select {
	case work := <-workCh:
		return work, nil
	case err := <-errc:
		return [3]string{}, err
	}
}

// SubmitWork can be used by external miner to submit their POW solution.
// It returns an indication if the work was accepted.
// Note either an invalid solution, a stale work a non-existent work will return false.
func (api *API) SubmitWork(nonce types.BlockNonce, hash, digest common.Hash) bool {
	// if api.p.config.PowMode != ModeNormal && api.p.config.PowMode != ModeTest {
	// 	return false
	// }

	var errc = make(chan error, 1)

	select {
	case api.p.submitWorkCh <- &mineResult{
		nonce:     nonce,
		mixDigest: digest,
		hash:      hash,
		errc:      errc,
	}:
	case <-api.p.exitCh:
		return false
	}

	err := <-errc
	return err == nil
}

// SubmitHashrate can be used for remote miners to submit their hash rate.
// This enables the node to report the combined hash rate of all miners
// which submit work through this node.
//
// It accepts the miner hash rate and an identifier which must be unique
// between nodes.
func (api *API) SubmitHashRate(rate hexutil.Uint64, id common.Hash) bool {
	// if api.ethash.config.PowMode != ModeNormal && api.ethash.config.PowMode != ModeTest {
	// 	return false
	// }

	var done = make(chan struct{}, 1)

	select {
	case api.p.submitRateCh <- &hashrate{done: done, rate: uint64(rate), id: id}:
	case <-api.p.exitCh:
		return false
	}

	// Block until hash rate submitted successfully.
	<-done

	return true
}

// GetHashrate returns the current hashrate for local CPU miner and remote miner.
func (api *API) GetHashrate() uint64 {
	return uint64(1234)
}

// APIs implements consensus.Engine, returning the user facing RPC APIs.
func (p *PoIC) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	// In order to ensure backward compatibility, we exposes ethash RPC APIs
	// to both eth and ethash namespaces.
	return []rpc.API{
		{
			Namespace: "eth",
			Version:   "1.0",
			Service:   &API{p},
			Public:    true,
		},
		{
			Namespace: "ethash",
			Version:   "1.0",
			Service:   &API{p},
			Public:    true,
		},
	}
}
