package encoders

import "unsafe"

/*Encoder interface has EncodiningFunction that is used for consensus protocol.
Various implementations can be created here: Grain, Des, TripleDes*/
type Encoder interface {
	EncodingFunction(x uint64, numOfBits uint64) uint64
}

/*EncoderImplementation wrapper*/
type EncoderImplementation struct {
}

/*GetEncoder - function returns active encoder. Here, it should be changed for
various implementations. Currently exists: GrainEncoder, DesEncoder,
 TripleDesEncoder*/
func GetEncoder() Encoder {
	//return GrainEncoder{}
	return DesEncoder{}
}

//help functions
func isLittleEndian() bool {
	n := uint32(0x01020304)
	return *(*byte)(unsafe.Pointer(&n)) == 0x04
}
