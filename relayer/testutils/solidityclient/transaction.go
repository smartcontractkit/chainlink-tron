package solidityclient

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronSolidityClient) GetTransactionInfoById(txhash string) (*api.TransactionInfo, error) {
	transactionInfo := api.TransactionInfo{}
	getTransactionInfoByIdEndpoint := "/walletsolidity/gettransactioninfobyid"

	err := tc.post(tc.baseURL+getTransactionInfoByIdEndpoint,
		&api.GetTransactionInfoByIDRequest{
			Value: txhash,
		}, &transactionInfo)

	if err != nil {
		return nil, fmt.Errorf("get transaction info by id request (%s) failed: %w", tc.baseURL+getTransactionInfoByIdEndpoint, err)
	}

	return &transactionInfo, nil
}
