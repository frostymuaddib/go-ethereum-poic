package misanu

//u ovoj datoteci ce stajati sve stvari vezane za time memory tradeoff
//kao i sve verzije istog, ukljucujuci grain i sl.

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"

	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
)

func (p *PoIC) getXforGivenYBlockEncoder(value uint64, threadID int, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, rez chan xForY) {
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

func (p *PoIC) locateRowBlockEncoder(threadId int, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}) (*uint64, error) {

	var ret *uint64
	for i := uint64(0); i < p.table.tableWidth; i++ {
		select {
		case <-abort:
			return nil, errors.New("ABORT")
		default:
			rez := make(chan xForY)
			go p.getXforGivenYBlockEncoder(value, threadId, startPos, endPos, numOfBits, abort, rez)
			pointX := <-rez
			if pointX.Err != nil {
				return nil, errors.New("ABORT")
			}
			if pointX.X != nil {
				ret = new(uint64)
				*ret = *pointX.X
				return ret, nil
			}
			value = encoder.EncodingFunction(value, numOfBits) //% (1 << numOfBits)
		}

	}
	return nil, nil
}

func (p *PoIC) getInterStateBlockEncoder(threadId int, pocetno uint64, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}) (*uint64, error) {
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

func (p *PoIC) getInterStateBlockEncoderV6(threadId int, pocetno uint64, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}) (*uint64, error) {
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

func largeRandom(numOfBits uint64) (uint64, error) {
	max := new(big.Int)
	max.SetString(fmt.Sprint(1<<(64-numOfBits)), 10)
	ran, err := rand.Int(rand.Reader, max)
	if err != nil {
		guicolour.BrightRedPrintf(true, "Немогуће направити велики ранд!\n")
		return 0, err
	}
	return ran.Uint64(), nil
}
