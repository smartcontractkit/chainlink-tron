package httpclient

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (thc *TronHttpClient) GetTransactionInfoById(txhash string) (*api.TransactionInfo, error) {
	transactionInfo := api.TransactionInfo{}
	getTransactionInfoByIdEndpoint := thc.urlPrefix + "/gettransactioninfobyid"

	err := thc.post(getTransactionInfoByIdEndpoint,
		&api.GetTransactionInfoByIDRequest{
			Value: txhash,
		}, &transactionInfo)

	if err != nil {
		return nil, fmt.Errorf("get transaction info by id request (%s) failed: %w", getTransactionInfoByIdEndpoint, err)
	}

	return &transactionInfo, nil
}
