package misanu

//u ovoj datoteci ce stajati sve stvari vezane za time memory tradeoff
//kao i sve verzije istog, ukljucujuci grain i sl.

import (
	"errors"
	"fmt"
	"runtime"
	"sort"
	"sync"

	"github.com/frostymuaddib/go-ethereum-poic/common"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"

	"github.com/frostymuaddib/go-ethereum-poic/core/types"
)

func (p *PoIC) getXforGivenYParallelFirstIteration(value uint64, threadID int, numOfBits uint64, abort chan struct{}, rez chan tableRows) {
	var ret tableRows
	p.tableMutex.RLock()
	defer p.tableMutex.RUnlock()
	target := value % (1 << numOfBits)
	select {
	case <-abort:
		ret.Err = errors.New("ABORT")
		rez <- ret
		return
	default:
		resultRow := sort.Search(int(p.table.tableHeight), func(i int) bool {
			if p.tablePartiallySorted.tableMap == nil {
				return false
			}
			result := toUint64(p.tablePartiallySorted.tableMap[uint64(i)*16+8 : uint64(i)*16+16])
			result %= (1 << numOfBits)
			return result >= target
		})
		resRow := uint64(resultRow)

		if resultRow < int(p.table.tableHeight) && toUint64(p.tablePartiallySorted.tableMap[resRow*16+8:resRow*16+16])%(1<<numOfBits) == target {
			//row found, check if there are others
			x := toUint64(p.tablePartiallySorted.tableMap[resRow*16 : resRow*16+8])
			y := toUint64(p.tablePartiallySorted.tableMap[resRow*16+8:resRow*16+16]) % (1 << numOfBits)
			ret.Xs = append(ret.Xs, x)
			select {
			case <-abort:
				ret.Err = errors.New("ABORT")
				rez <- ret
				return
			default:
				for i := uint64(1); i < p.table.tableHeight-uint64(resultRow) && toUint64(p.tablePartiallySorted.tableMap[resRow*16+i*16+8:resRow*16+i*16+16])%(1<<numOfBits) == y; i++ {
					ret.Xs = append(ret.Xs, toUint64(p.tablePartiallySorted.tableMap[resRow*16+i*16:resRow*16+8+i*16]))
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
			}

		} else {
			select {
			case <-abort:
				ret.Err = errors.New("ABORT")
				rez <- ret
				return
			default:
				rez <- ret
				return
			}
		}
	}

}

func (p *PoIC) getXforGivenYParallel(value uint64, threadID int, numOfBits uint64, abort chan struct{}, rez chan tableRows) {
	var ret tableRows
	p.tableMutex.RLock()
	defer p.tableMutex.RUnlock()
	target := value
	select {
	case <-abort:
		ret.Xs = nil
		ret.Err = errors.New("ABORT")
		rez <- ret
		return
	default:
		resultRow := sort.Search(int(p.table.tableHeight), func(i int) bool {
			if p.table.tableMap == nil {
				return false
			}
			result := toUint64(p.table.tableMap[uint64(i)*16+8 : uint64(i)*16+16])
			return result >= target
		})
		resRow := uint64(resultRow)

		if resultRow < int(p.table.tableHeight) && toUint64(p.table.tableMap[resRow*16+8:resRow*16+16]) == target {
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
			guicolour.BrightCyanPrintf(true, "\n\n Унутрашњост табеле %d!\nnadjena vredonst: %d\ntarget %d x %d\n", threadID, toUint64(p.table.tableMap[resRow*16+8:resRow*16+16]), target, x)

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

func (p *PoIC) locateRowParallel(threadId int, value uint64, iteration uint64, numOfBits uint64, abort chan struct{}) ([]uint64, uint64, uint64, error) {
	var i uint64
	for i = iteration; i < p.table.tableWidth; i++ {
		select {
		case <-abort:
			return nil, 0, 0, errors.New("ABORT")
		default:
			rez := make(chan tableRows)
			if i == 0 {
				go p.getXforGivenYParallelFirstIteration(value, threadId, numOfBits, abort, rez)
			} else {
				go p.getXforGivenYParallel(value, threadId, numOfBits, abort, rez)
			}
			rows := <-rez
			if rows.Err != nil {
				return nil, 0, 0, rows.Err
			}
			if len(rows.Xs) != 0 {
				return rows.Xs, value, i, nil
			}
			value = encoder.EncodingFunction(value, numOfBits)
		}

	}
	return nil, value, i, nil
}

func (p *PoIC) minerParallel(threadId int, value uint64, numOfBits uint64, abort chan struct{}, found chan *types.BlockNonce) {
	var numberOfTries uint64 = 32
	//var numberOfTries uint64 = 1 << (64 - numOfBits)
	var rows []uint64
	var rez *uint64
	var err error
	for filler := uint64(0); filler < numberOfTries; filler++ {
		select {
		case <-abort:
			return
		default:
			value = value % (1 << numOfBits)
			ran, er := largeRandom(numOfBits)

			if er != nil {
				//aborted returns
				return
			}
			value += ran << numOfBits

			currentValue := value
			currentIteration := uint64(0)
			for currentIteration < p.table.tableWidth {
				var e error
				rows, currentValue, currentIteration, e = p.locateRowParallel(threadId, currentValue, currentIteration, numOfBits, abort)
				if e != nil {
					//aborted returns
					return
				}
				for i := 0; i < len(rows); i++ {
					if currentIteration == 0 {
						rez, err = p.getInterStateBlockEncoder(threadId, rows[i], value, 0, p.table.tableHeight, numOfBits, abort)
					} else {
						guicolour.BrightCyanPrintf(true, "\n\n Red x %d \n trazimo %d\n", rows[i], value)
						rez, err = p.getInterStateBlockEncoderV6(threadId, rows[i], value, 0, p.table.tableHeight, numOfBits, abort)
					}
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
				currentValue = encoder.EncodingFunction(currentValue, numOfBits)
			}

		}
	}

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

func (p *PoIC) FindKeyForHashMultiThreadWholeTable(hashOfHeader *common.Hash, numOfBits uint64, stop <-chan struct{}) (*types.BlockNonce, error) {
	var value uint64 = bytesToInteger(hashOfHeader[:])
	if numOfBits == 0 {
		//this is error, we do not do %1<<0
		return nil, errors.New("Invalid numbOfBits. Must be greater than 0.")
	}
	abort := make(chan struct{})
	found := make(chan *types.BlockNonce)
	var pending sync.WaitGroup
	cpuNum := runtime.NumCPU()
	for i := 0; i < cpuNum; i++ {
		pending.Add(1)
		go func(threadId int, value uint64, numOfBits uint64, abort chan struct{}, found chan *types.BlockNonce) {
			defer pending.Done()
			p.minerParallel(threadId, value, numOfBits, abort, found)
		}(i, value, numOfBits, abort, found)
	}
	select {
	case <-stop:
		close(abort)
		return nil, errors.New("ABORTED")
	default:

		var result *types.BlockNonce = nil
	rezultati:
		for {
			select {
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
		if result == nil {
			//ovo sada nije greska, krenuti ponovo
			fmt.Println("Key with current digest not found. Retry.")
			return nil, nil
			//errors.New("Error: key not found!")
		} else {
			guicolour.BrightRedPrintf(true, "НАЂЕНО ЗА %d ВРЕДНОСТ %d", value, result)
			return result, nil
		}
	}
}
