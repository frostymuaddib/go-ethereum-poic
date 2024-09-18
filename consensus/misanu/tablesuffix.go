package misanu

import (
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"runtime"
	"sort"
	"sync"

	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/utils"

	mmap "github.com/edsrzf/mmap-go"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
	//"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/testutils"
)

const FILL_MASK uint64 = 0x00FFFFFFFFFFFFFF

func fillTo56WithOnes(number uint64, suffixLength uint64) uint64 {
	sl := uint64(1) << suffixLength
	//1<<3 = 00001000; 00001000 - 1 = 00000111; ~00000111 = 11111000
	newMask := FILL_MASK & ^(sl - uint64(1))
	return newMask + (number % sl)
}

//KORISTIM OVU VERZIJU!
//creates table that only compares suffixes (the numbers in table are only suffixes not whole
//64 bit numbers)
func (p *PoIC) CreateTableBinarySuffix(tableSize uint64) error {
	type tableEl struct {
		X uint64
		Y uint64
	}
	p.tableMutex.Lock()
	defer p.tableMutex.Unlock()
	p.table = new(TradeoffTable)
	p.table.tableWidth, p.table.tableHeight = chooseTableDimension()
	p.table.spaceSize = tableSize
	var (
		table *os.File
		error error
	)
	if preloadTable {
		table, error = os.Open(tablePath)
		p.table.tablePath = tablePath
		if error != nil {
			return error
		}
	} else {
		dir, err := ioutil.TempDir("", "timeMemoryTradeoffTable")
		guicolour.BrightCyanPrintf(true, "\n\n Прављење табеле димензије: %d x %d. Простор је %d\n",
			p.table.tableWidth, p.table.tableHeight, p.table.spaceSize)
		name := fmt.Sprintf("/tabela%d%d.csv", p.table.tableWidth, p.table.tableHeight)
		if err != nil {
			return err
		}
		guicolour.BrightCyanPrintf(true, "\n\n Путања табеле је: %s\n", dir+name)
		p.table.tablePath = dir + name
		table, error = os.OpenFile(p.table.tablePath, os.O_WRONLY|os.O_CREATE, 0644)
		if error != nil {
			return error
		}
		//tableArray := make([]tableEl, p.table.tableHeight)
		var step uint64

		cpuNum := runtime.NumCPU()
		step = p.table.tableHeight / uint64(cpuNum)
		var pendSort sync.WaitGroup

		tableArray := utils.RandomPermutationFromRangeLessMemory(1<<tableSize, p.table.tableHeight)
		guicolour.BrightCyanPrintf(true, "\n\n Направљен насумични низ величине: %d бајтова\n", uintptr(len(tableArray))*reflect.TypeOf(tableArray).Elem().Size())
		mod := uint64(1) << p.config.TableSize
		for i := uint64(0); i < uint64(cpuNum); i++ {
			pendSort.Add(1)
			go func(cpu uint64) {
				//time.Sleep(20 * time.Second)
				defer pendSort.Done()
				var i, j, tmp uint64
				for i = cpu * step; i < (cpu+1)*step; i++ {
					tmp = tableArray[i].X
					for j = 0; j < p.table.tableWidth; j++ {
						tmp = tmp % mod
						if j == 0 {
							tableArray[i].X = tmp
						} else if j == p.table.tableWidth-1 {
							tableArray[i].Y = tmp
						}
						tmp = fillTo56WithOnes(tmp, p.config.TableSize)
						tmp = Convert56to64Des(tmp)
						tmp = encoder.EncodingFunction(tmp, p.config.TableSize)
					}

				}

			}(i)
		}
		pendSort.Wait()
		guicolour.BrightCyanPrintf(true, "\n\n Почиње сортирање табеле.\n")
		sort.Slice(tableArray, func(i int, j int) bool {
			return tableArray[i].Y < tableArray[j].Y
		})
		for i := uint64(0); i < p.table.tableHeight; i++ {
			table.Write(toBinaryNumber(tableArray[i].X))
			table.Write(toBinaryNumber(tableArray[i].Y))
		}

		table.Close()
		//create mmap
		table, error = os.Open(dir + name)
		if error != nil {
			return error
		}
	}

	mapa, error := mmap.Map(table, mmap.RDONLY, 0)
	if error != nil {
		return error
	}
	guicolour.BrightCyanPrintf(true, "\n\n Табела је направљена. Иницијализација завршена.\n")
	p.table.tableMap = mapa
	return nil
}
