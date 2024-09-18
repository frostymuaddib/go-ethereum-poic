package RandomFork

// function for calculating n choose k - n over k
func Binomial(n uint64, k uint64) uint64 {
	if n < 0 || k < 0 {
		//ordinary, this is an error, but for our encoding, this has to be zero
		return 0
	}
	if n < k {
		//ordinary, this is an error, but for our encoding, this has to be zero
		return 0
	}
	// (n,k) = (n,n-k)
	if k > n/2 {
		k = n - k
	}

	ret := uint64(1)
	for i := uint64(1); i <= k; i++ {
		ret = (n - k + i) * ret / i
	}
	return ret
}

//decodes ordinal number
//numOfSetBits - number of bits that have to be 1
//ordinal - number that is being decoded.
//ordinal goes from 0 to (n over numOfSetBits)-1
func Decode(numOfSetBits uint64, ordinal uint64) uint64 {
	bits := uint64(0)
	for bit := uint64(63); numOfSetBits > 0; bit-- {
		nCk := Binomial(bit, numOfSetBits)
		if ordinal >= nCk {
			ordinal -= nCk
			bits |= (1 << bit)
			numOfSetBits--
		}
	}
	return bits
}

//decodes ordinal number to array of position of set bits from rightmost to leftmost
//set bit. e.g. 1011 will return 0 1 3
//numOfSetBits - number of bits that have to be 1
//ordinal - number that is being decoded.
//ordinal goes from 0 to (n over numOfSetBits)-1
func DecodeToPositions(numOfSetBits uint64, ordinal uint64) []uint8 {
	//bits := uint64(0)
	result := make([]uint8, numOfSetBits)
	for bit := uint64(63); numOfSetBits > 0; bit-- {
		nCk := Binomial(bit, numOfSetBits)
		if ordinal >= nCk {
			ordinal -= nCk
			//bits |= (1<<bit)
			result[numOfSetBits-1] = uint8(bit)
			numOfSetBits--
		}
	}
	return result
}

//applies fork with numOfSetBits bits of 1 to the given value
func ForkNumber(fork uint64, value uint64, numOfSetBits uint64) uint64 {
	forkbits := DecodeToPositions(numOfSetBits, fork)
	//forkbits := [4]uint8{0, 1, 2, 4}
	pos := uint8(0)
	result := uint64(0)
	accumPos := uint8(0)
	for _, bit := range forkbits {
		bit -= pos
		rest := value % (1 << bit)
		value >>= bit + 1
		result += rest << accumPos
		accumPos += bit
		pos += bit + 1

	}
	return result + (value << accumPos)
}

//applies fork with numOfSetBits bits of 1 to the given value
func ForkBits(forkbits []uint8, value uint64) uint64 {
	//forkbits := [4]uint8{0, 1, 2, 4}
	pos := uint8(0)
	result := uint64(0)
	accumPos := uint8(0)
	for _, bit := range forkbits {
		bit -= pos
		rest := value % (1 << bit)
		value >>= bit + 1
		result += rest << accumPos
		pos += bit + 1
		accumPos += bit

	}
	return result + (value << accumPos)
}
