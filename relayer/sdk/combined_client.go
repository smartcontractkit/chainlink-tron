package sdk

import (
	"context"
	"math/big"
	"net/url"
	"time"

	"github.com/ethereum/go-ethereum"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
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
	JsonRpcClient() EthClient
}

// EthClient interface required for mocking in tests
var _ EthClient = &ethclient.Client{}

//go:generate mockery --name EthClient --output ../mocks/
type EthClient interface {
	BalanceAt(ctx context.Context, account ethcommon.Address, blockNumber *big.Int) (*big.Int, error)
	BalanceAtHash(ctx context.Context, account ethcommon.Address, blockHash ethcommon.Hash) (*big.Int, error)
	BlockByHash(ctx context.Context, hash ethcommon.Hash) (*types.Block, error)
	BlockByNumber(ctx context.Context, number *big.Int) (*types.Block, error)
	BlockNumber(ctx context.Context) (uint64, error)
	BlockReceipts(ctx context.Context, blockNrOrHash rpc.BlockNumberOrHash) ([]*types.Receipt, error)
	CallContract(ctx context.Context, msg ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
	CallContractAtHash(ctx context.Context, msg ethereum.CallMsg, blockHash ethcommon.Hash) ([]byte, error)
	ChainID(ctx context.Context) (*big.Int, error)
	Client() *rpc.Client
	Close()
	CodeAt(ctx context.Context, account ethcommon.Address, blockNumber *big.Int) ([]byte, error)
	CodeAtHash(ctx context.Context, account ethcommon.Address, blockHash ethcommon.Hash) ([]byte, error)
	EstimateGas(ctx context.Context, msg ethereum.CallMsg) (uint64, error)
	FeeHistory(ctx context.Context, blockCount uint64, lastBlock *big.Int, rewardPercentiles []float64) (*ethereum.FeeHistory, error)
	FilterLogs(ctx context.Context, q ethereum.FilterQuery) ([]types.Log, error)
	HeaderByHash(ctx context.Context, hash ethcommon.Hash) (*types.Header, error)
	HeaderByNumber(ctx context.Context, number *big.Int) (*types.Header, error)
	NetworkID(ctx context.Context) (*big.Int, error)
	NonceAt(ctx context.Context, account ethcommon.Address, blockNumber *big.Int) (uint64, error)
	NonceAtHash(ctx context.Context, account ethcommon.Address, blockHash ethcommon.Hash) (uint64, error)
	PeerCount(ctx context.Context) (uint64, error)
	PendingBalanceAt(ctx context.Context, account ethcommon.Address) (*big.Int, error)
	PendingCallContract(ctx context.Context, msg ethereum.CallMsg) ([]byte, error)
	PendingCodeAt(ctx context.Context, account ethcommon.Address) ([]byte, error)
	PendingNonceAt(ctx context.Context, account ethcommon.Address) (uint64, error)
	PendingStorageAt(ctx context.Context, account ethcommon.Address, key ethcommon.Hash) ([]byte, error)
	PendingTransactionCount(ctx context.Context) (uint, error)
	SendTransaction(ctx context.Context, tx *types.Transaction) error
	StorageAt(ctx context.Context, account ethcommon.Address, key ethcommon.Hash, blockNumber *big.Int) ([]byte, error)
	StorageAtHash(ctx context.Context, account ethcommon.Address, key ethcommon.Hash, blockHash ethcommon.Hash) ([]byte, error)
	SubscribeFilterLogs(ctx context.Context, q ethereum.FilterQuery, ch chan<- types.Log) (ethereum.Subscription, error)
	SubscribeNewHead(ctx context.Context, ch chan<- *types.Header) (ethereum.Subscription, error)
	SuggestGasPrice(ctx context.Context) (*big.Int, error)
	SuggestGasTipCap(ctx context.Context) (*big.Int, error)
	SyncProgress(ctx context.Context) (*ethereum.SyncProgress, error)
	TransactionByHash(ctx context.Context, hash ethcommon.Hash) (tx *types.Transaction, isPending bool, err error)
	TransactionCount(ctx context.Context, blockHash ethcommon.Hash) (uint, error)
	TransactionInBlock(ctx context.Context, blockHash ethcommon.Hash, index uint) (*types.Transaction, error)
	TransactionReceipt(ctx context.Context, txHash ethcommon.Hash) (*types.Receipt, error)
	TransactionSender(ctx context.Context, tx *types.Transaction, block ethcommon.Hash, index uint) (ethcommon.Address, error)
}

type combinedClient struct {
	*fullnode.Client
	solidityClient *soliditynode.Client
	jsonRpcClient  EthClient
}

func NewCombinedClient(fullnodeClient *fullnode.Client, solidityClient *soliditynode.Client, jsonRpcClient EthClient) CombinedClient {
	return &combinedClient{
		Client:         fullnodeClient,
		solidityClient: solidityClient,
		jsonRpcClient:  jsonRpcClient,
	}
}

func CreateCombinedClient(fullnodeUrl, soliditynodeUrl, jsonrpcUrl *url.URL) (CombinedClient, error) {
	return CreateCombinedClientWithTimeout(fullnodeUrl, soliditynodeUrl, jsonrpcUrl, 15*time.Second)
}

func CreateCombinedClientWithTimeout(fullnodeUrl, soliditynodeUrl, jsonrpcUrl *url.URL, timeout time.Duration) (CombinedClient, error) {
	httpClient := CreateHttpClientWithTimeout(timeout)
	fullnodeClient := fullnode.NewClient(fullnodeUrl.String(), httpClient)
	soliditynodeClient := soliditynode.NewClient(soliditynodeUrl.String(), httpClient)
	jsonRpcClient, err := ethclient.Dial(jsonrpcUrl.String())
	if err != nil {
		return nil, err
	}

	return NewCombinedClient(fullnodeClient, soliditynodeClient, jsonRpcClient), nil
}

func (g *combinedClient) FullNodeClient() *fullnode.Client {
	return g.Client
}

func (g *combinedClient) SolidityClient() *soliditynode.Client {
	return g.solidityClient
}

func (g *combinedClient) JsonRpcClient() EthClient {
	return g.jsonRpcClient
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
