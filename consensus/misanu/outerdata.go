package misanu

//this will contain functions for obtaining some data from outer source (e.g. stock
//exchange) used to create mix digest, along with function for verifying that data
//during block validation

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"time"

	"github.com/frostymuaddib/go-ethereum-poic/common"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
	"github.com/frostymuaddib/go-ethereum-poic/core/types"
)

func abs(num int64) int64 {
	if num >= 0 {
		return num
	} else {
		return -num
	}
}

// func createRandom32bytes() (*[]byte, error) {
// 	random := make([]byte, 32)
//
// 	_, err := rand.Read(random)
// 	if err != nil {
// 		// handle error here
// 		fmt.Printf("Error while generating random ")
// 		return nil, err
// 	}
// 	return &random, nil
// }

func verifyOuterSource(header *types.Header, deltaTime int64) error {
	bytes := [32]byte(header.MixDigest)
	var timestamp int64
	if isLittleEndian() {
		timestamp = int64(binary.LittleEndian.Uint64(bytes[0:8]))
	} else {
		timestamp = int64(binary.BigEndian.Uint64(bytes[0:8]))
	}
	guicolour.BrightCyanPrintf(true, "Временска ознака током провере блока %d\n", timestamp)
	blockTime := int64(header.Time)
	if abs(blockTime-timestamp) > deltaTime {
		return errors.New("Invalid timestamp of outer data")
	}
	//Here it would contact outer source, fetch the data for given time and validate it.

	return nil
}

func readOuterSource(target []byte, timestamp int64) error {
	//reads from outer source, but for now it will be random number
	_, err := rand.Read(target)
	if err != nil {
		guicolour.BrightRedPrintf(true, "Error while reading from outer source")
		return err
	}
	return nil
}

func mixDigestFromOuterSource() (*common.Hash, error) {
	timestamp := time.Now().Unix()
	digest := make([]byte, 32)

	if isLittleEndian() {
		binary.LittleEndian.PutUint64(digest[0:8], uint64(timestamp))
	} else {
		binary.BigEndian.PutUint64(digest[0:8], uint64(timestamp))
	}
	//8 bytes are used for unix time, the rest of 24 bytes should be from public domain
	err := readOuterSource(digest[8:32], timestamp)
	if err != nil {
		// handle error here
		guicolour.BrightRedPrintf(true, "Error while creating mixDigest")
		return nil, err
	}
	ret, e := bytesToHash(&digest)
	if e != nil {
		guicolour.BrightRedPrintf(true, "Error converting bytes to hash.")
		return nil, err
	}
	return ret, nil
}
