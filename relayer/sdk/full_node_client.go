package sdk

import (
	"crypto/tls"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/fullnode"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
)

//go:generate mockery --name FullNodeClient --output ../mocks/
type FullNodeClient interface {
	TriggerConstantContract(from, contractAddress address.Address, method string, params []any) (*soliditynode.TriggerConstantContractResponse, error)
	EstimateEnergy(from, contractAddress address.Address, method string, params []any, tAmount int64) (*soliditynode.EnergyEstimateResult, error)
	GetNowBlock() (*soliditynode.Block, error)
	GetBlockByNum(num int32) (*soliditynode.Block, error)
	GetAccount(accountAddress address.Address) (*soliditynode.GetAccountResponse, error)
	GetTransactionInfoById(txhash string) (*soliditynode.TransactionInfo, error)

	DeployContract(ownerAddress address.Address, contractName, abiJson, bytecode string, oeLimit, curPercent, feeLimit int, params []interface{}) (*fullnode.DeployContractResponse, error)
	GetContract(address address.Address) (*fullnode.GetContractResponse, error)
	TriggerSmartContract(from, contractAddress address.Address, method string, params []any, feeLimit int32, tAmount int64) (*fullnode.TriggerSmartContractResponse, error)
	Transfer(fromAddress, toAddress address.Address, amount int64) (*common.Transaction, error)
	BroadcastTransaction(reqBody *common.Transaction) (*fullnode.BroadcastResponse, error)
	GetEnergyPrices() (*fullnode.EnergyPrices, error)
}

var _ FullNodeClient = &fullnode.Client{}

func CreateHttpClientWithTimeout(timeout time.Duration, insecureSkipVerify bool) *http.Client {
	// Create custom HTTP client with timeout
	return &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: insecureSkipVerify,
			},
		},
	}
}

func CreateFullNodeClient(httpUrl *url.URL) (FullNodeClient, error) {
	return CreateFullNodeClientWithTimeout(httpUrl, 15*time.Second)
}

func CreateFullNodeClientWithTimeout(httpUrl *url.URL, timeout time.Duration) (FullNodeClient, error) {
	httpClient := CreateHttpClientWithTimeout(timeout, hasInsecureFlag(httpUrl))

	// Create the client
	client := fullnode.NewClient(cleanUrlString(httpUrl), httpClient)
	return client, nil
}

func hasInsecureFlag(u *url.URL) bool {
	values := u.Query()
	insecureValues, ok := values["insecure"]
	if !ok || len(insecureValues) == 0 {
		return false
	}
	insecureValue := strings.ToLower(insecureValues[0])
	return insecureValue == "true" || insecureValue == "1"
}

// removes query params
func cleanUrlString(u *url.URL) string {
	cleanUrl := &url.URL{
		Scheme: u.Scheme,
		Host:   u.Host,
		Path:   u.Path,
	}
	return cleanUrl.String()
}
