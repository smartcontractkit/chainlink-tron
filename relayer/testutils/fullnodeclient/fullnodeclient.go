package fullnodeclient

import (
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/httpclient"
)

type TronFullNodeClient struct {
	tronclient *httpclient.TronHttpClient
}

func NewClient(baseURL string, client httpclient.HttpClient) *TronFullNodeClient {
	urlprefix := baseURL + "/wallet"
	tronclient := httpclient.NewTronHttpClient(urlprefix, client)
	return &TronFullNodeClient{
		tronclient: tronclient,
	}
}
