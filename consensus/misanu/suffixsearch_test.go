package misanu

//u ovoj datoteci ce stajati sve stvari vezane za time memory tradeoff
//kao i sve verzije istog, ukljucujuci grain i sl.

import (
	"testing"

	"github.com/frostymuaddib/go-ethereum-poic/common"
)

//rez, err = p.getInterStateBlockEncoderSuffix(threadId, rows[i], value, startPos, endPos, numOfBits, abort)
//func (p *PoIC) getInterStateBlockEncoderSuffix(threadId int, pocetno uint64, value uint64, startPos uint64, endPos uint64, numOfBits uint64, abort chan struct{}) (*uint64, error) {

func BenchmarkPoicCalculation(b *testing.B) {
	numOfBits := uint64(25)
	/*	conf := params.PoICConfig{
			TableSize:    numOfBits,
			EncryptSize:  0,
			ExpectedTime: 0,
			MiniPoW:      nil,
			OuterData:    nil,
		}

		p := &PoIC{
			config:       &conf,
			db:           nil,
			workCh:       make(chan *types.Block),
			resultCh:     make(chan *types.Block),
			fetchWorkCh:  make(chan *sealWork),
			submitWorkCh: make(chan *mineResult),
			fetchRateCh:  make(chan chan uint64),
			submitRateCh: make(chan *hashrate),
			exitCh:       make(chan chan error),
		}*/
	//taken from mainnet, random block
	blockHash := common.HexToHash("0xd783efa4d392943503f28438ad5830b2d5964696ffc285f338585e9fe0a37a05")

	var value uint64 = bytesToInteger(blockHash[:])
	currentValue := value % (1 << numOfBits)
	//p.CreateTableBinarySuffix(p.config.TableSize)
	//p.minerSuffix(threadId, value, startPos, endPos, numOfBits, abort, found)
	//p.getInterStateBlockEncoderSuffix(0, value, )

	//ovo treba racunati, to je moja primitiva
	for i := 0; i < b.N; i++ {
		res := fillTo56WithOnes(currentValue, numOfBits)
		res = Convert56to64Des(res)
		res = encoder.EncodingFunction(res, numOfBits)
	}
}
