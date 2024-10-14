package misanu

import (
	"errors"
	"sort"

	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
)

func (p *PoIC) SearchTableFoRows(value uint64, numOfBits uint64, abort chan struct{}) ([]uint64, error) {
	select {
	case <-abort:
		return nil, errors.New("ABORT")
	default:
		rez := make(chan tableRows)
		go p.getX(value, numOfBits, abort, rez)
		rows := <-rez
		if rows.Err != nil {
			return nil, rows.Err
		}
		if len(rows.Xs) != 0 {
			return rows.Xs, nil
		}

		// value = fillTo56WithOnes(value, numOfBits)
		// value = Convert56to64Des(value)
		// value = encoder.EncodingFunction(value, numOfBits)
	}

	return nil, nil
}

func (p *PoIC) getX(value uint64, numOfBits uint64, abort chan struct{}, rez chan tableRows) {
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

func (p *PoIC) SearchRow(value uint64, rows []uint64, numOfBits uint64) *uint64 {
	//guicolour.BrightWhitePrintf(true, "Редови који се претражују %v \n", rows)
	for i := 0; i < len(rows); i++ {
		rez, err := p.lookIntoRow(rows[i], value, numOfBits)
		if err != nil {
			//function aborted, return
			return nil
		}
		if rez != nil {
			guicolour.BrightCyanPrintf(true, "Вредност пронађена %d\n", *rez)
			return rez
		}
	}
	return nil
}

func (p *PoIC) lookIntoRow(firstColumn uint64, value uint64, numOfBits uint64) (*uint64, error) {
	var ret *uint64
	mod := uint64(1 << numOfBits)
	pocetno := firstColumn //% mod
	for i := uint64(0); i < p.table.tableWidth; i++ {
		novo := fillTo56WithOnes(pocetno, numOfBits)
		novo = Convert56to64Des(novo)
		novo = encoder.EncodingFunction(novo, numOfBits)
		if novo%mod == value%mod {
			ret = new(uint64)
			*ret = pocetno
			return ret, nil
		} else {
			pocetno = novo
		}
	}
	return nil, nil
}