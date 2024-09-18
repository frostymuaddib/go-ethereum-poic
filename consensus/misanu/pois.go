package misanu

//MILAN
//
// Transakcije se predstavljaju poljem TxHash, i onda se radi hash(header bez maxDigest +
// rand nonce) < target
//nakon toga se u MaxDigest upise hash a nonce u nonces
//pri proveri se gleda da li je upisan MaxDigest = heshu headera bez nonce i MixDigest
//metodom HashNoNonce().Bytes().
//Header u sebi ima polja MixDigest i nonce. MixDigest mogu da iskoristim z
//
//kako na osnovu broja i hedera naci sam block???
//header ima u sebi TransactionHash

import (
	"errors"
	"math/big"
	"time"

	"github.com/frostymuaddib/go-ethereum-poic/common"
	"github.com/frostymuaddib/go-ethereum-poic/consensus"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
	"github.com/frostymuaddib/go-ethereum-poic/core/types"
	"github.com/frostymuaddib/go-ethereum-poic/log"
	"github.com/frostymuaddib/go-ethereum-poic/rlp"
	"golang.org/x/crypto/sha3"
)

// ////////////////
// Time measuring//
// ////////////////
var blockTime time.Time

//ovde se vrsi inicijalizacija protokola, odnosno pravljenje time/memory tradeoff
//tabele

// Prepare initializes the consensus fields of a block header according to the
// rules of a particular engine. The changes are executed inline.
// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (p *PoIC) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	blockTime = time.Now()
	guicolour.BrightBluePrintf(true, "Почетно време: %s\n", blockTime.String())
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}

	header.Difficulty = p.CalcDifficulty(chain, uint64(header.Time), parent)

	return nil
}

// VerifySeal checks whether the crypto seal on a header is valid according to
// the consensus rules of the given engine.
func (p *PoIC) VerifySeal(chain consensus.ChainHeaderReader, header *types.Header) error {
	guicolour.BrightBluePrintf(true, "Ушао у VerifySeal\n")
	//TODO proverit sta je difficulty!!!
	// Ensure that we have a valid difficulty for the block
	if header.Difficulty.Sign() <= 0 {
		return errInvalidDifficulty
	}

	//broj block-a
	//number := header.Number.Uint64()
	return p.verifyKey(header)
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns the difficulty
// that a new block should have.
// TODO Treba staviti neko vreme
func (p *PoIC) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {

	// myTestDifficulty := new(big.Int).SetUint64(p.config.TableSize)
	// deltaTime := new(big.Int)
	// deltaTime.Sub(new(big.Int).SetUint64(time), parent.Time)
	// if !deltaTime.IsUint64() {
	// 	guicolour.BrightWhitePrintf(true, "t-t_p није UINT64!\n")
	// 	return myTestDifficulty
	// }
	// newDifficulty := (p.config.ExpectedTime * p.config.TableSize)/deltaTime.Uint64()
	// //p.config.
	//
	// //return big.NewInt(20)
	// if newDifficulty> 64 {
	// 	newDifficulty=60
	// }
	// if newDifficulty== 0 {
	// 	newDifficulty=p.config.TableSize
	// }
	// return new(big.Int).SetUint64(newDifficulty)
	return new(big.Int).SetUint64(p.config.TableSize)
}

func (p *PoIC) verifyKey(header *types.Header) error {
	//headerInBytes := header.HashNoNonce().Bytes()
	//calclulatedKey := header.Nonce
	//TODO Ovde ide poziv ka programu
	//headerHash := header.HashNoNonceWithDigest()
	headerHash := HashHeaderWithDigest(header.HashNoNonce(), header.MixDigest)
	//tmp := [8]byte(header.Nonce)
	//key := bytesToInteger(tmp[:])
	key := header.Nonce.Uint64()
	//fmt.Printf("\n\n\n KEY: %v\n", key)
	var error error
	error = nil
	//return p.VerifyEntry(&headerHash, key, p.config.TableSize)
	if !header.Difficulty.IsUint64() {
		return errors.New("Invalid value for difficulty. It is not uint64")
	}
	if p.config.MiniPoW != nil {
		guicolour.BrightWhitePrintf(true, "Провера MiniPoW блока тежине %d\n", p.config.MiniPoW.MiniPowDiff)
		error = verifyMiniPow(p.config.MiniPoW.MiniPowDiff, headerHash)
	} else if p.config.OuterData != nil {
		guicolour.BrightWhitePrintf(true, "Провера спољног извора са временом %d\n", p.config.OuterData.TimeInSecs)
		error = verifyOuterSource(header, p.config.OuterData.TimeInSecs)
	}
	if error != nil {
		return error
	}
	var test uint64
	test = header.Difficulty.Uint64()
	if test == 0 {
		guicolour.BrightWhitePrintf(true, "Тежина блока је нула. Враћа се на %d\n ", p.config.TableSize)
		test = p.config.TableSize
	}
	guicolour.BrightWhitePrintf(true, "Тежина блока при провери: %d\n ", test)
	//OVDE MENJAM ZA SUFFIX
	//return p.VerifyEntry(&headerHash, key, test)
	return p.VerifyEntrySuffix(&headerHash, key, test)

}

// Seal generates a new block for the given input block with the local miner's
// seal place on top.
// samo rudarenje se desava Ovde
func (p *PoIC) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	header := block.Header()

	if p.workCh != nil {
		guicolour.BrightCyanPrintf(true, "UPISUJEM U KANAL\n\n\n")
		p.workCh <- block
	}

	var key *types.BlockNonce
	var externalRez *types.Block
	var err error
	var digestCount int = 0
	var cummulativeCounter uint64
	for {
		digestCount++
		var mixDig *common.Hash
		var error error
		if p.config.MiniPoW != nil {
			//guicolour.BrightWhitePrintf(true, "Одабран мини PoW тежине %d\n", p.config.MiniPoW.MiniPowDiff)
			mixDig, error = mixDigestMiniPoW(p.config.MiniPoW.MiniPowDiff, block, stop)
		} else if p.config.OuterData != nil {
			guicolour.BrightWhitePrintf(true, "Одабран спољни извор са временом %d\n", p.config.OuterData.TimeInSecs)
			mixDig, error = mixDigestFromOuterSource()
		} else {
			guicolour.BrightWhitePrintf(true, "Одабран насумични MixDigest\n")
			mixDig, error = createRandomMixDigest()
		}
		if error != nil {
			return error
		}
		header.MixDigest = *mixDig
		//hash := header.HashNoNonceWithDigest()
		hash := HashHeaderWithDigest(header.HashNoNonce(), header.MixDigest)
		//guicolour.BrightYellowPrintf(true, "\n\n Покреће се претрага табеле за хеш: %x чији је mixDigest (насумичнга вредност) %x",
		//	hash, header.MixDigest)
		// fmt.Printf("Pokrece se pretraga tabele za hash %x i za mixDigest %x.\n", hash, header.MixDigest)
		//mixDigest u sebi ima 32 bita, znaci raditi rand()
		if !block.Difficulty().IsUint64() {
			return errors.New("Invalid value for difficulty. It is not uint64")
		}
		//guicolour.BrightWhitePrintf(true, "Покреће се Seal за тежину %d\n", block.Difficulty().Uint64())
		//key, err = p.FindKeyForHashMultiThread(&hash, p.config.TableSize, stop)
		if block.Difficulty().Uint64() == 0 {
			guicolour.BrightWhitePrintf(true, "Тежина је 0. Поставља се на подразумевано из genesis.json: %d\n", p.config.TableSize)
			key, err, externalRez, cummulativeCounter = p.FindKeyForHashMultiThreadSuffix(&hash, p.config.TableSize, stop, p.resultCh)
		} else {
			//OVDE SE RADI TRENUTNO BEZ TEZINE
			//key, err = p.FindKeyForHashMultiThread(&hash, block.Difficulty().Uint64(), stop)
			// key, err = p.FindKeyForHashMultiThreadWholeTable(&hash, block.Difficulty().Uint64(), stop)
			key, err, externalRez, cummulativeCounter = p.FindKeyForHashMultiThreadSuffix(&hash, block.Difficulty().Uint64(), stop, p.resultCh)
		}

		if err != nil {
			return err
		}
		if key != nil {
			//we have the seal, break for
			break
		} else if externalRez != nil {
			//imamo spoljni rezultat
			guicolour.BrightYellowPrintf(true, "\n\n Нађена спољашња вредност: %v.\n", externalRez)
			guicolour.BrightBluePrintf(true, "Завршно време: %s\n", time.Now().Sub(blockTime).String())
			go func() {
				select {
				case <-stop:
					return
				case results <- externalRez:
				default:
					log.Warn("External sealing result is not read by miner", "sealhash", p.SealHash(header))

				}

			}()
			return nil
		}
	}

	header.Nonce = *key
	printRez := [8]byte(*key)
	guicolour.BrightYellowPrintf(true, "\n\n Нађена nonce вредност: %d. Број покушаја: %d\n", bytesToInteger(printRez[:]), digestCount)
	// err = p.verifyKey(header)
	// guicolour.BrightYellowPrintf(true, "PROVERA: ", err)
	guicolour.BrightBluePrintf(true, "Завршно време: %s\n", time.Now().Sub(blockTime).String())
	guicolour.BrightBluePrintf(true, "Број позива Encoding функције: %v\n", cummulativeCounter)
	//return block.WithSeal(header), nil

	go func() {
		select {
		case <-stop:
			return
		case results <- block.WithSeal(header):
		default:
			log.Warn("Sealing result is not read by miner", "sealhash", p.SealHash(header))

		}

	}()

	return nil

}

func (ethash *PoIC) SealHash(header *types.Header) (hash common.Hash) {
	hasher := sha3.NewLegacyKeccak256()

	enc := []interface{}{
		header.ParentHash,
		header.UncleHash,
		header.Coinbase,
		header.Root,
		header.TxHash,
		header.ReceiptHash,
		header.Bloom,
		header.Difficulty,
		header.Number,
		header.GasLimit,
		header.GasUsed,
		header.Time,
		header.Extra,
	}
	if header.BaseFee != nil {
		enc = append(enc, header.BaseFee)
	}
	if header.WithdrawalsHash != nil {
		panic("withdrawal hash set on ethash")
	}
	rlp.Encode(hasher, enc)
	hasher.Sum(hash[:0])
	return hash
}
