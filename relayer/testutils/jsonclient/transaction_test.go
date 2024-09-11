package jsonclient

import (
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

var expectedGetTransactionInfoById = &api.TransactionInfo{
	ID:             "7c2d4206c03a883dd9066d620335dc1be272a8dc733cfa3f6d10308faa37facc",
	Fee:            1100000,
	BlockNumber:    32880248,
	BlockTimeStamp: 1681368027000,
	ContractResult: []string{
		"",
	},
	Receipt: api.ResourceReceipt{
		NetFee: 100000,
	},
}
