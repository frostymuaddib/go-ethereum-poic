package misanu

import (
	//"fmt"
	//"io/ioutil"
	//"os"
	//"runtime"
	//"sync"

	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"sort"
	"sync"

	mmap "github.com/edsrzf/mmap-go"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/RandomFork"
	"github.com/frostymuaddib/go-ethereum-poic/consensus/misanu/guicolour"
)

type TradeoffTableSet struct {
	spaceSize      uint64
	numberOfTables uint64
	tableWidth     uint64
	smallerTables  []TradeoffTablePartial
	totalHeight    uint64
}

type TradeoffTablePartial struct {
	tableMap          mmap.MMap
	tableMapPartially mmap.MMap
	tableHeight       uint64
	tablePath         string
	tablePathPart     string
	usesFork          bool
	fork              uint64
	forkBits          []uint8
}

func ChooseTableDimension_Set() (height uint64, width uint64) {
	width = 1 << 10
	height = 1 << 20
	return height, width
}

//creates a set of smaller tables with forks that are kept in tmp dir and mmaped
//the name of each table is tabela_{redBr}_{fork}_{widh}x{height}.csv
//if fork is 0, no fork is used
func (p *PoIC) CreateTableSet(tableSize uint64, numberOfTables uint64) error {
	p.tableSet = new(TradeoffTableSet)
	p.tableSet.tableWidth, p.tableSet.totalHeight = ChooseTableDimension_Set()
	p.tableSet.numberOfTables = numberOfTables
	p.tableSet.smallerTables = make([]TradeoffTablePartial, numberOfTables)

	type tableEl struct {
		X uint64
		Y uint64
	}
	dir, err := ioutil.TempDir("", "timeMemoryTableSet")
	if err != nil {
		return err
	}
	guicolour.BrightCyanPrintf(true, "\n\n Прављење %d табела димензије: %d x %d. Простор је %d\n",
		numberOfTables, p.tableSet.tableWidth, p.tableSet.totalHeight/uint64(numberOfTables), p.table.spaceSize)
	tableArray := make([][]tableEl, p.tableSet.numberOfTables)
	for i := uint64(0); i < numberOfTables; i++ {
		p.tableSet.smallerTables[i].tableHeight = p.tableSet.totalHeight / uint64(numberOfTables)
		name := fmt.Sprintf("tabela_%d_%dx%d.csv", i, p.tableSet.tableWidth, p.tableSet.smallerTables[i].tableHeight)
		p.tableSet.smallerTables[i].tablePath = dir + name
		if i == 0 {
			//first table does not use fork
			p.tableSet.smallerTables[i].usesFork = false
			p.tableSet.smallerTables[i].fork = 0
		} else {
			//TODO: UBACITI RANDOM FORK, ALI ZA SADA IDEMO REDOM
			p.tableSet.smallerTables[i].usesFork = true
			p.tableSet.smallerTables[i].fork = i
			p.tableSet.smallerTables[i].forkBits = RandomFork.DecodeToPositions(8, p.tableSet.smallerTables[i].fork)
		}
		table, tableError := os.Create(p.tableSet.smallerTables[i].tablePath)
		if tableError != nil {
			return tableError
		}
		guicolour.BrightCyanPrintf(true, "\n\n Путања табеле %d је: %s\n", i, dir+name)
		tableArray[i] = make([]tableEl, p.tableSet.smallerTables[i].tableHeight)
		cpuNum := runtime.NumCPU()
		step := p.tableSet.smallerTables[i].tableHeight / uint64(cpuNum)
		var pend sync.WaitGroup
		for c := 0; c < cpuNum; c++ {
			pend.Add(1)
			go func(cpu uint64) {
				defer pend.Done()
				var o, j, tmp uint64
				for o = cpu * step; o < (cpu+1)*step; o++ {
					tmp = Convert56to64Des(o)
					for j = 0; j < p.tableSet.tableWidth; j++ {
						if j == 0 {
							tableArray[i][o].X = tmp
						} else if j == p.tableSet.tableWidth-1 {
							tableArray[i][o].Y = tmp
						}
						if p.tableSet.smallerTables[i].usesFork {
							tmp = RandomFork.ForkBits(p.tableSet.smallerTables[i].forkBits, tmp)
						}
						tmp = encoder.EncodingFunction(tmp, p.config.TableSize)
						//Ovo mozda za sada da ne diram, ali mozda mora da se iskomentarise
						tmp = Convert56to64Des(tmp)

					}
				}
			}(uint64(c))
		}
		pend.Wait()
		guicolour.BrightCyanPrintf(true, "\n\n Почиње сортирање %d табеле \n", i)
		sort.Slice(tableArray[i], func(a int, b int) bool {
			return tableArray[i][a].Y < tableArray[i][b].Y
		})
		for o := uint64(0); o < p.tableSet.smallerTables[i].tableHeight; o++ {
			table.Write(toBinaryNumber(tableArray[i][o].X))
			table.Write(toBinaryNumber(tableArray[i][o].Y))
		}
		table.Close()

		p.tableSet.smallerTables[i].tablePathPart = p.tableSet.smallerTables[i].tablePath + ".part"
		table, tableError = os.Create(p.tableSet.smallerTables[i].tablePathPart)
		if tableError != nil {
			return tableError
		}
		guicolour.BrightCyanPrintf(true, "\n\n Почиње сортирање part %d табеле \n", i)
		sort.Slice(tableArray[i], func(a int, b int) bool {
			return tableArray[i][a].Y%(1<<p.tableSet.spaceSize) < tableArray[i][b].Y%(1<<p.tableSet.spaceSize)
		})

		for o := uint64(0); o < p.tableSet.smallerTables[i].tableHeight; o++ {
			table.Write(toBinaryNumber(tableArray[i][o].X))
			table.Write(toBinaryNumber(tableArray[i][o].Y))
		}
		table.Close()

		table, tableError = os.Create(p.tableSet.smallerTables[i].tablePath)
		if tableError != nil {
			return tableError
		}
		tablePart, tableErrorPart := os.Create(p.tableSet.smallerTables[i].tablePathPart)
		if tableErrorPart != nil {
			return tableError
		}

		mapa, errorMapa := mmap.Map(table, mmap.RDONLY, 0)
		if errorMapa != nil {
			return errorMapa
		}
		mapaPart, errorMapaPart := mmap.Map(tablePart, mmap.RDONLY, 0)
		if errorMapaPart != nil {
			return errorMapaPart
		}
		guicolour.BrightCyanPrintf(true, "\n\n Табела %d је направљена.\n", i)
		p.tableSet.smallerTables[i].tableMap = mapa
		p.tableSet.smallerTables[i].tableMapPartially = mapaPart

	}
	guicolour.BrightCyanPrintf(true, "\n\n Иницијализација завршена.\n")
	return nil
}
