package txm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	txmgrtypes "github.com/smartcontractkit/chainlink-framework/chains/txmgr/types"
)

type TronTx struct {
	FromAddress     address.Address
	ContractAddress address.Address
	Method          string
	Params          []any
	Attempt         uint64
	OutOfTimeErrors uint64
	EnergyBumpTimes uint32
	Meta            *txmgrtypes.TxMeta[common.Address, common.Hash]
}
