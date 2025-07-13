package sdk

import (
	"context"
	"net/url"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/fullnode"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
)

//go:generate mockery --name CombinedClient --output ../mocks/
type CombinedClient interface {
	FullNodeClient
	TriggerConstantContractFullNode(ctx context.Context, from, contractAddress address.Address, method string, params []any) (*soliditynode.TriggerConstantContractResponse, error)
	GetNowBlockFullNode(ctx context.Context) (*soliditynode.Block, error)
	GetBlockByNumFullNode(ctx context.Context, num int32) (*soliditynode.Block, error)
	GetAccountFullNode(ctx context.Context, accountAddress address.Address) (*soliditynode.GetAccountResponse, error)
	GetTransactionInfoByIdFullNode(ctx context.Context, txhash string) (*soliditynode.TransactionInfo, error)

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
func (g *combinedClient) GetAccount(ctx context.Context, accountAddress address.Address) (*soliditynode.GetAccountResponse, error) {
	return g.solidityClient.GetAccount(ctx, accountAddress)
}

// GetAccount from BASE58 address using fullnode client
func (g *combinedClient) GetAccountFullNode(ctx context.Context, accountAddress address.Address) (*soliditynode.GetAccountResponse, error) {
	return g.Client.GetAccount(ctx, accountAddress)
}

// GetTransactionInfoByID returns transaction receipt by ID using solidity client
func (g *combinedClient) GetTransactionInfoById(ctx context.Context, txhash string) (*soliditynode.TransactionInfo, error) {
	return g.solidityClient.GetTransactionInfoById(ctx, txhash)
}

// GetTransactionInfoByID returns transaction receipt by ID using fullnode client
func (g *combinedClient) GetTransactionInfoByIdFullNode(ctx context.Context, txhash string) (*soliditynode.TransactionInfo, error) {
	return g.Client.GetTransactionInfoById(ctx, txhash)
}

// TriggerConstantContract and return tx result using solidity client
func (g *combinedClient) TriggerConstantContract(ctx context.Context, from, contractAddress address.Address, method string, params []any) (*soliditynode.TriggerConstantContractResponse, error) {
	return g.solidityClient.TriggerConstantContract(ctx, from, contractAddress, method, params)
}

// TriggerConstantContract and return tx result using solidity client
func (g *combinedClient) TriggerConstantContractFullNode(ctx context.Context, from, contractAddress address.Address, method string, params []any) (*soliditynode.TriggerConstantContractResponse, error) {
	return g.Client.TriggerConstantContract(ctx, from, contractAddress, method, params)
}

// GetNowBlock return TIP block using solidity client
func (g *combinedClient) GetNowBlock(ctx context.Context) (*soliditynode.Block, error) {
	return g.solidityClient.GetNowBlock(ctx)
}

// GetNowBlock return TIP block using fullnode client
func (g *combinedClient) GetNowBlockFullNode(ctx context.Context) (*soliditynode.Block, error) {
	return g.Client.GetNowBlock(ctx)
}

// GetBlockByNum block from number using solidity client
func (g *combinedClient) GetBlockByNum(ctx context.Context, num int32) (*soliditynode.Block, error) {
	return g.solidityClient.GetBlockByNum(ctx, num)
}

// GetBlockByNum block from number using fullnode client
func (g *combinedClient) GetBlockByNumFullNode(ctx context.Context, num int32) (*soliditynode.Block, error) {
	return g.Client.GetBlockByNum(ctx, num)
}
