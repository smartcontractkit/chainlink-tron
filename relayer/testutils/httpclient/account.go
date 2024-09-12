package httpclient

import (
	"fmt"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (thc *TronHttpClient) GetAccount(address string) (*api.GetAccountResponse, error) {
	getAccountEndpoint := thc.urlPrefix + "/getaccount"
	getAccountResponse := api.GetAccountResponse{}

	err := thc.post(getAccountEndpoint, &api.GetAccountRequest{
		Address: address,
		Visible: true,
	}, &getAccountResponse)
	if err != nil {
		return nil, fmt.Errorf("get account (%s) failed: %w", getAccountEndpoint, err)
	}

	return &getAccountResponse, nil
}
