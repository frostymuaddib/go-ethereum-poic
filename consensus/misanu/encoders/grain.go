package encoders

//GrainEncoder encodingFunction
type GrainEncoder struct {
	EncoderImplementation
}

func linearFunction(nfsr *BitVector, lfsr *BitVector) uint8 {
	var bit uint8
	var sum uint8
	bit, _ = lfsr.GetBit(0)
	sum += bit
	bit, _ = lfsr.GetBit(13)
	sum += bit
	bit, _ = lfsr.GetBit(23)
	sum += bit
	bit, _ = lfsr.GetBit(38)
	sum += bit
	bit, _ = lfsr.GetBit(51)
	sum += bit
	bit, _ = lfsr.GetBit(62)
	sum += bit
	return sum % 2
}

func nonLinearFunction(nfsr *BitVector, lfsr *BitVector) uint8 {
	var bit uint8
	var sum uint8

	nfsr9, _ := nfsr.GetBit(9)
	nfsr15, _ := nfsr.GetBit(15)
	nfsr21, _ := nfsr.GetBit(21)
	nfsr28, _ := nfsr.GetBit(28)
	nfsr33, _ := nfsr.GetBit(33)
	nfsr37, _ := nfsr.GetBit(37)
	nfsr45, _ := nfsr.GetBit(45)
	nfsr52, _ := nfsr.GetBit(52)
	nfsr60, _ := nfsr.GetBit(60)
	nfsr63, _ := nfsr.GetBit(63)
	bit, _ = lfsr.GetBit(0)
	sum += bit
	sum += nfsr63
	sum += nfsr60
	sum += nfsr52
	sum += nfsr45
	sum += nfsr37
	sum += nfsr33
	sum += nfsr28
	sum += nfsr21
	sum += nfsr15
	sum += nfsr9
	bit, _ = nfsr.GetBit(0)
	sum += bit

	sum += nfsr63 * nfsr60
	sum += nfsr37 * nfsr33
	sum += nfsr15 * nfsr9

	sum += nfsr60 * nfsr52 * nfsr45
	sum += nfsr33 * nfsr28 * nfsr21

	sum += nfsr63 * nfsr45 * nfsr28 * nfsr9
	sum += nfsr60 * nfsr52 * nfsr37 * nfsr33
	sum += nfsr63 * nfsr60 * nfsr21 * nfsr15

	sum += nfsr63 * nfsr60 * nfsr52 * nfsr45 * nfsr37
	sum += nfsr33 * nfsr28 * nfsr21 * nfsr15 * nfsr9

	sum += nfsr52 * nfsr45 * nfsr37 * nfsr33 * nfsr28 * nfsr21

	return sum % 2
}
func filterFunction(nfsr *BitVector, lfsr *BitVector) uint8 {
	var bit uint8
	var hx uint8

	lfsr3, _ := lfsr.GetBit(3)
	lfsr64, _ := lfsr.GetBit(64)
	lfsr25, _ := lfsr.GetBit(25)
	lfsr46, _ := lfsr.GetBit(46)
	nfsr63, _ := nfsr.GetBit(63)

	hx += lfsr25
	hx += nfsr63

	hx += lfsr3 * lfsr64
	hx += lfsr46 * lfsr64
	hx += lfsr64 * nfsr63

	hx += lfsr3 * lfsr25 * lfsr46
	hx += lfsr3 * lfsr46 * lfsr64
	hx += lfsr3 * lfsr46 * nfsr63
	hx += lfsr25 * lfsr46 * nfsr63
	hx += lfsr46 * lfsr64 * nfsr63

	hx %= 2

	bit, _ = nfsr.GetBit(0)
	return (hx + bit) % 2
}

//gubi se najmanj element, novi se dodaje na najvecu poziciju (80)
//ovo je ShiftToRight
func siftuj(registar []uint8, novaVrednost uint8) {
	for i := 0; i < 79; i++ {
		registar[i] = registar[i+1]
	}
	registar[79] = novaVrednost
}

func initializationRegisters(nfsr *BitVector, lfsr *BitVector, binaryNum *BitVector, numOfBits uint64) {
	var bit uint8
	for i := uint64(0); i < 80; i++ {
		//na primer na svako drugo mesto u nelinearnom registru stavljamo po jedan bit
		//binarnog vektora - kao da je x1x1x1x1x1x1x....111111111

		// 40 ne treba da zakucamo, nego je to 2x duzina binaryNum
		if i >= 80-numOfBits/2 {
			bit, _ = binaryNum.GetBit(i - 80 + numOfBits/2)
			nfsr.SetBit(i, bit)
			bit, _ = binaryNum.GetBit(i - 80 + numOfBits)
			lfsr.SetBit(i, bit)
		} else {
			nfsr.SetBit(i, 1)
			lfsr.SetBit(i, 1)
		}

	}

	for i := 0; i < 160; i++ {
		tl := linearFunction(nfsr, lfsr)
		tn := nonLinearFunction(nfsr, lfsr)

		f := filterFunction(nfsr, lfsr)

		bit, _ = lfsr.GetBit(0)
		nfsr.ShiftToRight((tn + f + bit) % 2)
		lfsr.ShiftToRight((tl + f) % 2)
	}

}

func grain(nfsr *BitVector, lfsr *BitVector, zr *BitVector) {

	for i := uint64(0); i < 100; i++ {
		tl := linearFunction(nfsr, lfsr)
		tn := nonLinearFunction(nfsr, lfsr)
		// zr stores output with h(x)
		zr.SetBit(i, filterFunction(nfsr, lfsr))
		bit, _ := lfsr.GetBit(0)
		nfsr.ShiftToRight((tn + bit) % 2)
		lfsr.ShiftToRight(tl)
	}

}

func decToBinary(n uint64, binaryNum []uint8, numOfBits uint64) {
	// counter for binary array
	var i uint64
	for n > 0 {
		// storing remainder in binary array
		binaryNum[i] = uint8(n % 2)
		n = n / 2
		i++
		if i == numOfBits {
			break
		}
	}
	for ; i < numOfBits; i++ {
		binaryNum[i] = 0
	}
}

func binToDec(binNum []uint8, numOfBits uint64) (br uint64) {
	br = 0
	var stepen2 uint64 = 1
	var i uint64
	for i = 0; i < numOfBits; i++ {
		if binNum[i] == 1 {
			br += stepen2
		}
		stepen2 *= 2
	}
	return br
}

func kodniElement(infVektor *BitVector) uint8 {
	var rez uint8

	bit, _ := infVektor.GetBit(0)
	rez += bit
	bit, _ = infVektor.GetBit(4)
	rez += bit
	bit, _ = infVektor.GetBit(9)
	rez += bit
	bit, _ = infVektor.GetBit(28)
	rez += bit
	bit, _ = infVektor.GetBit(43)
	rez += bit
	bit, _ = infVektor.GetBit(84)
	rez += bit
	bit, _ = infVektor.GetBit(99)
	rez += bit
	return rez % 2
}

// ovo se menja sa ShiftToLeft
func shiftInfVector(infVector []uint8, newElem uint8, numOfBits uint64) {
	for i := numOfBits - 1; i > 0; i-- {
		infVector[i] = infVector[i-1]
	}
	infVector[0] = newElem
}

/*EncodingFunction - final implementation of Grain used for consensus */
func (e GrainEncoder) EncodingFunction(key uint64, numOfBits uint64) uint64 {
	var i uint64
	var lfsr = CreateBitVectors(80, false)
	var nfsr = CreateBitVectors(80, false)

	var binaryNum = CreateBitVectorFromDecimal(key, numOfBits)
	var grainOutput = CreateBitVectors(100, false)

	//decToBinary(x, binaryNum, numOfBits)

	initializationRegisters(nfsr, lfsr, binaryNum, numOfBits)

	grain(nfsr, lfsr, grainOutput)

	var T = 1000
	var izlazIzPomerackog = make([]uint8, T)

	for t := 0; t < T; t++ {
		r := kodniElement(grainOutput /*, 100*/)
		izlazIzPomerackog[t] = r
		//shiftInfVector(grainOutput, r, 100)
		grainOutput.ShiftToLeft(r)
	}
	// od 1000 brisemo 880 i ostaje nam 120
	// kojih 120? kako mi odredimo
	var pomocni [120]uint8
	for i = 0; i < 120; i++ {
		pomocni[i] = izlazIzPomerackog[i*8]
	}
	var codeWord = make([]uint8, numOfBits)
	for i = 0; i < numOfBits; i++ {
		codeWord[i] = pomocni[119-i]
	}
	var y = binToDec(codeWord, numOfBits)
	return y
}
