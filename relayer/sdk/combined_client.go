package sdk

import (
	"net/url"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/fullnode"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
)

var _ FullNodeClient = &CombinedClient{}

type CombinedClient struct {
	*fullnode.Client
	SolidityClient *soliditynode.Client
}

func NewCombinedClient(fullnodeClient *fullnode.Client, solidityClient *soliditynode.Client) *CombinedClient {
	return &CombinedClient{
		Client:         fullnodeClient,
		SolidityClient: solidityClient,
	}
}

func CreateCombinedClient(fullnodeUrl, soliditynodeUrl *url.URL) (*CombinedClient, error) {
	return CreateCombinedClientWithTimeout(fullnodeUrl, soliditynodeUrl, 15*time.Second)
}

func CreateCombinedClientWithTimeout(fullnodeUrl, soliditynodeUrl *url.URL, timeout time.Duration) (*CombinedClient, error) {
	httpClient := CreateHttpClientWithTimeout(timeout)
	fullnodeClient := fullnode.NewClient(fullnodeUrl.String(), httpClient)
	soliditynodeClient := soliditynode.NewClient(soliditynodeUrl.String(), httpClient)

	return NewCombinedClient(fullnodeClient, soliditynodeClient), nil
}

// We manually override methods that we want to use the solidity client for (all read methods).

// GetAccount from BASE58 address
func (g *CombinedClient) GetAccount(accountAddress address.Address) (*soliditynode.GetAccountResponse, error) {
	return g.SolidityClient.GetAccount(accountAddress)
}

// GetTransactionInfoByID returns transaction receipt by ID
func (g *CombinedClient) GetTransactionInfoById(txhash string) (*soliditynode.TransactionInfo, error) {
	return g.SolidityClient.GetTransactionInfoById(txhash)
}

// TriggerConstantContract and return tx result
func (g *CombinedClient) TriggerConstantContract(from, contractAddress address.Address, method string, params []any) (*soliditynode.TriggerConstantContractResponse, error) {
	return g.SolidityClient.TriggerConstantContract(from, contractAddress, method, params)
}

// GetNowBlock return TIP block
func (g *CombinedClient) GetNowBlock() (*soliditynode.Block, error) {
	return g.SolidityClient.GetNowBlock()
}

// GetBlockByNum block from number
func (g *CombinedClient) GetBlockByNum(num int32) (*soliditynode.Block, error) {
	return g.SolidityClient.GetBlockByNum(num)
}
