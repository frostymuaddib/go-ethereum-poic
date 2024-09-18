package misanu

//contains various test parameters

//digest se cita sa spoljnog izvora + timestamp
const OUTER_DIGEST = false

//this is hack in order to skip initialization at the beginning every time
const (
	preloadTable     bool   = false
	tablePath        string = "/home/frosty/tabela25x15.csv"
	preloadTablePart bool   = false
	taplePartPath    string = "/scratch/ethereum/tabela102433554432.csv.part"
)

const (
	TABLE_WIDTH  uint64 = 1 << 2
	TABLE_HEIGHT uint64 = 1 << 2
)

const (
	//number of seconds that passed since block's creation and reading outer data
	OUTER_TIME_DELTA int64 = 3
)
