package sdk

import (
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc"
)

//go:generate mockery --name GrpcClient --output ../mocks/
type GrpcClient interface {
	Start(opts ...grpc.DialOption) error
	Stop()
	GetEnergyPrices() (*api.PricesResponseMessage, error)
	GetTransactionInfoByID(id string) (*core.TransactionInfo, error)
	DeployContract(from, contractName string,
		abi *core.SmartContract_ABI, codeStr string,
		feeLimit, curPercent, oeLimit int64,
	) (*api.TransactionExtention, error)
	Broadcast(tx *core.Transaction) (*api.Return, error)
	EstimateEnergy(from, contractAddress, method, jsonString string,
		tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error)
	TriggerContract(from, contractAddress, method, jsonString string,
		feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error)
	TriggerConstantContract(from, contractAddress, method, jsonString string) (*api.TransactionExtention, error)
	GetNowBlock() (*api.BlockExtention, error)
	GetContractABI(address string) (*core.SmartContract_ABI, error)
	GetBlockByNum(num int64) (*api.BlockExtention, error)
}
