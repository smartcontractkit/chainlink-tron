package jsonclient

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronJsonClient) GetEnergyPrices() (*api.EnergyPrice, error) {
	energyPrices := api.EnergyPrice{}
	getEnergyPricesEndpoint := "/wallet/getenergyprices"

	err := tc.get(tc.baseURL+getEnergyPricesEndpoint, &energyPrices)
	if err != nil {
		return nil, fmt.Errorf("get energy prices request (%s) failed: %w", tc.baseURL+getEnergyPricesEndpoint, err)
	}
	return &energyPrices, nil
}

func (tc *TronJsonClient) GetNowBlock() (*api.Block, error) {
	block := api.Block{}
	getNowBlockEndpoint := "/wallet/getnowblock"

	err := tc.post(tc.baseURL+getNowBlockEndpoint, nil, &block)
	if err != nil {
		return nil, fmt.Errorf("get latest block request (%s) failed: %w", tc.baseURL+getNowBlockEndpoint, err)
	}

	return &block, nil
}

func (tc *TronJsonClient) GetBlockByNum(num int32) (*api.Block, error) {
	block := api.Block{}
	getBlockByNumEndpoint := "/wallet/getblockbynum"

	err := tc.post(tc.baseURL+getBlockByNumEndpoint,
		&api.GetBlockByNumRequest{
			Num: num,
		}, &block)
	if err != nil {
		return nil, fmt.Errorf("get block by num request (%s) failed: %w", tc.baseURL+getBlockByNumEndpoint, err)
	}

	return &block, nil
}
