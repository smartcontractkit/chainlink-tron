package reader

import (
	"context"
	"encoding/hex"
	"fmt"
	"math"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-tron/relayer"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
)

//go:generate mockery --name Reader --output ../mocks/
type Reader interface {
	CallContract(ctx context.Context, contractAddress address.Address, method string, params []any) (map[string]interface{}, error)
	CallContractByAbi(ctx context.Context, abi *common.JSONABI, contractAddress address.Address, method string, params []any) (map[string]interface{}, error)
	CallContractFullNode(ctx context.Context, contractAddress address.Address, method string, params []any) (map[string]interface{}, error)
	LatestBlockHeight(ctx context.Context) (uint64, error)
	GetEventsFromBlock(ctx context.Context, contractAddress address.Address, eventName string, blockNum uint64) ([]map[string]interface{}, error)

	BaseClient() sdk.CombinedClient
}

var _ Reader = (*ReaderClient)(nil)

type ReaderClient struct {
	rpc  sdk.CombinedClient
	lggr logger.Logger
	abi  map[string]*common.JSONABI
}

func NewReader(rpc sdk.CombinedClient, lggr logger.Logger) *ReaderClient {
	return &ReaderClient{
		rpc:  rpc,
		lggr: lggr,
		abi:  map[string]*common.JSONABI{},
	}
}

func (c *ReaderClient) BaseClient() sdk.CombinedClient {
	return c.rpc
}

func (c *ReaderClient) getContractABI(ctx context.Context, contractAddress address.Address) (*common.JSONABI, error) {
	// return cached abi if cached
	if abi, ok := c.abi[contractAddress.String()]; ok {
		return abi, nil
	}

	// otherwise fetch from chain
	response, err := c.rpc.GetContract(ctx, contractAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get contract ABI: %w", err)
	}
	// cache abi for future use
	c.abi[contractAddress.String()] = response.ABI
	return response.ABI, nil
}

func (c *ReaderClient) CallContract(ctx context.Context, contractAddress address.Address, method string, params []any) (map[string]interface{}, error) {
	// get contract abi
	abi, err := c.getContractABI(ctx, contractAddress)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("error fetching abi: %w", err)
	}

	// get method signature
	methodSignature, err := abi.GetFunctionSignature(method)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to get method sighash: %w", err)
	}

	// call triggerconstantcontract
	res, err := c.rpc.TriggerConstantContract(
		ctx,
		/* from= */ address.ZeroAddress,
		/* contractAddress= */ contractAddress,
		/* method= */ methodSignature,
		/* params= */ params,
	)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to call triggerconstantcontract: %w", err)
	}
	if !res.Result.Result || len(res.ConstantResult) == 0 {
		return map[string]interface{}{}, fmt.Errorf("failed to call contract: res=%+v", res)
	}

	// parse return value
	parser, err := abi.GetOutputParser(method)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to get abi parser: %w", err)
	}
	result := map[string]interface{}{}
	constantResultBytes, err := hex.DecodeString(res.ConstantResult[0])
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to decode constant result: %w", err)
	}
	err = parser.UnpackIntoMap(result, constantResultBytes)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to unpack result: %w", err)
	}
	return result, nil
}

func (c *ReaderClient) CallContractByAbi(ctx context.Context, abi *common.JSONABI, contractAddress address.Address, method string, params []any) (map[string]interface{}, error) {
	// get method signature
	methodSignature, err := abi.GetFunctionSignature(method)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to get method sighash: %w", err)
	}

	// call triggerconstantcontract
	res, err := c.rpc.TriggerConstantContract(
		ctx,
		/* from= */ address.ZeroAddress,
		/* contractAddress= */ contractAddress,
		/* method= */ methodSignature,
		/* params= */ params,
	)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to call triggerconstantcontract: %w", err)
	}
	if !res.Result.Result || len(res.ConstantResult) == 0 {
		return map[string]interface{}{}, fmt.Errorf("failed to call contract: res=%+v", res)
	}

	// parse return value
	parser, err := abi.GetOutputParser(method)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to get abi parser: %w", err)
	}
	result := map[string]interface{}{}
	constantResultBytes, err := hex.DecodeString(res.ConstantResult[0])
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to decode constant result: %w", err)
	}
	err = parser.UnpackIntoMap(result, constantResultBytes)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to unpack result: %w", err)
	}
	return result, nil
}

// Same as CallContract, but uses the fullnode client instead of the solidity client, which means it uses the non-finalized state of the chain.
func (c *ReaderClient) CallContractFullNode(ctx context.Context, contractAddress address.Address, method string, params []any) (map[string]interface{}, error) {
	// get contract abi
	abi, err := c.getContractABI(ctx, contractAddress)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("error fetching abi: %w", err)
	}

	// get method signature
	methodSignature, err := abi.GetFunctionSignature(method)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to get method sighash: %w", err)
	}

	// call triggerconstantcontract
	res, err := c.rpc.TriggerConstantContractFullNode(
		ctx,
		/* from= */ address.ZeroAddress,
		/* contractAddress= */ contractAddress,
		/* method= */ methodSignature,
		/* params= */ params,
	)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to call triggerconstantcontract: %w", err)
	}
	if !res.Result.Result || len(res.ConstantResult) == 0 {
		return map[string]interface{}{}, fmt.Errorf("failed to call contract: res=%+v", res)
	}

	// parse return value
	parser, err := abi.GetOutputParser(method)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to get abi parser: %w", err)
	}
	result := map[string]interface{}{}
	constantResultBytes, err := hex.DecodeString(res.ConstantResult[0])
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to decode constant result: %w", err)
	}
	err = parser.UnpackIntoMap(result, constantResultBytes)
	if err != nil {
		return map[string]interface{}{}, fmt.Errorf("failed to unpack result: %w", err)
	}
	return result, nil
}

func (c *ReaderClient) LatestBlockHeight(ctx context.Context) (uint64, error) {
	nowBlock, err := c.rpc.GetNowBlock(ctx)
	if err != nil {
		return 0, fmt.Errorf("couldn't get latest block: %w", err)
	}

	return uint64(nowBlock.BlockHeader.RawData.Number), nil
}

func (c *ReaderClient) GetEventsFromBlock(ctx context.Context, contractAddress address.Address, eventName string, blockNum uint64) ([]map[string]interface{}, error) {
	// check if block number fits in int32
	if blockNum > uint64(math.MaxInt32) {
		return nil, fmt.Errorf("block number %d exceeds maximum int32 value", blockNum)
	}

	// get abi
	abi, err := c.getContractABI(ctx, contractAddress)
	if err != nil {
		c.lggr.Error(fmt.Errorf("failed to get contract abi: %w", err))
		return nil, err
	}

	// get event topic hash
	eventSignature, err := abi.GetFunctionSignature(eventName)
	if err != nil {
		c.lggr.Error(fmt.Errorf("failed to get event signature: %w", err))
		return nil, err
	}
	eventTopicHash := relayer.GetEventTopicHash(eventSignature)

	// get block
	block, err := c.rpc.GetBlockByNum(ctx, int32(blockNum))
	if err != nil {
		c.lggr.Error(fmt.Errorf("failed to get block by number: %w", err))
		return nil, err
	}

	contractAddressHex := contractAddress.Hex()

	// iterate over transactions
	eventLogs := []soliditynode.Log{}
	for _, tx := range block.Transactions {
		contract := tx.Transaction.RawData.Contract
		// This should be exactly 1 for any contract transaction.
		if contract == nil || len(contract) < 1 {
			continue
		}
		if contract[0].Parameter.TypeUrl != "type.googleapis.com/protocol.TriggerSmartContract" {
			continue
		}
		if contractAddressHex != contract[0].Parameter.Value.ContractAddress {
			continue
		}
		transactionInfo, err := c.rpc.GetTransactionInfoById(ctx, tx.TxID)
		if err != nil {
			c.lggr.Error(fmt.Errorf("failed to fetch transaction info: %w", err))
			continue
		}

		for _, log := range transactionInfo.Log {
			// we don't bother comparing log.Address since we already matched the contract address
			// before retrieving the transaction. log.Address is a string in hex, but without the 0x41
			// prefix, which is why a simple match did not work before.

			// check first topic in log against event topic hash
			if len(log.Topics) == 0 || log.Topics[0] != eventTopicHash {
				continue
			}
			eventLogs = append(eventLogs, log)
		}
	}

	parser, err := abi.GetInputParser(eventName)
	if err != nil {
		return nil, fmt.Errorf("failed to get input parser for event %s: %w", eventName, err)
	}

	var events = []map[string]interface{}{}
	for _, log := range eventLogs {
		event := make(map[string]interface{})
		dataBytes, err := hex.DecodeString(log.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to decode event data: %w", err)
		}
		err = parser.UnpackIntoMap(event, dataBytes)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack event log: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}
