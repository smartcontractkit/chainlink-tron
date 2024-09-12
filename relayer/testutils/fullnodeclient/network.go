package fullnodeclient

import (
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronFullNodeClient) GetEnergyPrices() (*api.EnergyPrice, error) {
	return tc.tronclient.GetEnergyPrices()
}

func (tc *TronFullNodeClient) GetNowBlock() (*api.Block, error) {
	return tc.tronclient.GetNowBlock()
}

func (tc *TronFullNodeClient) GetBlockByNum(num int32) (*api.Block, error) {
	return tc.tronclient.GetBlockByNum(num)
}
