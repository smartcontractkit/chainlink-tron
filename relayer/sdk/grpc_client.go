package sdk

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

//go:generate mockery --name GrpcClient --output ../mocks/
type GrpcClient interface {
	Start(opts ...grpc.DialOption) error
	Stop()
	Transfer(from, toAddress string, amount int64) (*api.TransactionExtention, error)
	GetAccount(addr string) (*core.Account, error)
	GetEnergyPrices() (*api.PricesResponseMessage, error)
	GetTransactionInfoByID(id string) (*core.TransactionInfo, error)
	DeployContract(from, contractName string,
		abi *core.SmartContract_ABI, codeStr string,
		feeLimit, curPercent, oeLimit int64,
	) (*api.TransactionExtention, error)
	Broadcast(tx *core.Transaction) (*api.Return, error)
	EstimateEnergy(from, contractAddress, method string, params []any,
		tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error)
	TriggerContract(from, contractAddress, method string, params []any,
		feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error)
	TriggerConstantContract(from, contractAddress, method string, params []any) (*api.TransactionExtention, error)
	GetNowBlock() (*api.BlockExtention, error)
	GetContractABI(address string) (*core.SmartContract_ABI, error)
	GetBlockByNum(num int64) (*api.BlockExtention, error)
	
}

var _ GrpcClient = &client.GrpcClient{}

func CreateGrpcClient(grpcUrl *url.URL) (*client.GrpcClient, error) {
	return CreateGrpcClientWithTimeout(grpcUrl, 15*time.Second)
}

func CreateGrpcClientWithTimeout(grpcUrl *url.URL, timeout time.Duration) (*client.GrpcClient, error) {
	// TODO: check scheme
	hostname := grpcUrl.Hostname()
	port := grpcUrl.Port()
	if port == "" {
		port = "50051"
	}

	insecureTransport := false
	values := grpcUrl.Query()
	insecureValues, ok := values["insecure"]
	if ok {
		if len(insecureValues) > 0 {
			insecureValue := strings.ToLower(insecureValues[0])
			insecureTransport = insecureValue == "true" || insecureValue == "1"
		}
	}

	var transportCredentials credentials.TransportCredentials
	if insecureTransport {
		transportCredentials = insecure.NewCredentials()
	} else {
		transportCredentials = credentials.NewTLS(nil)
	}

	grpcClient := client.NewGrpcClientWithTimeout(hostname+":"+port, timeout)
	err := grpcClient.Start(grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return nil, fmt.Errorf("failed to start GrpcClient: %+w", err)
	}

	return grpcClient, nil
}
