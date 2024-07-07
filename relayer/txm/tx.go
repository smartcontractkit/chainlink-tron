package txm

type TronTx struct {
	FromAddress     string
	ContractAddress string
	Method          string
	Params          []any
	Attempt         uint64
	OutOfTimeErrors uint64
	EnergyBumpTimes uint64
}
