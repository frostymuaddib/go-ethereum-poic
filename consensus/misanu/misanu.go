//Ovde pocinje kod za nas algoritam

package misanu

import (
	// "bytes"
	// "math/rand"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"sync"
	"time"

	mapset "github.com/deckarep/golang-set"

	// "github.com/frostymuaddib/go-ethereum-poic/accounts"
	"github.com/frostymuaddib/go-ethereum-poic/common"
	"github.com/frostymuaddib/go-ethereum-poic/trie"
"github.com/frostymuaddib/go-ethereum-poic/core/tracing"
	// "github.com/frostymuaddib/go-ethereum-poic/common/hexutil"
	// "github.com/frostymuaddib/go-ethereum-poic/consensus"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misc"
	"github.com/frostymuaddib/go-ethereum-poic/core/state"
	"github.com/frostymuaddib/go-ethereum-poic/core/types"

	// "github.com/frostymuaddib/go-ethereum-poic/crypto"
	// "github.com/frostymuaddib/go-ethereum-poic/crypto/sha3"
	"github.com/frostymuaddib/go-ethereum-poic/ethdb"
	"github.com/frostymuaddib/go-ethereum-poic/log"
	"github.com/frostymuaddib/go-ethereum-poic/params"

	// "github.com/frostymuaddib/go-ethereum-poic/rlp"

	// lru "github.com/hashicorp/golang-lru"
	//"github.com/frostymuaddib/go-ethereum-poic/core/state"

	"github.com/frostymuaddib/go-ethereum-poic/consensus"
	"github.com/holiman/uint256"
)

//TODO: PROVERITI OVO
// Ethash proof-of-work protocol constants.
var (
	FrontierBlockReward    *big.Int = big.NewInt(5e+18) // Block reward in wei for successfully mining a block
	ByzantiumBlockReward   *big.Int = big.NewInt(3e+18) // Block reward in wei for successfully mining a block upward from Byzantium
	maxUncles                       = 2                 // Maximum number of uncles allowed in a single block
	allowedFutureBlockTime          = 15 * time.Second  // Max time from current time allowed for blocks, before they're considered future blocks
)

var (
	errLargeBlockTime    = errors.New("timestamp too big")
	errZeroBlockTime     = errors.New("timestamp equals parent's")
	errTooManyUncles     = errors.New("too many uncles")
	errDuplicateUncle    = errors.New("duplicate uncle")
	errUncleIsAncestor   = errors.New("uncle is ancestor")
	errDanglingUncle     = errors.New("uncle's parent is not ancestor")
	errInvalidDifficulty = errors.New("non-positive difficulty")
	errInvalidMixDigest  = errors.New("invalid mix digest")
	errInvalidPoW        = errors.New("invalid proof-of-work")
)

// sealWork wraps a seal work package for remote sealer.
type sealWork struct {
	errc chan error
	res  chan [3]string
}

type mineResult struct {
	nonce     types.BlockNonce
	mixDigest common.Hash
	hash      common.Hash

	errc chan error
}

// hashrate wraps the hash rate submitted by the remote sealer.
type hashrate struct {
	id   common.Hash
	ping time.Time
	rate uint64

	done chan struct{}
}

type PoIC struct {
	config *params.PoICConfig
	// da li ce mi ovo trebati?
	db ethdb.Database

	signer common.Address // Ethereum address of the signing key
	//signFn SignerFn       // Signer function to authorize hashes with
	lock sync.RWMutex // Protects the signer fields
	//this is PoIC trade-off table
	table                *TradeoffTable
	tablePartiallySorted *TradeoffTable
	tableMutex           sync.RWMutex

	//this is for many partial tables
	tableSet *TradeoffTableSet

	// Remote sealer related fields
	workCh       chan *types.Block // Notification channel to push new work to remote sealer
	resultCh     chan *types.Block // Channel used by mining threads to return result
	fetchWorkCh  chan *sealWork    // Channel used for remote sealer to fetch mining work
	submitWorkCh chan *mineResult  // Channel used for remote sealer to submit their mining result
	fetchRateCh  chan chan uint64  // Channel used to gather submitted hash rate for local or remote sealer.
	submitRateCh chan *hashrate    // Channel used for remote sealer to submit their mining hashrate
	exitCh       chan chan error   // Notification channel to exiting backend threads
}

//Funkcije neophodne da se algoritam smatra konsenzusom

// Author retrieves the Ethereum address of the account that minted the given
// block, which may be different from the header's coinbase if a consensus
// engine is based on signatures.
func (p *PoIC) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of a
// given engine.
func (p *PoIC) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Short circuit if the header is known, or it's parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return p.verifyHeader(chain, header, parent, false, true)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications (the order is that of
// the input slice).
func (p *PoIC) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header) (chan<- struct{}, <-chan error) {
	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = p.verifyHeaderWorker(chain, headers, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut

}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of a given engine.
func (p *PoIC) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// Verify that there are at most 2 uncles included in this block
	if len(block.Uncles()) > maxUncles {
		return errTooManyUncles
	}
	// Gather the set of past uncles and ancestors
	uncles, ancestors := mapset.NewSet(), make(map[common.Hash]*types.Header)
	//MILAN ovde pokupimo sedam prethodnih blokova
	number, parent := block.NumberU64()-1, block.ParentHash()
	for i := 0; i < 7; i++ {
		ancestor := chain.GetBlock(parent, number)
		if ancestor == nil {
			break
		}
		ancestors[ancestor.Hash()] = ancestor.Header()
		for _, uncle := range ancestor.Uncles() {
			uncles.Add(uncle.Hash())
		}
		parent, number = ancestor.ParentHash(), number-1
	}
	ancestors[block.Hash()] = block.Header()
	uncles.Add(block.Hash())

	// Verify each of the uncles that it's recent, but not an ancestor
	for _, uncle := range block.Uncles() {
		// Make sure every uncle is rewarded only once
		hash := uncle.Hash()
		if uncles.Contains(hash) {
			return errDuplicateUncle
		}
		uncles.Add(hash)

		// Make sure the uncle has a valid ancestry
		if ancestors[hash] != nil {
			return errUncleIsAncestor
		}
		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == block.ParentHash() {
			return errDanglingUncle
		}
		if err := p.verifyHeader(chain, uncle, ancestors[uncle.ParentHash], true, true); err != nil {
			return err
		}
	}
	return nil
}

// Finalize runs any post-transaction state modifications (e.g. block rewards)
// and assembles the final block.
// Note: The block header and state database might be updated to reflect any
// consensus rules that happen at finalization (e.g. block rewards).
// Finalize implements consensus.Engine, accumulating the block and uncle rewards,
// setting the final state and assembling the block.
// func (p *PoIC) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction,
// 	uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {

// 	// Accumulate any block and uncle rewards and commit the final state root
// 	accumulateRewards(chain.Config(), state, header, uncles)
// 	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

// 	// Header seems complete, assemble into a block and return
// 	return types.NewBlock(header, txs, uncles, receipts), nil
//}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards.
//func (p *PoIC) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, withdrawals []*types.Withdrawal) {
func (p *PoIC)	Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body) {
	// Accumulate any block and uncle rewards
	accumulateRewards(chain.Config(), state, header, body.Uncles)
}

// FinalizeAndAssemble implements consensus.Engine, accumulating the block and
// uncle rewards, setting the final state and assembling the block.
//func (p *PoIC) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt, withdrawals []*types.Withdrawal) (*types.Block, error) {
func (p *PoIC) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, body *types.Body, receipts []*types.Receipt) (*types.Block, error) {
	// if len(withdrawals) > 0 {
	// 	return nil, errors.New("PoIC does not support withdrawals")
	// }
	// Finalize block
	p.Finalize(chain, header, state, body)

	// Assign the final state root to header.
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// Header seems complete, assemble into a block and return
	//body := types.Body{Transactions: txs, Uncles: uncles, Withdrawals: nil}
	return types.NewBlock(header, body, receipts, trie.NewStackTrie(nil)), nil
}

// APIs returns the RPC APIs this consensus engine provides.
// func (p *PoIC) APIs(chain consensus.ChainReader) []rpc.API {
// 	var dummy []rpc.API
// 	return dummy
//}

// Close terminates any background threads maintained by the consensus engine.
func (p *PoIC) Close() error {
	return p.ClearTable()
	return nil
}

//pomocne Funkcije
func (p *PoIC) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header, uncle bool, seal bool) error {
	//proverava da li extra-data deo zaglavlja bloka jeste odgovarajuce velicine
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	//proverava vremensku oznaku blokova
	if uncle {
		// if header.Time > math.MaxBig256 {
		// 	return errLargeBlockTime
		// }
	} else {
		if header.Time > uint64(time.Now().Add(allowedFutureBlockTime).Unix()) {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time <= parent.Time {
		return errZeroBlockTime
	}

	// Verify the block's difficulty based in it's timestamp and parent's difficulty
	expected := p.CalcDifficulty(chain, uint64(header.Time), parent)
	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
	}

	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}

	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor

	if uint64(diff) >= limit || header.GasLimit < params.MinGasLimit {
		return fmt.Errorf("invalid gas limit: have %d, want %d += %d", header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the engine specific seal securing the block
	if seal {
		if err := p.VerifySeal(chain, header); err != nil {
			return err
		}
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyDAOHeaderExtraData(chain.Config(), header); err != nil {
		return err
	}
	//Ovo je izbaceno u novoj verziji
	//if err := misc.VerifyForkHashes(chain.Config(), header, uncle); err != nil {
	//	return err
	//}
	return nil
}

func (p *PoIC) verifyHeaderWorker(chain consensus.ChainHeaderReader, headers []*types.Header, index int) error {
	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if chain.GetHeader(headers[index].Hash(), headers[index].Number.Uint64()) != nil {
		return nil // known block
	}
	return p.verifyHeader(chain, headers[index], parent, false, true)
}

// Some weird constants to avoid constant memory allocs for them.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {
	// Select the correct block reward based on chain progression
	blockReward := FrontierBlockReward
	if config.IsByzantium(header.Number) {
		blockReward = ByzantiumBlockReward
	}
	// Accumulate the rewards for the miner and any included uncles
	reward := new(big.Int).Set(blockReward)
	r := new(big.Int)
	for _, uncle := range uncles {
		r.Add(uncle.Number, big8)
		r.Sub(r, header.Number)
		r.Mul(r, blockReward)
		r.Div(r, big8)
		uint256_r, _:= uint256.FromBig(r)
		state.AddBalance(uncle.Coinbase, uint256_r, tracing.BalanceIncreaseRewardMineUncle)

		r.Div(blockReward, big32)
		reward.Add(reward, r)
	}
	uint256_reward, _:= uint256.FromBig(reward)
	state.AddBalance(header.Coinbase, uint256_reward, tracing.BalanceIncreaseRewardMineBlock) 
}

// New creates a PoIC
func New(config *params.PoICConfig, db ethdb.Database) *PoIC {
	// Set any missing consensus parameters to their defaults
	// conf := *config
	// if conf.Epoch == 0 {
	// 	conf.Epoch = epochLength
	// }
	// Allocate the snapshot caches and create the engine
	// recents, _ := lru.NewARC(inmemorySnapshots)
	// signatures, _ := lru.NewARC(inmemorySignatures)

	// return &Clique{
	// 	config:     &conf,
	// 	db:         db,
	// 	recents:    recents,
	// 	signatures: signatures,
	// 	proposals:  make(map[common.Address]bool),
	// }
	conf := *config

	p := &PoIC{
		config:       &conf,
		db:           db,
		workCh:       make(chan *types.Block),
		resultCh:     make(chan *types.Block),
		fetchWorkCh:  make(chan *sealWork),
		submitWorkCh: make(chan *mineResult),
		fetchRateCh:  make(chan chan uint64),
		submitRateCh: make(chan *hashrate),
		exitCh:       make(chan chan error),
	}

	// err := p.CreateTableBinary(p.config.TableSize)
	err := p.CreateTableBinarySuffix(p.config.TableSize)
	if err != nil {
		return nil
	}

	// err = p.CreateTableSet(p.config.TableSize /*numberOfTables*/, 8)
	// if err != nil {
	// 	return nil
	// }
	go p.remote()
	return p

}

var (
	errNoMiningWork      = errors.New("no mining work available yet")
	errInvalidSealResult = errors.New("invalid or stale proof-of-work solution")
)

//deo za komunikaciju sa pool managerom i rudarima
func (p *PoIC) remote() {
	var (
		works       = make(map[common.Hash]*types.Block)
		rates       = make(map[common.Hash]hashrate)
		currentWork *types.Block
	)
	//TODO prepraviti da vraca header, minipow diff i difficulty
	// getWork returns a work package for external miner.
	//
	// The work package consists of 3 strings:
	//   result[0], 32 bytes hex encoded current block header pow-hash
	//   result[1], 32 bytes hex encoded seed hash used for DAG
	//   result[2], 32 bytes hex encoded boundary condition ("target"), 2^256/difficulty
	getWork := func() ([3]string, error) {
		var res [3]string
		if currentWork == nil {
			return res, errNoMiningWork
		}
		res[0] = currentWork.Header().HashNoNonce().Hex()
		tmp := big.NewInt(2)
		res[1] = common.BytesToHash(tmp.Bytes()).Hex()

		// Calculate the "target" to be returned to the external sealer.
		n := big.NewInt(1)
		res[2] = common.BytesToHash(n.Bytes()).Hex()

		// Trace the seal work fetched by remote sealer.
		works[currentWork.Header().HashNoNonce()] = currentWork

		return res, nil
	}

	// submitWork verifies the submitted pow solution, returning
	// whether the solution was accepted or not (not can be both a bad pow as well as
	// any other error, like no pending work or stale mining result).
	submitWork := func(nonce types.BlockNonce, mixDigest common.Hash, hash common.Hash) bool {
		// Make sure the work submitted is present

		block := works[hash]
		if block == nil {
			log.Info("Work submitted but none pending", "hash", hash)
			return false
		}

		//fmt.Printf("MILAN: %v\n", *block)
		// Verify the correctness of submitted result.
		header := block.Header()
		header.Nonce = nonce
		header.MixDigest = mixDigest
		//fmt.Printf("AASdasdfasdasd1\n %v %v \n", header.Nonce, header.MixDigest)
		//fmt.Printf("AASdasdfasdasd2\n %v %v \n", nonce, mixDigest)
		if err := p.VerifySeal(nil, header); err != nil {
			log.Warn("Invalid proof-of-work submitted", "hash", hash, "err", err)
			return false
		}

		// Make sure the result channel is created.
		if p.resultCh == nil {
			log.Warn("Ethash result channel is empty, submitted mining result is rejected")
			return false
		}
		//fmt.Printf("AAAAAAAAAAAAAAAAAAAAAAAAa\n\n\n")
		// Solutions seems to be valid, return to the miner and notify acceptance.

		p.resultCh <- block.WithSeal(header)
		delete(works, hash)
		return true

		/*select {
		case p.resultCh <- block.WithSeal(header):
			delete(works, hash)
			return true
		default:
			log.Info("Work submitted is stale", "hash", hash)
			return false
		}*/
	}

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case block := <-p.workCh:
			if currentWork != nil && block.ParentHash() != currentWork.ParentHash() {
				// Start new round mining, throw out all previous work.
				works = make(map[common.Hash]*types.Block)
			}
			// Update current work with new received block.
			// Note same work can be past twice, happens when changing CPU threads.
			currentWork = block

		case work := <-p.fetchWorkCh:
			// Return current mining work to remote miner.
			miningWork, err := getWork()
			if err != nil {
				work.errc <- err
			} else {
				work.res <- miningWork
			}

		case result := <-p.submitWorkCh:
			// Verify submitted PoW solution based on maintained mining blocks.
			//fmt.Printf("\n\n\n %v\n%v\n%v\n\n\n", result.nonce.Uint64(), result.mixDigest.Hex(), result.hash.Hex())
			if submitWork(result.nonce, result.mixDigest, result.hash) {
				result.errc <- nil
			} else {
				result.errc <- errInvalidSealResult
			}

		case result := <-p.submitRateCh:
			// Trace remote sealer's hash rate by submitted value.
			rates[result.id] = hashrate{rate: result.rate, ping: time.Now()}
			close(result.done)

		case req := <-p.fetchRateCh:
			// Gather all hash rate submitted by remote sealer.
			var total uint64
			for _, rate := range rates {
				// this could overflow
				total += rate.rate
			}
			req <- total

		case <-ticker.C:
			// Clear stale submitted hash rate.
			for id, rate := range rates {
				if time.Since(rate.ping) > 10*time.Second {
					delete(rates, id)
				}
			}

		case errc := <-p.exitCh:
			// Exit remote loop if ethash is closed and return relevant error.
			errc <- nil
			log.Trace("Ethash remote sealer is exiting")
			return
		}
	}
}
