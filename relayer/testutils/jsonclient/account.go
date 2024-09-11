package jsonclient

import (
	"fmt"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronJsonClient) GetAccount(address string) (*api.GetAccountResponse, error) {
	getAccountEndpoint := "/wallet/getaccount"
	getAccountResponse := api.GetAccountResponse{}

	err := tc.post(tc.baseURL+getAccountEndpoint, &api.GetAccountRequest{
		Address: address,
		Visible: true,
	}, &getAccountResponse)
	if err != nil {
		return nil, fmt.Errorf("get account (%s) failed: %w", tc.baseURL+getAccountEndpoint, err)
	}

	return &getAccountResponse, nil
}
