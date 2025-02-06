package sdk

import (
	"net/url"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/fullnode"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
)

//go:generate mockery --name CombinedClient --output ../mocks/
type CombinedClient interface {
	FullNodeClient
	TriggerConstantContractFullNode(from, contractAddress address.Address, method string, params []any) (*soliditynode.TriggerConstantContractResponse, error)
	GetNowBlockFullNode() (*soliditynode.Block, error)
	GetBlockByNumFullNode(num int32) (*soliditynode.Block, error)
	GetAccountFullNode(accountAddress address.Address) (*soliditynode.GetAccountResponse, error)
	GetTransactionInfoByIdFullNode(txhash string) (*soliditynode.TransactionInfo, error)

	FullNodeClient() *fullnode.Client
	SolidityClient() *soliditynode.Client
}

type combinedClient struct {
	*fullnode.Client
	solidityClient *soliditynode.Client
}

func NewCombinedClient(fullnodeClient *fullnode.Client, solidityClient *soliditynode.Client) CombinedClient {
	return &combinedClient{
		Client:         fullnodeClient,
		solidityClient: solidityClient,
	}
}

func CreateCombinedClient(fullnodeUrl, soliditynodeUrl *url.URL) (CombinedClient, error) {
	return CreateCombinedClientWithTimeout(fullnodeUrl, soliditynodeUrl, 15*time.Second)
}

func CreateCombinedClientWithTimeout(fullnodeUrl, soliditynodeUrl *url.URL, timeout time.Duration) (CombinedClient, error) {
	httpClient := CreateHttpClientWithTimeout(timeout)
	fullnodeClient := fullnode.NewClient(fullnodeUrl.String(), httpClient)
	soliditynodeClient := soliditynode.NewClient(soliditynodeUrl.String(), httpClient)

	return NewCombinedClient(fullnodeClient, soliditynodeClient), nil
}

func (g *combinedClient) FullNodeClient() *fullnode.Client {
	return g.Client
}

func (g *combinedClient) SolidityClient() *soliditynode.Client {
	return g.solidityClient
}

// We manually override methods that we want to use the solidity client for (all read methods).
// We also provide the fullnode versions of these methods for flexibility

// GetAccount from BASE58 address using solidity client
func (g *combinedClient) GetAccount(accountAddress address.Address) (*soliditynode.GetAccountResponse, error) {
	return g.solidityClient.GetAccount(accountAddress)
}

// GetAccount from BASE58 address using fullnode client
func (g *combinedClient) GetAccountFullNode(accountAddress address.Address) (*soliditynode.GetAccountResponse, error) {
	return g.Client.GetAccount(accountAddress)
}

// GetTransactionInfoByID returns transaction receipt by ID using solidity client
func (g *combinedClient) GetTransactionInfoById(txhash string) (*soliditynode.TransactionInfo, error) {
	return g.solidityClient.GetTransactionInfoById(txhash)
}

// GetTransactionInfoByID returns transaction receipt by ID using fullnode client
func (g *combinedClient) GetTransactionInfoByIdFullNode(txhash string) (*soliditynode.TransactionInfo, error) {
	return g.Client.GetTransactionInfoById(txhash)
}

// TriggerConstantContract and return tx result using solidity client
func (g *combinedClient) TriggerConstantContract(from, contractAddress address.Address, method string, params []any) (*soliditynode.TriggerConstantContractResponse, error) {
	return g.solidityClient.TriggerConstantContract(from, contractAddress, method, params)
}

// TriggerConstantContract and return tx result using solidity client
func (g *combinedClient) TriggerConstantContractFullNode(from, contractAddress address.Address, method string, params []any) (*soliditynode.TriggerConstantContractResponse, error) {
	return g.Client.TriggerConstantContract(from, contractAddress, method, params)
}

// GetNowBlock return TIP block using solidity client
func (g *combinedClient) GetNowBlock() (*soliditynode.Block, error) {
	return g.solidityClient.GetNowBlock()
}

// GetNowBlock return TIP block using fullnode client
func (g *combinedClient) GetNowBlockFullNode() (*soliditynode.Block, error) {
	return g.Client.GetNowBlock()
}

// GetBlockByNum block from number using solidity client
func (g *combinedClient) GetBlockByNum(num int32) (*soliditynode.Block, error) {
	return g.solidityClient.GetBlockByNum(num)
}

// GetBlockByNum block from number using fullnode client
func (g *combinedClient) GetBlockByNumFullNode(num int32) (*soliditynode.Block, error) {
	return g.Client.GetBlockByNum(num)
}
