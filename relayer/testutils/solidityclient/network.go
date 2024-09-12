package solidityclient

import (
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronSolidityClient) GetNowBlock() (*api.Block, error) {
	return tc.tronclient.GetNowBlock()
}

func (tc *TronSolidityClient) GetBlockByNum(num int32) (*api.Block, error) {
	return tc.tronclient.GetBlockByNum(num)
}
