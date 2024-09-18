package encoders

import (
	"crypto/des"
	"encoding/binary"

	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
)

/*DesEncoder uses Des for encoding function*/
type DesEncoder struct {
	EncoderImplementation
}

func convertUint64(val uint64) []byte {
	//arrayLength := uint(math.Ceil(float64(numOfBits)/float64(8)))

	//des zahteva da je kljuc sa 8 bajtova. OStalo dopuniti sa jedinicama npr
	result := make([]byte, 8)
	if isLittleEndian() {
		binary.LittleEndian.PutUint64(result, val)
	} else {
		binary.BigEndian.PutUint64(result, val)
	}

	return result

}

func uint64ToByteArray(val uint64) []byte {
	result := make([]byte, 8)
	if isLittleEndian() {
		binary.LittleEndian.PutUint64(result, val)
	} else {
		binary.BigEndian.PutUint64(result, val)
	}

	return result
}

func convertByteArray(val [8]byte) uint64 {
	if isLittleEndian() {
		return binary.LittleEndian.Uint64(val[:])
	}
	return binary.BigEndian.Uint64(val[:])

}

//Text that we encode - 00FF00FF00FF00FF
var text = [8]byte{0, 0xFF, 0, 0xFF, 0, 0xFF, 0, 0xFF}

/*EncodingFunction - implemented des for consensus algorithm*/
func (d DesEncoder) EncodingFunction(key uint64, numOfBits uint64) uint64 {
	//uzimamo samo numOfBIts kljuca, ostalo popunimo sa 111111
	block, error := des.NewCipher(convertUint64(key))
	if error != nil {
		guicolour.BrightRedPrintf(true, "НЕМОГУЋЕ НАПРАВИТИ ДЕС КОДЕР! %s\n", error.Error())
		return 0
	}
	var code [8]byte
	block.Encrypt(code[:], text[:])
	//rezultat moramo da modujemo sa 2<<(numOfBits)
	return convertByteArray(code) //%(2<<numOfBits)
}
