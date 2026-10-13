package txm

import (
	"time"

	"github.com/smartcontractkit/chainlink-tron/relayer/gotron-sdk/pkg/address"
)

type TronTx struct {
	FromAddress     address.Address
	ContractAddress address.Address
	Method          string
	Params          []any
	Attempt         uint64
	OutOfTimeErrors uint64
	EnergyBumpTimes uint32
	ID              string // idempotency key
	State           TxState
	CreateTs        time.Time
}
