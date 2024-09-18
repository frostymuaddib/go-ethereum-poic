package misanu

//u ovoj datoteci ce stajati sve stvari vezane za time memory tradeoff
//kao i sve verzije istog, ukljucujuci grain i sl.

import (
	"errors"
	"sort"

	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/RandomFork"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"

	"github.com/frostymuaddib/go-ethereum-poic/core/types"
)

func (p *PoIC) getXforGivenYMultiFirstIteration(value uint64, threadID int, table TradeoffTablePartial, numOfBits uint64, abort chan struct{}, rez chan tableRows) {
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
		resultRow := sort.Search(int(table.tableHeight), func(i int) bool {
			if table.tableMapPartially == nil {
				return false
			}
			result := toUint64(table.tableMapPartially[uint64(i)*16+8 : uint64(i)*16+16])
			result %= (1 << numOfBits)
			return result >= target
		})
		resRow := uint64(resultRow)

		if resultRow < int(table.tableHeight) && toUint64(table.tableMapPartially[resRow*16+8:resRow*16+16])%(1<<numOfBits) == target {
			//row found, check if there are others
			x := toUint64(table.tableMapPartially[resRow*16 : resRow*16+8])
			y := toUint64(table.tableMapPartially[resRow*16+8:resRow*16+16]) % (1 << numOfBits)
			ret.Xs = append(ret.Xs, x)
			select {
			case <-abort:
				ret.Err = errors.New("ABORT")
				rez <- ret
				return
			default:
				for i := uint64(1); toUint64(p.tablePartiallySorted.tableMap[resRow*16+i*16+8:resRow*16+8+i*16+16])%(1<<numOfBits) == y; i++ {
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

func (p *PoIC) getXforGivenYMulti(value uint64, threadID int, table TradeoffTablePartial, numOfBits uint64, abort chan struct{}, rez chan tableRows) {
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
		resultRow := sort.Search(int(table.tableHeight), func(i int) bool {
			if table.tableMapPartially == nil {
				return false
			}
			result := toUint64(table.tableMapPartially[uint64(i)*16+8 : uint64(i)*16+16])
			return result >= target
		})
		resRow := uint64(resultRow)

		if resultRow < int(table.tableHeight) && toUint64(table.tableMapPartially[resRow*16+8:resRow*16+16]) == target {
			//row found, check if there are others
			x := toUint64(table.tableMapPartially[resRow*16 : resRow*16+8])
			y := toUint64(table.tableMapPartially[resRow*16+8 : resRow*16+16])
			ret.Xs = append(ret.Xs, x)
			for i := uint64(1); toUint64(table.tableMapPartially[resRow*16+i*16+8:resRow*16+8+i*16+16]) == y; i++ {
				select {
				case <-abort:
					ret.Xs = nil
					ret.Err = errors.New("ABORT")
					rez <- ret
					return
				default:
					ret.Xs = append(ret.Xs, toUint64(table.tableMapPartially[resRow*16+i*16:resRow*16+8+i*16]))
				}
			}

		} else {
			ret.Xs = nil
			ret.Err = nil
			rez <- ret
			return
		}
	}

}

func (p *PoIC) locateRowMulti(threadId int, value uint64, iteration uint64, table TradeoffTablePartial, numOfBits uint64, abort chan struct{}) ([]uint64, uint64, uint64, error) {
	var i uint64
	for i = iteration; i < p.table.tableWidth; i++ {
		select {
		case <-abort:
			return nil, 0, 0, errors.New("ABORT")
		default:
			rez := make(chan tableRows)
			if i == 0 {
				go p.getXforGivenYMultiFirstIteration(value, threadId, table, numOfBits, abort, rez)
			} else {
				go p.getXforGivenYMulti(value, threadId, table, numOfBits, abort, rez)
			}
			rows := <-rez
			if rows.Err != nil {
				return nil, 0, 0, rows.Err
			}
			if len(rows.Xs) != 0 {
				return rows.Xs, value, i, nil
			}
			//primenim fork pre nego sto sam izracunao vrednost - da bi poredjenje bilo dobro
			//trazim binarno vrednost, ako je nema uradim fork vrednosti pa E(vrednost)
			//pa dopunim
			if table.usesFork {
				value = RandomFork.ForkBits(table.forkBits, value)
				value = Convert56to64Des(value)
			}
			value = encoder.EncodingFunction(value, numOfBits)

		}

	}
	retRow := new(uint64)
	*retRow = table.tableHeight
	return nil, value, i, nil
}

//OVCDE ISPRAVI DSA RADE ZA MALE TABELE
func (p *PoIC) getInterStateMultiFirst(threadId int, pocetno uint64, value uint64, table TradeoffTablePartial, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}) (*uint64, error) {
	var ret *uint64
	mod := uint64(1 << numOfBits)
	for i := uint64(0); i < p.table.tableWidth; i++ {
		select {
		case <-abort:
			return nil, errors.New("ABORTED")
		default:
			novo := encoder.EncodingFunction(pocetno, numOfBits)
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

func (p *PoIC) getInterStateBlockEncoderV62(threadId int, pocetno uint64, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}) (*uint64, error) {
	var ret *uint64
	for i := uint64(0); i < p.table.tableWidth; i++ {
		select {
		case <-abort:
			return nil, errors.New("ABORTED")
		default:
			novo := encoder.EncodingFunction(pocetno, numOfBits)
			if novo == value {
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

func (p *PoIC) minerMultipleTables(threadId int, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, found chan *types.BlockNonce) {
	var numberOfTries uint64 = 32
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
			for _, table := range p.tableSet.smallerTables {
				//setnja po tabelama
				currentIteration := uint64(0)
				for currentIteration < p.table.tableWidth {
					var e error
					//MILAN ovde - prepraviti sve funkcije da traze po tabeli
					rows, currentValue, currentIteration, e = p.locateRowMulti(threadId, currentValue, currentIteration, table, numOfBits, abort)
					if e != nil {
						//aborted returns
						return
					}
					for i := 0; i < len(rows); i++ {
						if currentIteration == 0 {
							rez, err = p.getInterStateBlockEncoder(threadId, rows[i], value, startPos, endPos, numOfBits, abort)
						} else {
							rez, err = p.getInterStateBlockEncoderV6(threadId, rows[i], value, startPos, endPos, numOfBits, abort)
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
