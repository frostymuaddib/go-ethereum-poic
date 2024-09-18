package misanu

//this will contain functions for obtaining some data from outer source (e.g. stock
//exchange) used to create mix digest, along with function for verifying that data
//during block validation

import (
	"errors"
	"runtime"
	"sync"

	"github.com/frostymuaddib/go-ethereum-poic/common"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
	"github.com/frostymuaddib/go-ethereum-poic/core/types"
)

var POW_MASKS = [...]byte{0x0, 0x80, 0xC0, 0xE0, 0xF0, 0xF8, 0xFC, 0xFE, 0xFF}

func startsWithNumOfZeroes(value []byte, numOfMiniBits int) bool {
	iter := numOfMiniBits / 8
	var i int
	for i = 0; i < iter; i++ {
		if value[i]&POW_MASKS[8] != 0 { //all 8 bits must be zero
			return false
		}
	}
	numOfMiniBits -= 8 * iter
	return (value[i] & POW_MASKS[numOfMiniBits]) == 0
}

func verifyMiniPow(numOfMiniBits int, headerHash common.Hash) error {
	if startsWithNumOfZeroes(headerHash.Bytes(), numOfMiniBits) {
		return nil
	} else {
		return errors.New("Invalid mini-pow value.")
	}
}

//we are looking a random value that will be part of mix digest that will make hash
//of header with MixDigest begin with specific number of zeroes (predefined)
func mixDigestMiniPoW(numOfMiniBits int, block *types.Block, stop <-chan struct{}) (*common.Hash, error) {
	var pending sync.WaitGroup
	cpuNum := runtime.NumCPU()
	abort := make(chan struct{})
	found := make(chan *common.Hash)
	for i := 0; i < cpuNum; i++ {
		pending.Add(1)
		go func(threadId int, numOfMiniBits int, block *types.Block, abort chan struct{}, found chan *common.Hash) {
			defer pending.Done()
			miniPowMiner(threadId, numOfMiniBits, block, abort, found)
		}(i, numOfMiniBits, block, abort, found)

	}
	var result *common.Hash
	for {
		select {
		case <-stop:
			close(abort)
			return nil, errors.New("ABORTED")
		case result = <-found:
			close(abort)
			return result, nil
		default:

		}

	}
}

func miniPowMiner(threadId int, numOfMiniBits int, block *types.Block, abort chan struct{}, found chan *common.Hash) {
	header := block.Header() //this is copy of header - safe for modifying - see core/types/block.go
	for {
		random, err := createRandomMixDigest()
		if err != nil {
			guicolour.BrightRedPrintf(true, "Error while generating random in miniPowMiner")
			return
		}
		header.MixDigest = *random
		select {
		case <-abort:
			return
		default:
			//hash := header.HashNoNonceWithDigest()
			hash := HashHeaderWithDigest(header.HashNoNonce(), header.MixDigest)
			if startsWithNumOfZeroes(hash.Bytes(), numOfMiniBits) {
				select {
				case <-abort:
					return
				default:
					found <- random
					//guicolour.BrightCyanPrintf(true, "Мали PoW нашао хеш.\n")
					return
				}
			}
		}

	}

}

func HashHeaderWithDigest(headerNoNonce common.Hash, mixDigest common.Hash) common.Hash {
	return types.RlpHash([]interface{}{
		headerNoNonce,
		mixDigest,
	})
}
