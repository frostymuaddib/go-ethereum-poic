package misanu

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sort"
	"sync"

	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/utils"

	mmap "github.com/edsrzf/mmap-go"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
	//"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/testutils"
)

type TradeoffTable struct {
	tableMap    mmap.MMap
	tableHeight uint64
	tableWidth  uint64
	spaceSize   uint64
	tablePath   string
	//currentPosition uint64
}

const (
	mask7 uint64 = 0x7F
	mask8 uint64 = 0xFF
)

func Convert56to64Des(x56 uint64) uint64 {
	x64 := uint64(0)
	for i := uint64(0); x56 > 0; i++ {
		x64 += ((x56 & mask7) << 1) << (i * 8)
		x56 >>= 7
	}
	return x64
}

func Convert64to56Des(x64 uint64) uint64 {
	x56 := uint64(0)
	for i := uint64(0); x64 > 0; i++ {
		x56 += ((x64 & mask8) >> 1) << (i * 7)
		x64 >>= 8
	}
	return x56
}

//ovo je br bitova elementa u tabeli, zato smo pravili tabelu 2^20.
//za sada u ovoj varijanti i pravim tabelu 2x2^20, a za time-memory tradeoff
//ce biti prvo 2^5x2^15
//TODO: implementirati logiku za generisanje velicine tableLen

func chooseTableDimension() (uint64, uint64) {
	//prvo proveravamo da li postoji konfiguracioni fajl sa dimenzijama
	//tabele, ako nema citamo default

	//zakucano ime!
	file, err := os.Open("dimenzije.txt")
	if err != nil {
		guicolour.BrightYellowPrintf(true, "Нема dimenzije.txt. Учитавање подразумеваних вредности.\n")
		return TABLE_WIDTH, TABLE_HEIGHT
	} else {
		var w uint64
		var h uint64
		n, e := fmt.Fscanf(file, "%d %d", &w, &h)
		if n != 2 {
			guicolour.BrightYellowPrintf(true, "Нису учитане обе димензије из dimenzije.txt. Учитавање подразумеваних вредности.\n")
			return TABLE_WIDTH, TABLE_HEIGHT
		}
		if e != nil {
			guicolour.BrightYellowPrintf(true, "Грешка при читању dimenzije.txt. Учитавање подразумеваних вредности.\n")
			return TABLE_WIDTH, TABLE_HEIGHT
		}
		guicolour.BrightYellowPrintf(true, "Учитане вредности: %d x %d.\n", w, h)
		return 1 << w, 1 << h
	}

	guicolour.BrightYellowPrintf(true, "Нема dimenzije.txt. Учитавање подразумеваних вредности.\n")
	return TABLE_WIDTH, TABLE_HEIGHT
}

func toBinaryNumber(num uint64) []byte {
	bin := make([]byte, 8)
	if isLittleEndian() {
		binary.LittleEndian.PutUint64(bin, num)
	} else {
		binary.BigEndian.PutUint64(bin, num)
	}
	return bin
}

func toUint64(bin []byte) uint64 {
	if isLittleEndian() {
		return binary.LittleEndian.Uint64(bin)
	} else {
		return binary.BigEndian.Uint64(bin)
	}
}

func (p *PoIC) CreateTableBinary(tableSize uint64) error {
	type tableEl struct {
		X uint64
		Y uint64
	}
	p.tableMutex.Lock()
	defer p.tableMutex.Unlock()
	p.table = new(TradeoffTable)
	p.tablePartiallySorted = new(TradeoffTable)
	p.table.tableWidth, p.table.tableHeight = chooseTableDimension()
	p.tablePartiallySorted.tableWidth, p.tablePartiallySorted.tableHeight = p.table.tableWidth, p.table.tableHeight
	p.table.spaceSize = tableSize
	p.tablePartiallySorted.spaceSize = tableSize
	var (
		table        *os.File
		tablePartial *os.File
		error        error
	)
	if preloadTable {
		table, error = os.Open(tablePath)
		p.table.tablePath = tablePath
		if error != nil {
			return error
		}

		if preloadTablePart {
			tablePartial, error = os.Open(taplePartPath)
			p.tablePartiallySorted.tablePath = taplePartPath
			if error != nil {
				return error
			}
		} else {
			mapa, error := mmap.Map(table, mmap.RDONLY, 0)
			if error != nil {
				return error
			}
			tableArray := make([]tableEl, p.table.tableHeight)
			for i := uint64(0); i < p.table.tableHeight; i++ {
				tableArray[i].X = toUint64(mapa[i*16 : i*16+8])
				tableArray[i].Y = toUint64(mapa[i*16+8 : i*16+16])
			}
			mapa.Unmap()
			guicolour.BrightCyanPrintf(true, "\n\n Почиње сортирање part табеле.\n")
			sort.Slice(tableArray, func(i int, j int) bool {
				return tableArray[i].Y%(1<<p.table.spaceSize) < tableArray[j].Y%(1<<p.table.spaceSize)
			})
			p.tablePartiallySorted.tablePath = p.table.tablePath + ".part"
			tablePartial, error = os.OpenFile(p.tablePartiallySorted.tablePath, os.O_WRONLY|os.O_CREATE, 0644)
			if error != nil {
				return error
			}
			for i := uint64(0); i < p.table.tableHeight; i++ {
				tablePartial.Write(toBinaryNumber(tableArray[i].X))
				tablePartial.Write(toBinaryNumber(tableArray[i].Y))
			}
			tablePartial.Close()
			tablePartial, error = os.Open(p.tablePartiallySorted.tablePath)
			if error != nil {
				return error
			}

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
		tableArray := make([]tableEl, p.table.tableHeight)
		var step uint64
		cpuNum := runtime.NumCPU()
		//deo za proveru duplikata, iskomentarisati kada se to ne radi
		// tc := make([]*testutils.TableCounter, cpuNum)
		// for i := 0; i < cpuNum; i++ {
		// 	tc[i] = testutils.NewTableCounter()
		//
		// }
		// var uniqueNumbers uint64
		// var uniqMutex sync.Mutex

		step = p.table.tableHeight / uint64(cpuNum)
		var pendSort sync.WaitGroup
		permutation := utils.RandomPermutationFromRange(tableSize, p.table.tableHeight)
		guicolour.BrightCyanPrintf(true, "Milan\n")
		for _, el := range permutation {
			guicolour.BrightCyanPrintf(true, "%d\n", el)
		}
		for i := uint64(0); i < uint64(cpuNum); i++ {
			pendSort.Add(1)
			go func(cpu uint64) {
				defer pendSort.Done()
				var i, j, tmp uint64
				for i = cpu * step; i < (cpu+1)*step; i++ {
					//tmp = Convert56to64Des(i)
					tmp = Convert56to64Des(permutation[i])
					for j = 0; j < p.table.tableWidth; j++ {
						// sum := uint8(0)
						// for k := 0; k < cpuNum; k++ {
						// 	sum += tc[k].ReturnCounter(tmp)
						// }
						// if sum == 0 {
						// 	uniqMutex.Lock()
						// 	uniqueNumbers++
						// 	uniqMutex.Unlock()
						// }
						// tc[cpu].PutNumber(tmp)
						if j == 0 {
							tableArray[i].X = tmp
						} else if j == p.table.tableWidth-1 {
							tableArray[i].Y = tmp
						}
						tmp = encoder.EncodingFunction(tmp, p.config.TableSize)
					}

				}

			}(i)
		}
		pendSort.Wait()
		//guicolour.BrightWhitePrintf(true, "Број различитих елемената у табели %d\n Број елемената укупно %d\n ",uniqueNumbers,p.table.tableWidth*p.table.tableHeight)
		//deo za proveru duplikata, iskomentarisati kada se to ne radi
		// tstf, _ := os.Create("/home/frosty/duplikati.txt")
		// br := uint64(0xFFFFFFFFFFFFFFFF)
		// for i := uint64(0); i < br; i++ {
		// 	add := uint8(0)
		// 	for j := 0; j < cpuNum; j++ {
		// 		add += tc[j].ReturnCounter(i)
		// 	}
		//
		// 	if add > 0 {
		// 		tstf.WriteString(fmt.Sprint(i))
		// 		tstf.WriteString(": ")
		// 		tstf.WriteString(fmt.Sprint(add))
		// 		tstf.WriteString("\n")
		// 	}
		// }
		// tstf.Close()

		guicolour.BrightCyanPrintf(true, "\n\n Почиње сортирање табеле.\n")
		sort.Slice(tableArray, func(i int, j int) bool {
			return tableArray[i].Y < tableArray[j].Y
		})
		for i := uint64(0); i < p.table.tableHeight; i++ {
			table.Write(toBinaryNumber(tableArray[i].X))
			table.Write(toBinaryNumber(tableArray[i].Y))
		}

		table.Close()
		//partial table
		p.tablePartiallySorted.tablePath = p.table.tablePath + ".part"
		tablePartial, error = os.OpenFile(p.tablePartiallySorted.tablePath, os.O_WRONLY|os.O_CREATE, 0644)
		if error != nil {
			return error
		}
		guicolour.BrightCyanPrintf(true, "\n\n Почиње сортирање part табеле.\n")
		sort.Slice(tableArray, func(i int, j int) bool {
			return tableArray[i].Y%(1<<p.table.spaceSize) < tableArray[j].Y%(1<<p.table.spaceSize)
		})
		for i := uint64(0); i < p.table.tableHeight; i++ {
			tablePartial.Write(toBinaryNumber(tableArray[i].X))
			tablePartial.Write(toBinaryNumber(tableArray[i].Y))
		}
		tablePartial.Close()
		//create mmap
		table, error = os.Open(dir + name)
		if error != nil {
			return error
		}
		tablePartial, error = os.Open(p.tablePartiallySorted.tablePath)
		if error != nil {
			return error
		}

	}

	mapa, error := mmap.Map(table, mmap.RDONLY, 0)
	if error != nil {
		return error
	}
	mapaPart, errorpart := mmap.Map(tablePartial, mmap.RDONLY, 0)
	if errorpart != nil {
		return errorpart
	}
	guicolour.BrightCyanPrintf(true, "\n\n Табела је направљена. Иницијализација завршена.\n")
	p.table.tableMap = mapa
	p.tablePartiallySorted.tableMap = mapaPart
	return nil
}

func (p *PoIC) ClearTable() error {
	p.tableMutex.Lock()
	error := p.table.tableMap.Unmap()
	p.table.tableMap = nil
	if error != nil {
		guicolour.BrightRedPrintf(true, "Немогуће урадити unmap табеле\n")
		p.tableMutex.Unlock()
		return error
	}
	guicolour.BrightCyanPrintf(true, "Umap табеле урађен\n")
	if !preloadTable {
		error = os.Remove(p.table.tablePath)
	}
	if error != nil {
		p.tableMutex.Unlock()
		return error
	}
	guicolour.BrightCyanPrintf(true, "Датотека табеле обрисана\n")
	p.tableMutex.Unlock()
	return nil
}

func (p *PoIC) GetCurrentRowBinary(startPosition uint64, endPosition uint64) (key uint64, val uint64, newPosition uint64, err error) {
	p.tableMutex.RLock()
	if p.table.tableMap == nil {
		p.tableMutex.RUnlock()
		return 0, 0, startPosition, errors.New("Table deleted.")
	}
	tableLen := len(p.table.tableMap)
	if startPosition*16+16 > uint64(tableLen) {
		p.tableMutex.RUnlock()
		return 0, 0, startPosition, errors.New("Table out of bounds.")
	}
	key = toUint64(p.table.tableMap[startPosition*16 : startPosition*16+8])
	val = toUint64(p.table.tableMap[startPosition*16+8 : startPosition*16+16])
	newPosition = startPosition + 1
	p.tableMutex.RUnlock()
	return key, val, newPosition, nil
}
