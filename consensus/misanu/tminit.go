package misanu

//u ovoj datoteci ce stajati sve stvari vezane za time memory tradeoff
//kao i sve verzije istog, ukljucujuci grain i sl.

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"runtime"
	"sync"
	"unsafe"

	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/encoders"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"

	"github.com/frostymuaddib/go-ethereum-poic/common"
	"github.com/frostymuaddib/go-ethereum-poic/core/types"
)

type xForY struct {
	X         *uint64
	RowNumber *uint64
	Err       error
}

var encoder = encoders.GetEncoder()

// isLittleEndian returns whether the local system is running in little or big
// endian byte order.
func isLittleEndian() bool {
	n := uint32(0x01020304)
	return *(*byte)(unsafe.Pointer(&n)) == 0x04
}

func bytesToInteger(arr []byte) uint64 {
	if isLittleEndian() {
		return binary.LittleEndian.Uint64(arr)
	} else {
		return binary.BigEndian.Uint64(arr)
	}

}

func integerToBytes(number uint64) *[]byte {
	bs := make([]byte, 32)
	if isLittleEndian() {
		binary.LittleEndian.PutUint64(bs, number)
	} else {
		binary.BigEndian.PutUint64(bs, number)
	}
	return &bs
}

func createRandom32bytes() (*[]byte, error) {
	random := make([]byte, 32)

	_, err := rand.Read(random)
	if err != nil {
		// handle error here
		fmt.Printf("Error while generating random ")
		return nil, err
	}
	return &random, nil
}

func createRandomMixDigest() (*common.Hash, error) {
	random, err := createRandom32bytes()
	if err != nil {
		// handle error here
		fmt.Printf("Error while generating random ")
		return nil, err
	}
	ret, e := bytesToHash(random)
	if e != nil {
		fmt.Printf("Error converting bytes to hash.")
		return nil, err
	}
	return ret, nil
}

func bytesToNonce(number *[]byte) (*types.BlockNonce, error) {
	//nas zanima samo prvih 8 bajtova, vise od toga nece bit, ali imamo proveru za saki slucaj
	if len(*number) > 8 {
		for i := 8; i < len(*number); i++ {
			if (*number)[i] > 0 {
				return nil, errors.New("Invalid value for nonce: greater than 8 bytes")
			}
		}
	}
	var ret [8]byte
	for i := 0; i < 8; i++ {
		ret[i] = (*number)[i]
	}
	val := types.BlockNonce(ret)
	return &val, nil
}

func bytesToHash(number *[]byte) (*common.Hash, error) {
	//nas zanima samo prvih 32 bajtova, vise od toga nece bit, ali imamo proveru za saki slucaj
	if len(*number) > 32 {
		for i := 32; i < len(*number); i++ {
			if (*number)[i] > 0 {
				return nil, errors.New("Invalid value for mixDigest: greater than 32 bytes")
			}
		}
	}
	var ret [32]byte
	for i := 0; i < 32; i++ {
		ret[i] = (*number)[i]
	}
	val := common.Hash(ret)
	return &val, nil
}

func (p *PoIC) FindKeyForHashMultiThread(hashOfHeader *common.Hash, numOfBits uint64, stop <-chan struct{}) (*types.BlockNonce, error) {
	var value uint64 = bytesToInteger(hashOfHeader[:])
	if numOfBits == 0 {
		//this is error, we do not do %1<<0
		return nil, errors.New("Invalid numbOfBits. Must be greater than 0.")
	}
	//DES_MOD
	//value = value % (1 << numOfBits)
	//guicolour.BrightCyanPrintf(true, "\n\n Радимо Proof-of-Inverted-Capacity за хеш вредност: %d\n", value)

	abort := make(chan struct{})
	found := make(chan *types.BlockNonce)
	var pending sync.WaitGroup
	cpuNum := runtime.NumCPU()
	var step uint64
	step = p.table.tableHeight / uint64(cpuNum)
	//guicolour.BrightCyanPrintf(true, "\n\n Покреће се %d нити за претрагу табеле.\n ", cpuNum)
	for i := 0; i < cpuNum; i++ {
		pending.Add(1)
		go func(threadId int, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}, found chan *types.BlockNonce) {
			defer pending.Done()
			p.minerVer6BlockEncoder(threadId, value, startPos, endPos, numOfBits, abort, found)
		}(i, value, uint64(i)*step, uint64((i+1))*step, numOfBits, abort, found)
	}
	select {
	case <-stop:
		close(abort)
		return nil, errors.New("ABORTED")
	default:

		var result *types.BlockNonce = nil
	rezultati:
		for {
			guicolour.BrightRedPrintf(true, "CPU: %d\n", cpuNum)
			select {
			// case <-stop:
			// 	// Outside abort, stop all miner threads
			// 	close(abort)
			case result = <-found:
				// One of the threads found a block, abort all others
				cpuNum--
				if result != nil {
					guicolour.BrightCyanPrintf(true, "\n\n Нит нашла рез %d!\n", result)
					close(abort)
					break rezultati
				} else {
					if cpuNum == 0 {
						//no results
						break rezultati
					}
				}

				// case <-ethash.update:
				// 	// Thread count was changed on user request, restart
				// 	close(abort)
				// 	pend.Wait()
				// 	return ethash.Seal(chain, block, stop)
			}
		}
		pending.Wait()
		// Wait for all miners to terminate and return the block
		if result == nil {
			//ovo sada nije greska, krenuti ponovo
			fmt.Println("Key with current digest not found. Retry.")
			return nil, nil
			//errors.New("Error: key not found!")
		} else {
			guicolour.BrightRedPrintf(true, "НАЂЕНО ЗА %d ВРЕДНОСТ %d", value, result)
			return result, nil
		}
	}
}

/*func (p *PoIC) VerifyEntry(hashOfHeader *common.Hash, key uint64, numOfBits uint64) error {
	var value uint64 = bytesToInteger(hashOfHeader[:])

	if numOfBits == 0 {
		//this is error, we do not do % 1<<0
		return errors.New("Invalid numOfBits. Must be greater than 0.")
	}
	value = value % (1 << numOfBits)
	guicolour.BrightCyanPrintf(true, "\n\n Проверавање nonce вредности:\n nonce = %d\n хеш = %d\n",
		key, value)
	//fmt.Println("proveravamo za broj ", value, " key: ", key)

	correctVal := encoder.EncodingFunction(key, numOfBits)
	//DES_MOD: dodati po modulu od numOfBits bitova
	correctVal = correctVal % (1 << numOfBits)
	if correctVal == value {
		guicolour.BrightCyanPrintf(true, "\n\n Провера успешна за !\n")
		return nil
	} else {
		guicolour.BrightRedPrintf(true, "\n\n Провера неуспешна.\n Хеш блока = %d\n Исправни хеш = %d\n", value, correctVal)
		return errors.New("Invalid seal!")
	}
}
*/
func (p *PoIC) getInterState(threadId int, pocetno uint64, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}) (*uint64, error) {
	var ret *uint64
	for i := uint64(0); i < p.table.tableWidth; i++ {
		select {
		case <-abort:
			return nil, errors.New("ABORTED")
		default:
			if encoder.EncodingFunction(pocetno, numOfBits) == value {
				ret = new(uint64)
				*ret = pocetno
				return ret, nil
			} else {
				pocetno = encoder.EncodingFunction(pocetno, numOfBits)
			}
		}
	}
	return nil, nil
}
