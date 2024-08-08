package sdk

import (
	"net/url"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

var _ GrpcClient = &CombinedGrpcClient{}

type CombinedGrpcClient struct {
	*client.GrpcClient
	SolidityClient *SolidityGrpcClient
}

func NewCombinedGrpcClient(grpcClient *client.GrpcClient, solidityClient *SolidityGrpcClient) *CombinedGrpcClient {
	return &CombinedGrpcClient{
		GrpcClient:     grpcClient,
		SolidityClient: solidityClient,
	}
}

func CreateCombinedGrpcClient(grpcUrl, solidityGrpcUrl *url.URL) (*CombinedGrpcClient, error) {
	return CreateCombinedGrpcClientWithTimeout(grpcUrl, solidityGrpcUrl, 15*time.Second)
}

func CreateCombinedGrpcClientWithTimeout(grpcUrl, solidityGrpcUrl *url.URL, timeout time.Duration) (*CombinedGrpcClient, error) {
	fullClient, err := CreateGrpcClientWithTimeout(grpcUrl, timeout)
	if err != nil {
		return nil, err
	}

	solidityClient, err := CreateSolidityGrpcClientWithTimeout(solidityGrpcUrl, timeout)
	if err != nil {
		return nil, err
	}

	return &CombinedGrpcClient{
		GrpcClient:     fullClient,
		SolidityClient: solidityClient,
	}, nil
}

// We manually override methods that we want to use the solidity client for (all read methods).

// GetAccount from BASE58 address
func (g *CombinedGrpcClient) GetAccount(addr string) (*core.Account, error) {
	return g.SolidityClient.GetAccount(addr)
}

// GetEnergyPrices retrieves energy prices
func (g *CombinedGrpcClient) GetEnergyPrices() (*api.PricesResponseMessage, error) {
	return g.SolidityClient.GetEnergyPrices()
}

// GetTransactionInfoByID returns transaction receipt by ID
func (g *CombinedGrpcClient) GetTransactionInfoByID(id string) (*core.TransactionInfo, error) {
	return g.SolidityClient.GetTransactionInfoByID(id)
}

// TriggerConstantContract and return tx result
func (g *CombinedGrpcClient) TriggerConstantContract(from, contractAddress, method string, params []any) (*api.TransactionExtention, error) {
	return g.SolidityClient.TriggerConstantContract(from, contractAddress, method, params)
}

// GetNowBlock return TIP block
func (g *CombinedGrpcClient) GetNowBlock() (*api.BlockExtention, error) {
	return g.SolidityClient.GetNowBlock()
}

// GetBlockByNum block from number
func (g *CombinedGrpcClient) GetBlockByNum(num int64) (*api.BlockExtention, error) {
	return g.SolidityClient.GetBlockByNum(num)
}
