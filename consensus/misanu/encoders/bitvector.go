package encoders

import (
	"bytes"
	"errors"
	"fmt"

	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
)

//BitVector - structure represents simple bit vector implementation
type BitVector struct {
	bits      []uint64
	BitNumber uint64
}

//CreateBitVectors - creates bitvector with given length. Can be initialized
//with ones or zeroes
func CreateBitVectors(length uint64, initWithOne bool) (result *BitVector) {
	result = new(BitVector)
	arrayLen := (length / 64) + 1
	result.BitNumber = length
	result.bits = make([]uint64, arrayLen)
	for i := uint64(0); i < arrayLen-1; i++ {
		if initWithOne {
			result.bits[i] = 0xFFFFFFFFFFFFFFFF
		} else {
			result.bits[i] = 0
		}
	}
	if initWithOne {
		result.bits[arrayLen-1] = 0xFFFFFFFFFFFFFFFF % uint64(1<<length-(arrayLen-1)*64)
	} else {
		result.bits[arrayLen-1] = 0
	}
	return result
}

//ToString - prints the content of BitVector
func (b *BitVector) ToString() string {

	var rez bytes.Buffer
	for i := 0; i < len(b.bits); i++ {
		rez.WriteString(fmt.Sprintf("%X.", b.bits[i]))
	}
	return rez.String()
}

//ToDecimal - dummy
func (b *BitVector) ToDecimal(numOfBits uint) uint64 {
	// var rez uint64 = 0
	// var pow uint64 = 1
	// for i := 0; i < len(b.bits); i++ {
	// 	rez += pow * uint64(b.bits[i])
	// 	pow <<= 64
	// }
	// return rez
	return 0
}

//CreateBitVectorFromDecimal - creates new bit vector and initializes it with
//given decimal value.
func CreateBitVectorFromDecimal(decimal uint64, numOfBits uint64) (result *BitVector) {
	result = new(BitVector)
	result.BitNumber = numOfBits
	arrayLength := (numOfBits / 64) + 1
	result.bits = make([]uint64, arrayLength)
	var i uint64
	// for i = 0; i < arrayLength; i++ {
	// 	result.bits[i] = uint64(decimal % 256)
	// 	decimal -= uint64(result.bits[i])
	// 	decimal /= 256
	// 	if decimal == 0 {
	// 		i++
	// 		break
	// 	}
	// }
	result.bits[0] = decimal
	for i = 1; i < arrayLength; i++ {
		result.bits[i] = 0
	}

	return result
	// return nil
}

//CopyTo - copies values from one bit vector to another
func (b *BitVector) CopyTo(from *BitVector) {
	var min uint64
	if b.BitNumber < from.BitNumber {
		min = b.BitNumber
	} else {
		min = from.BitNumber
	}
	arrayLen := min / 64
	for i := uint64(0); i < arrayLen; i++ {
		b.bits[arrayLen] = from.bits[arrayLen]
	}
	//poslednji uint64 mozda nije ceo u broju, mora bit po bit
	bitsRemaining := min - arrayLen*64
	if bitsRemaining != 0 {
		b.bits[arrayLen] = b.bits[arrayLen] - (b.bits[arrayLen] % (1 << uint(bitsRemaining))) +
			(from.bits[arrayLen] % (1 << uint(bitsRemaining)))
	}
}

//CopyToRange - copies values from one bit vector to another but only given
//number of bits
func (b *BitVector) CopyToRange(from *BitVector, numOfBits uint64) {
	var min uint64
	if numOfBits > from.BitNumber {
		numOfBits = from.BitNumber
	}
	if b.BitNumber < numOfBits {
		min = b.BitNumber
	} else {
		min = numOfBits
	}
	arrayLen := min / 64
	for i := uint64(0); i < arrayLen; i++ {
		b.bits[arrayLen] = from.bits[arrayLen]
	}
	//poslednji uint64 mozda nije ceo u broju, mora bit po bit
	bitsRemaining := min - arrayLen*64
	if bitsRemaining != 0 {
		b.bits[arrayLen] = b.bits[arrayLen] - (b.bits[arrayLen] % (1 << uint(bitsRemaining))) +
			(from.bits[arrayLen] % (1 << uint(bitsRemaining)))
	}
}

//GetBit - returns specific bit at given position
func (b *BitVector) GetBit(position uint64) (uint8, error) {
	if position >= b.BitNumber {
		guicolour.BrightMagentaPrintf(true, "GET: Seeked position greater than bit length.")
		return 0, errors.New("seeked position greater than bit length")
	}
	arrayPos := position / 64
	bitNum := position - arrayPos*64
	if b.bits[arrayPos]&(1<<bitNum) > 0 {
		return 1, nil
	}
	return 0, nil

}

//SetBit - sets a specific bit of a bit vector
func (b *BitVector) SetBit(position uint64, bit uint8) error {
	if position >= b.BitNumber {
		guicolour.BrightMagentaPrintf(true, "SET: Seeked position greater than bit length.")
		return errors.New("seeked position greater than bit length")
	}
	if bit > 1 {
		return errors.New("invalid bit value")
	}
	arrayPos := position / 64
	bitNum := position - arrayPos*64
	if bit == 1 {
		b.bits[arrayPos] |= (1 << bitNum)
		return nil
	}
	b.bits[arrayPos] &= ((^uint64(0)) ^ (1 << bitNum))
	return nil

}

/*ShiftToRight siftuj(int *registar, int novaVrednost)*/
func (b *BitVector) ShiftToRight(newBit uint8) error {
	if newBit > 1 {
		return errors.New("invalid bit value")
	}
	var i int
	for i = 0; i < len(b.bits)-1; i++ {
		b.bits[i] >>= 1
		if b.bits[i+1]%2 == 1 {
			b.bits[i] |= (1 << 63)
		}
	}

	b.bits[i] >>= 1
	if newBit == 1 {
		b.bits[i] |= (1 << ((b.BitNumber - uint64(i*64)) - 1))
	}
	return nil
}

/*ShiftToLeft - shiftInfVector(int *inf_vector, int newElem)*/
func (b *BitVector) ShiftToLeft(newBit uint8) error {
	if newBit > 1 {
		return errors.New("invalid bit value")
	}
	var i int
	restBits := (b.BitNumber - (b.BitNumber/64)*64)
	//svi do restBits treba da budu 1, znaci
	for i = len(b.bits) - 1; i > 0; i-- {
		b.bits[i] <<= 1
		if b.bits[i-1]&(1<<63) > 0 {
			b.bits[i]++
		}
		if i == len(b.bits)-1 && restBits != 0 {
			b.bits[i] &= (1 << restBits) - 1
		}
	}
	b.bits[0] <<= 1
	if newBit == 1 {
		b.bits[0]++
	}
	return nil
}
