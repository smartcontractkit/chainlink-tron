package sdk

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
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
var _ GrpcClient = &combinedGrpcClient{}

type combinedGrpcClient struct {
	*client.GrpcClient
	SolidityClient api.WalletSolidityClient
	grpcTimeout    time.Duration
}

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

func CreateCombinedGrpcClient(grpcUrl, solidityGrpcUrl *url.URL) (*combinedGrpcClient, error) {
	return CreateCombinedGrpcClientWithTimeout(grpcUrl, solidityGrpcUrl, 15*time.Second)
}

func CreateCombinedGrpcClientWithTimeout(grpcUrl, solidityGrpcUrl *url.URL, timeout time.Duration) (*combinedGrpcClient, error) {
	fullClient, err := CreateGrpcClientWithTimeout(grpcUrl, timeout)
	if err != nil {
		return nil, err
	}
	// build solidity wallet client
	solidityHostname := solidityGrpcUrl.Hostname()
	solidityPort := solidityGrpcUrl.Port()

	insecureTransport := false
	values := solidityGrpcUrl.Query()
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

	solidityConn, err := grpc.Dial(solidityHostname+":"+solidityPort, grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return nil, fmt.Errorf("failed to init solidity wallet connection: %v", err)
	}
	solidityClient := api.NewWalletSolidityClient(solidityConn)

	return &combinedGrpcClient{
		GrpcClient:     fullClient,
		SolidityClient: solidityClient,
		grpcTimeout:    timeout,
	}, nil
}

// TODO: We manually override methods that we want to use the solidity client for (all read methods).
// These are largely a copy paste from gotron-sdk so we may want to move these over in the future.

func (g *combinedGrpcClient) getContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), g.grpcTimeout)
	return ctx, cancel
}

// GetAccount from BASE58 address
func (g *combinedGrpcClient) GetAccount(addr string) (*core.Account, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	acc, err := g.SolidityClient.GetAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(acc.Address, account.Address) {
		return nil, fmt.Errorf("account not found")
	}
	return acc, nil
}

// GetEnergyPrices retrieves energy prices
func (g *combinedGrpcClient) GetEnergyPrices() (*api.PricesResponseMessage, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	result, err := g.SolidityClient.GetEnergyPrices(ctx, new(api.EmptyMessage))
	if err != nil {
		return nil, fmt.Errorf("get energy prices: %v", err)
	}

	return result, nil
}

// GetTransactionInfoByID returns transaction receipt by ID
func (g *combinedGrpcClient) GetTransactionInfoByID(id string) (*core.TransactionInfo, error) {
	transactionID := new(api.BytesMessage)
	var err error

	transactionID.Value, err = common.FromHex(id)
	if err != nil {
		return nil, fmt.Errorf("get transaction by id error: %v", err)
	}

	ctx, cancel := g.getContext()
	defer cancel()

	txi, err := g.SolidityClient.GetTransactionInfoById(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(txi.Id, transactionID.Value) {
		return txi, nil
	}
	return nil, fmt.Errorf("transaction info not found")
}

// TriggerConstantContract and return tx result
func (g *combinedGrpcClient) TriggerConstantContract(from, contractAddress, method string, params []any) (*api.TransactionExtention, error) {
	var err error
	fromDesc := address.HexToAddress("410000000000000000000000000000000000000000")
	if len(from) > 0 {
		fromDesc, err = address.Base58ToAddress(from)
		if err != nil {
			return nil, err
		}
	}
	contractDesc, err := address.Base58ToAddress(contractAddress)
	if err != nil {
		return nil, err
	}

	dataBytes, err := abi.Pack(method, params)
	if err != nil {
		return nil, err
	}

	ct := &core.TriggerSmartContract{
		OwnerAddress:    fromDesc.Bytes(),
		ContractAddress: contractDesc.Bytes(),
		Data:            dataBytes,
	}

	ctx, cancel := g.getContext()
	defer cancel()

	return g.SolidityClient.TriggerConstantContract(ctx, ct)
}

// GetNowBlock return TIP block
func (g *combinedGrpcClient) GetNowBlock() (*api.BlockExtention, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	result, err := g.SolidityClient.GetNowBlock2(ctx, new(api.EmptyMessage))

	if err != nil {
		return nil, fmt.Errorf("Get block now: %v", err)
	}

	return result, nil
}

// GetBlockByNum block from number
func (g *combinedGrpcClient) GetBlockByNum(num int64) (*api.BlockExtention, error) {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	ctx, cancel := g.getContext()
	defer cancel()

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	result, err := g.SolidityClient.GetBlockByNum2(ctx, numMessage, maxSizeOption)

	if err != nil {
		return nil, fmt.Errorf("Get block by num: %v", err)

	}
	return result, nil
}
