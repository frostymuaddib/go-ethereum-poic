package encoders

import (
	"crypto/des"
	"encoding/binary"
	"fmt"
)

/*TripleDesEncoder uses TripleDes for encoding function*/
type TripleDesEncoder struct {
	EncoderImplementation
}

func convertUint64to24bit(val uint64, numOfBits uint64) []byte {
	//des zahteva da je kljuc sa 24 bajtova. OStalo dopuniti sa jedinicama npr
	result := make([]byte, 24)
	temp := make([]byte, 24)
	if isLittleEndian() {
		binary.LittleEndian.PutUint64(temp, val)
	} else {
		binary.BigEndian.PutUint64(temp, val)
	}

	for i := uint(0); i < 24; i++ {
		if numOfBits == 0 {
			result[i] = 0xFF
		} else if numOfBits < 8 {
			var shift byte = 1 << numOfBits
			result[i] = (temp[i] % shift) + 0xFF - shift + 1
			numOfBits = 0
		} else {
			result[i] = temp[i]
			numOfBits -= 8
		}

	}

	return result

}

func convertByteArray24(val [24]byte) uint64 {
	if isLittleEndian() {
		return binary.LittleEndian.Uint64(val[:])
	}
	return binary.BigEndian.Uint64(val[:])

}

/*EncodingFunction - tripleDes*/
func (d TripleDesEncoder) EncodingFunction(key uint64, numOfBits uint64, isInit bool) uint64 {

	block, error := des.NewTripleDESCipher(convertUint64to24bit(key, numOfBits))
	if error != nil {
		fmt.Printf("НЕМОГУЋЕ НАПРАВИТИ ДЕС КОДЕР! %s\n", error.Error())
		return 0
	}
	//Moramo odabrati tekst koji se sifruje. Bice 00FF00FF00FF00FF...
	var text [24]byte
	for i := 0; i < 24; i++ {
		if i%2 == 0 {
			text[i] = 0
		} else {
			text[i] = 0xFF
		}
	}
	var code [24]byte
	block.Encrypt(code[:], text[:])

	return convertByteArray24(code)
}
