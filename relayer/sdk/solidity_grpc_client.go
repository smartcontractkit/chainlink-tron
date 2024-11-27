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
	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

// TODO: move into gotron-sdk
type SolidityGrpcClient struct {
	client      api.WalletSolidityClient
	grpcTimeout time.Duration
}

func CreateSolidityGrpcClient(grpcUrl *url.URL) (*SolidityGrpcClient, error) {
	return CreateSolidityGrpcClientWithTimeout(grpcUrl, 15*time.Second)
}

func CreateSolidityGrpcClientWithTimeout(solidityGrpcUrl *url.URL, grpcTimeout time.Duration) (*SolidityGrpcClient, error) {
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

	return &SolidityGrpcClient{
		client:      solidityClient,
		grpcTimeout: grpcTimeout,
	}, nil
}

func (g *SolidityGrpcClient) getContext() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithTimeout(context.Background(), g.grpcTimeout)
	return ctx, cancel
}

// GetAccount from BASE58 address
func (g *SolidityGrpcClient) GetAccount(addr string) (*core.Account, error) {
	account := new(core.Account)
	var err error

	account.Address, err = common.DecodeCheck(addr)
	if err != nil {
		return nil, err
	}

	ctx, cancel := g.getContext()
	defer cancel()

	acc, err := g.client.GetAccount(ctx, account)
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(acc.Address, account.Address) {
		return nil, fmt.Errorf("account not found")
	}
	return acc, nil
}

// GetEnergyPrices retrieves energy prices
func (g *SolidityGrpcClient) GetEnergyPrices() (*api.PricesResponseMessage, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	result, err := g.client.GetEnergyPrices(ctx, new(api.EmptyMessage))
	if err != nil {
		return nil, fmt.Errorf("get energy prices: %v", err)
	}

	return result, nil
}

// GetTransactionInfoByID returns transaction receipt by ID
func (g *SolidityGrpcClient) GetTransactionInfoByID(id string) (*core.TransactionInfo, error) {
	transactionID := new(api.BytesMessage)
	var err error

	transactionID.Value, err = common.FromHex(id)
	if err != nil {
		return nil, fmt.Errorf("get transaction by id error: %v", err)
	}

	ctx, cancel := g.getContext()
	defer cancel()

	txi, err := g.client.GetTransactionInfoById(ctx, transactionID)
	if err != nil {
		return nil, err
	}
	if bytes.Equal(txi.Id, transactionID.Value) {
		return txi, nil
	}
	return nil, fmt.Errorf("transaction info not found")
}

// TriggerConstantContract and return tx result
func (g *SolidityGrpcClient) TriggerConstantContract(from, contractAddress, method string, params []any) (*api.TransactionExtention, error) {
	var err error
	fromDesc := address.ZeroAddress
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

	return g.client.TriggerConstantContract(ctx, ct)
}

// GetNowBlock return TIP block
func (g *SolidityGrpcClient) GetNowBlock() (*api.BlockExtention, error) {
	ctx, cancel := g.getContext()
	defer cancel()

	result, err := g.client.GetNowBlock2(ctx, new(api.EmptyMessage))

	if err != nil {
		return nil, fmt.Errorf("Get block now: %v", err)
	}

	return result, nil
}

// GetBlockByNum block from number
func (g *SolidityGrpcClient) GetBlockByNum(num int64) (*api.BlockExtention, error) {
	numMessage := new(api.NumberMessage)
	numMessage.Num = num

	ctx, cancel := g.getContext()
	defer cancel()

	maxSizeOption := grpc.MaxCallRecvMsgSize(32 * 10e6)
	result, err := g.client.GetBlockByNum2(ctx, numMessage, maxSizeOption)

	if err != nil {
		return nil, fmt.Errorf("Get block by num: %v", err)

	}
	return result, nil
}

// Balance returns TRX balance of address
func (g *SolidityGrpcClient) GetAccountBalance(address tronaddress.Address) (int64, error) {
	account, err := g.GetAccount(address.String())
	if err != nil {
		return 0, fmt.Errorf("failed to get account: %w", err)
	}

	return account.GetBalance(), nil
}
