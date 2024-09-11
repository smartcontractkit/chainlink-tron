package solidityclient

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronSolidityClient) GetNowBlock() (*api.Block, error) {
	block := api.Block{}
	getNowBlockEndpoint := "/walletsolidity/getnowblock"

	err := tc.get(tc.baseURL+getNowBlockEndpoint, &block)
	if err != nil {
		return nil, fmt.Errorf("get latest block request (%s) failed: %w", tc.baseURL+getNowBlockEndpoint, err)
	}

	return &block, nil
}

func (tc *TronSolidityClient) GetBlockByNum(num int32) (*api.Block, error) {
	block := api.Block{}
	getBlockByNumEndpoint := "/walletsolidity/getblockbynum"

	err := tc.post(tc.baseURL+getBlockByNumEndpoint,
		&api.GetBlockByNumRequest{
			Num: num,
		}, &block)
	if err != nil {
		return nil, fmt.Errorf("get block by num request (%s) failed: %w", tc.baseURL+getBlockByNumEndpoint, err)
	}

	return &block, nil
}
