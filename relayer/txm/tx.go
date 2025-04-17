package txm

import (
	"github.com/fbsobreira/gotron-sdk/pkg/address"
)

type TronTx struct {
	FromAddress     address.Address
	ContractAddress address.Address
	Method          string
	Params          []any
	Attempt         uint64
	OutOfTimeErrors uint64
	EnergyBumpTimes uint32
}
