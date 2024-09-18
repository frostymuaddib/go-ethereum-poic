package misanu

//u ovoj datoteci ce stajati sve stvari vezane za time memory tradeoff
//kao i sve verzije istog, ukljucujuci grain i sl.

import (
	"errors"
	"runtime"
	"sort"
	"sync"

	"github.com/frostymuaddib/go-ethereum-poic/common"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"

	"github.com/frostymuaddib/go-ethereum-poic/core/types"
)

func (p *PoIC) getInterStateBlockEncoderSuffix(threadId int, pocetno uint64, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, ccMtx *sync.Mutex, callCounter *uint64) (*uint64, error) {
	var ret *uint64
	mod := uint64(1 << numOfBits)
	for i := uint64(0); i < p.table.tableWidth; i++ {
		select {
		case <-abort:
			return nil, errors.New("ABORTED")
		default:
			novo := fillTo56WithOnes(pocetno, numOfBits)
			novo = Convert56to64Des(novo)
			novo = encoder.EncodingFunction(novo, numOfBits)
			ccMtx.Lock()
			*callCounter++
			ccMtx.Unlock()
			if novo%mod == value%mod {
				ret = new(uint64)
				*ret = pocetno
				return ret, nil
			} else {
				pocetno = novo
			}
		}
	}
	return nil, nil
}

func (p *PoIC) getXforGivenYSuffix(value uint64, threadID int, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, rez chan tableRows) {
	var ret tableRows
	p.tableMutex.RLock()
	defer p.tableMutex.RUnlock()
	target := value % (1 << numOfBits)
	select {
	case <-abort:
		ret.Xs = nil
		ret.Err = errors.New("ABORT")
		rez <- ret
		return
	default:
		resultRow := sort.Search(int(endPos-startPos), func(i int) bool {
			if p.table.tableMap == nil {
				return false
			}
			result := toUint64(p.table.tableMap[startPos+uint64(i)*16+8 : startPos+uint64(i)*16+16])
			return result >= target
		})
		resRow := uint64(resultRow) + startPos

		if resultRow < int(endPos-startPos) && toUint64(p.table.tableMap[resRow*16+8:resRow*16+16]) == target {
			//row found, check if there are others
			x := toUint64(p.table.tableMap[resRow*16 : resRow*16+8])
			y := toUint64(p.table.tableMap[resRow*16+8 : resRow*16+16])
			ret.Xs = append(ret.Xs, x)
			for i := uint64(1); i < p.table.tableHeight-uint64(resultRow) && toUint64(p.table.tableMap[resRow*16+i*16+8:resRow*16+i*16+16]) == y; i++ {
				select {
				case <-abort:
					ret.Xs = nil
					ret.Err = errors.New("ABORT")
					rez <- ret
					return
				default:
					ret.Xs = append(ret.Xs, toUint64(p.table.tableMap[resRow*16+i*16:resRow*16+8+i*16]))
				}
			}
			select {
			case <-abort:
				ret.Err = errors.New("ABORT")
				rez <- ret
				return
			default:
				rez <- ret
				return
			}

		} else {
			ret.Xs = nil
			ret.Err = nil
			rez <- ret
			return
		}
	}

}

func (p *PoIC) locateRowSuffix(threadId int, value uint64, iteration uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, ccMtx *sync.Mutex, callCounter *uint64) ([]uint64, uint64, uint64, error) {
	var i uint64
	for i = iteration; i < p.table.tableWidth; i++ {
		select {
		case <-abort:
			return nil, 0, 0, errors.New("ABORT")
		default:
			rez := make(chan tableRows)
			go p.getXforGivenYSuffix(value, threadId, startPos, endPos, numOfBits, abort, rez)
			rows := <-rez
			if rows.Err != nil {
				return nil, 0, 0, rows.Err
			}
			if len(rows.Xs) != 0 {
				return rows.Xs, value, i, nil
			}
			value = fillTo56WithOnes(value, numOfBits)
			value = Convert56to64Des(value)
			value = encoder.EncodingFunction(value, numOfBits)
			ccMtx.Lock()
			*callCounter++
			ccMtx.Unlock()
		}

	}
	return nil, value, i, nil
}

func (p *PoIC) minerSuffix(threadId int, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, found chan *types.BlockNonce, mtx *sync.Mutex, cummulativeCounter *uint64) {
	var rows []uint64
	var rez *uint64
	var err error
	var callCounter uint64 = 0
	var ccMtx sync.Mutex
	select {
	case <-abort:
		return
	default:
		value = value % (1 << numOfBits)

		currentValue := value
		currentIteration := uint64(0)
		for currentIteration < p.table.tableWidth {
			var e error
			rows, currentValue, currentIteration, e = p.locateRowSuffix(threadId, currentValue, currentIteration, startPos, endPos, numOfBits, abort, &ccMtx, &callCounter)
			if e != nil {
				//aborted returns
				return
			}
			for i := 0; i < len(rows); i++ {

				//MILAN ZA TESTIRANJE OVO!
				rez, err = p.getInterStateBlockEncoderSuffix(threadId, rows[i], value, startPos, endPos, numOfBits, abort, &ccMtx, &callCounter)
				if err != nil {
					//function aborted, return
					return
				}
				if rez != nil {
					ret, _ := bytesToNonce(integerToBytes(*rez))
					select {
					case found <- ret:
						guicolour.BrightCyanPrintf(true, "\n\n Нит %d нашла вредност %d\n", threadId, *rez)
					case <-abort:
					}
					return
				}

			}
			currentIteration++
			currentValue = fillTo56WithOnes(currentValue, numOfBits)
			currentValue = Convert56to64Des(currentValue)
			currentValue = encoder.EncodingFunction(currentValue, numOfBits)
			ccMtx.Lock()
			callCounter++
			ccMtx.Unlock()
		}

	}
	mtx.Lock()
	*cummulativeCounter += callCounter
	mtx.Unlock()
	for {

		select {
		case found <- nil:
			return
		case <-abort:
			return
		default:
		}
	}
}

func (p *PoIC) FindKeyForHashMultiThreadSuffix(hashOfHeader *common.Hash, numOfBits uint64, stop <-chan struct{}, foundExternal chan *types.Block) (*types.BlockNonce, error, *types.Block, uint64) {
	var value uint64 = bytesToInteger(hashOfHeader[:])
	if numOfBits == 0 {
		//this is error, we do not do %1<<0
		return nil, errors.New("Invalid numbOfBits. Must be greater than 0."), nil, 0
	}
	abort := make(chan struct{})
	found := make(chan *types.BlockNonce)
	var pending sync.WaitGroup
	cpuNum := runtime.NumCPU()
	var step uint64
	step = p.table.tableHeight / uint64(cpuNum)

	var cummulativeCounter uint64 = 0
	var mtx sync.Mutex
	for i := 0; i < cpuNum; i++ {
		pending.Add(1)
		go func(threadId int, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, found chan *types.BlockNonce) {
			defer pending.Done()
			p.minerSuffix(threadId, value, startPos, endPos, numOfBits, abort, found, &mtx, &cummulativeCounter)
		}(i, value, uint64(i)*step, uint64((i+1))*step, numOfBits, abort, found)
	}
	select {
	case <-stop:
		close(abort)
		return nil, errors.New("ABORTED"), nil, 0
	default:

		var result *types.BlockNonce = nil
		var resultExternal *types.Block = nil
	rezultati:
		for {
			select {
			case resultExternal = <-foundExternal:
				//rezultat je nadjen spoljnim rudarom, prekinuti sve ostale rudare
				if resultExternal != nil {
					guicolour.BrightBluePrintf(true, "\n\n\n Спољни рудар нашао резултат %v! \n", *resultExternal)
					// prekini sve niti!
					close(abort)
					break rezultati
				}
			case result = <-found:
				// One of the threads found a block, abort all others
				cpuNum--
				if result != nil {
					guicolour.BrightCyanPrintf(true, "\n\n Нит нашла рез %d!\n", result)
					close(abort)
					break rezultati
				} else {
					if cpuNum == 0 {
						//no results
						break rezultati
					}
				}
			}

		}
		pending.Wait()
		// Wait for all miners to terminate and return the block
		if result == nil && resultExternal == nil {
			//ovo sada nije greska, krenuti ponovo
			//fmt.Println("Key with current digest not found. Retry.")
			return nil, nil, nil, 0
			//errors.New("Error: key not found!")
		} else if result != nil {
			guicolour.BrightRedPrintf(true, "НАЂЕНО ЗА %d ВРЕДНОСТ %d", value, result)
			return result, nil, nil, cummulativeCounter
		} else {
			guicolour.BrightRedPrintf(true, "НАЂЕНО СПОЉА ЗА %v", resultExternal)
			return nil, nil, resultExternal, 0
		}
	}
}

func (p *PoIC) VerifyEntrySuffix(hashOfHeader *common.Hash, key uint64, numOfBits uint64) error {
	var value uint64 = bytesToInteger(hashOfHeader[:])

	if numOfBits == 0 {
		//this is error, we do not do % 1<<0
		return errors.New("Invalid numOfBits. Must be greater than 0.")
	}
	//value = value % (1 << numOfBits)
	guicolour.BrightCyanPrintf(true, "\n\n Проверавање nonce вредности:\n nonce = %d\n хеш = %d\n",
		key, value)
	//fmt.Println("proveravamo za broj ", value, " key: ", key)

	correctVal := fillTo56WithOnes(key, numOfBits)
	correctVal = Convert56to64Des(correctVal)
	correctVal = encoder.EncodingFunction(correctVal, numOfBits)
	if correctVal%(1<<numOfBits) == value%(1<<numOfBits) {
		guicolour.BrightCyanPrintf(true, "\n\n Провера успешна за !\n")
		return nil
	} else {
		guicolour.BrightRedPrintf(true, "\n\n Провера неуспешна.\n Хеш блока = %d\n Исправни хеш = %d\n", value, correctVal)
		return errors.New("Invalid seal!")
	}
}
