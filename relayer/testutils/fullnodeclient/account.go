package fullnodeclient

import (
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronFullNodeClient) GetAccount(address string) (*api.GetAccountResponse, error) {
	return tc.tronclient.GetAccount(address)
}
