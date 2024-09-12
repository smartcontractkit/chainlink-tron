package httpclient

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (thc *TronHttpClient) GetEnergyPrices() (*api.EnergyPrice, error) {
	energyPrices := api.EnergyPrice{}
	getEnergyPricesEndpoint := thc.urlPrefix + "/getenergyprices"

	err := thc.get(getEnergyPricesEndpoint, &energyPrices)
	if err != nil {
		return nil, fmt.Errorf("get energy prices request (%s) failed: %w", getEnergyPricesEndpoint, err)
	}
	return &energyPrices, nil
}

func (thc *TronHttpClient) GetNowBlock() (*api.Block, error) {
	block := api.Block{}
	getNowBlockEndpoint := thc.urlPrefix + "/getnowblock"

	err := thc.post(getNowBlockEndpoint, nil, &block)
	if err != nil {
		return nil, fmt.Errorf("get latest block request (%s) failed: %w", getNowBlockEndpoint, err)
	}

	return &block, nil
}

func (thc *TronHttpClient) GetBlockByNum(num int32) (*api.Block, error) {
	block := api.Block{}
	getBlockByNumEndpoint := thc.urlPrefix + "/getblockbynum"

	err := thc.post(getBlockByNumEndpoint,
		&api.GetBlockByNumRequest{
			Num: num,
		}, &block)
	if err != nil {
		return nil, fmt.Errorf("get block by num request (%s) failed: %w", getBlockByNumEndpoint, err)
	}

	return &block, nil
}
