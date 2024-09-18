package misanu

//u ovoj datoteci ce stajati sve stvari vezane za time memory tradeoff
//kao i sve verzije istog, ukljucujuci grain i sl.

import (
	"errors"
	"sort"

	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"

	"github.com/frostymuaddib/go-ethereum-poic/core/types"
)

type tableRows struct {
	Xs  []uint64
	Err error
}

func (p *PoIC) getXforGivenYBlockEncoderV5(value uint64, threadID int, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, rez chan xForY) {
	//DES_MOD value je vec modovan sa 1<<n
	x, y, currentPosition, err := p.GetCurrentRowBinary(startPos, endPos)
	var ret xForY

	for err == nil {
		select {
		case <-abort:
			ret.X = nil
			ret.Err = errors.New("ABORT")
			rez <- ret
			return
		default:

			if y%(1<<numOfBits) == value%(1<<numOfBits) {
				select {

				case <-abort:
					ret.X = nil
					ret.Err = errors.New("ABORT")
					rez <- ret
					return
				default:
					ret.X = &x
					ret.RowNumber = &currentPosition
					ret.Err = nil
					rez <- ret
					return
				}
			}
			x, y, currentPosition, err = p.GetCurrentRowBinary(currentPosition, endPos)
		}
	}
	ret.X = nil
	ret.Err = nil
	rez <- ret
}

func (p *PoIC) getXforGivenYBinarySearchV5(value uint64, threadID int, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, rez chan tableRows) {
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
		resultRow := sort.Search(int(endPos-startPos), func(i int) bool {
			if p.table.tableMap == nil {
				return false
			}
			result := toUint64(p.table.tableMap[startPos+uint64(i)*16+8 : startPos+uint64(i)*16+16])
			result %= (1 << numOfBits)
			return result >= target
		})
		resRow := uint64(resultRow) + startPos

		if resultRow < int(endPos-startPos) && toUint64(p.table.tableMap[resRow*16+8:resRow*16+16])%(1<<numOfBits) == target {
			//row found, check if there are others
			x := toUint64(p.table.tableMap[resRow*16 : resRow*16+8])
			y := toUint64(p.table.tableMap[resRow*16+8:resRow*16+16]) % (1 << numOfBits)
			ret.Xs = append(ret.Xs, x)
			select {
			case <-abort:
				ret.Err = errors.New("ABORT")
				rez <- ret
				return
			default:
				for i := uint64(1); toUint64(p.table.tableMap[resRow*16+i*16+8:resRow*16+8+i*16+16])%(1<<numOfBits) == y; i++ {

					tmp := toUint64(p.table.tableMap[resRow*16+i*16 : resRow*16+8+i*16])
					if tmp != x {
						//row is different
						ret.Xs = append(ret.Xs, tmp)
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

func (p *PoIC) locateRowBlockEncoderV5(threadId int, value uint64, iteration uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}) ([]uint64, uint64, uint64, error) {
	var i uint64
	for i = iteration; i < p.table.tableWidth; i++ {
		select {
		case <-abort:
			return nil, 0, 0, errors.New("ABORT")
		default:
			rez := make(chan tableRows)
			go p.getXforGivenYBinarySearchV5(value, threadId, startPos, endPos, numOfBits, abort, rez)
			rows := <-rez
			if rows.Err != nil {
				return nil, 0, 0, rows.Err
			}
			if len(rows.Xs) != 0 {
				return rows.Xs, value, i, nil
			}
			value = encoder.EncodingFunction(value, numOfBits) //% (1 << numOfBits)
		}

	}
	retRow := new(uint64)
	*retRow = endPos
	return nil, value, i, nil
}

//This variant of miner uses different, hopefully faster table search

func (p *PoIC) minerVer5BlockEncoder(threadId int, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, found chan *types.BlockNonce) {
	var numberOfTries uint64 = 32
	var rows []uint64
	//TODO RANDOMIZOVATI
	//Ovde ima smisla ograniciti jer ako prolazi kroz sve varijante dopune nikada
	//nece da zavrsi?
	//for filler := uint64(0); filler <combinations;filler++ {
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
				rows, currentValue, currentIteration, e = p.locateRowBlockEncoderV5(threadId, currentValue, currentIteration, startPos, endPos, numOfBits, abort)
				if e != nil {
					//aborted returns
					return
				}
				//guicolour.BrightGreenPrintf(true, "ROW NUMBER %d\n", rowNumber)
				// if len(rows) >0{
				// 	guicolour.BrightGreenPrintf(true, "Нађено %d редова\n", len(rows))
				// }
				for i := 0; i < len(rows); i++ {
					rez, err := p.getInterStateBlockEncoder(threadId, rows[i], value, startPos, endPos, numOfBits, abort)
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
