package fullnodeclient

import (
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronFullNodeClient) GetTransactionInfoById(txhash string) (*api.TransactionInfo, error) {
	return tc.tronclient.GetTransactionInfoById(txhash)
}
