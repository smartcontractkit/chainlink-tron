package relayer

import (
	"bytes"
	"encoding/json"
	"fmt"

	tronabi "github.com/fbsobreira/gotron-sdk/pkg/abi"
	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
)

// Tron zero address - https://developers.tron.network/docs/faq#3-what-is-the-destruction-address-of-tron
const (
	TRON_ZERO_ADDR_B58 = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"
	TRON_ZERO_ADDR_HEX = "410000000000000000000000000000000000000000"
)

type Reader interface {
	CallContract(tronaddress.Address, string, []map[string]string) (map[string]interface{}, error)
	LatestBlockHeight() (blockHeight uint64, err error)
	GetEventLogsFromBlock(address tronaddress.Address, eventType string, blockNum uint64) ([]*core.TransactionInfo_Log, error)

	BaseClient() GrpcClient
}

var _ Reader = (*ReaderClient)(nil)

type ReaderClient struct {
	rpc  GrpcClient
	lggr logger.Logger
	abi  map[string]*core.SmartContract_ABI
}

func NewReader(rpc GrpcClient, lggr logger.Logger) *ReaderClient {
	return &ReaderClient{
		rpc:  rpc,
		lggr: lggr,
		abi:  map[string]*core.SmartContract_ABI{},
	}
}

func (c *ReaderClient) BaseClient() GrpcClient {
	return c.rpc
}

func (c *ReaderClient) getContractABI(address tronaddress.Address) (abi *core.SmartContract_ABI, err error) {
	// return cached abi if cached
	if abi, ok := c.abi[address.String()]; ok {
		return abi, nil
	}

	// otherwise fetch from chain
	abi, err = c.rpc.GetContractABI(address.String())
	if err != nil {
		return nil, fmt.Errorf("failed to get contract ABI: %w", err)
	}
	// cache abi for future use
	c.abi[address.String()] = abi

	return
}

func (c *ReaderClient) CallContract(address tronaddress.Address, method string, params []map[string]string) (result map[string]interface{}, err error) {
	// parse params if defined
	paramsJsonStr := ""
	if len(params) > 0 {
		paramsJsonBytes, err := json.Marshal(params)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal params: %+w", err)
		}

		paramsJsonStr = string(paramsJsonBytes)
	}

	// get contract abi
	abi, err := c.getContractABI(address)
	if err != nil {
		return result, fmt.Errorf("error fetching abi: %w", err)
	}

	// get method signature
	methodSignature, err := GetFunctionSignature(abi, method)
	if err != nil {
		return result, fmt.Errorf("failed to get method sighash: %w", err)
	}

	// call triggerconstantcontract
	res, err := c.rpc.TriggerConstantContract(
		/* from= */ TRON_ZERO_ADDR_B58,
		/* contractAddress= */ address.String(),
		/* method= */ methodSignature,
		/* jsonString= */ paramsJsonStr,
	)
	if err != nil {
		return result, fmt.Errorf("failed to call triggerconstantcontract: %w", err)
	}
	if !res.Result.Result || len(res.ConstantResult) == 0 {
		return result, fmt.Errorf("failed to call contract: res=%+v", res)
	}

	// parse return value
	parser, err := tronabi.GetParser(abi, method)
	if err != nil {
		return result, fmt.Errorf("failed to get abi parser: %w", err)
	}
	result = map[string]interface{}{}
	err = parser.UnpackIntoMap(result, res.ConstantResult[0])
	if err != nil {
		return result, fmt.Errorf("failed to unpack result: %w", err)
	}
	return
}

func (c *ReaderClient) LatestBlockHeight() (blockHeight uint64, err error) {
	nowBlock, err := c.rpc.GetNowBlock()
	if err != nil {
		return blockHeight, fmt.Errorf("couldn't get latest block: %w", err)
	}

	return uint64(nowBlock.GetBlockHeader().GetRawData().Number), nil
}

func (c *ReaderClient) GetEventLogsFromBlock(address tronaddress.Address, eventType string, blockNum uint64) ([]*core.TransactionInfo_Log, error) {
	// get abi
	abi, err := c.getContractABI(address)
	if err != nil {
		c.lggr.Error(fmt.Errorf("failed to get contract abi: %w", err))
		return nil, err
	}

	// get event topic hash
	eventSignature, err := GetFunctionSignature(abi, eventType)
	if err != nil {
		c.lggr.Error(fmt.Errorf("failed to get event signature: %w", err))
		return nil, err
	}
	eventTopicHash := GetEventTopicHash(eventSignature)

	// get block
	block, err := c.rpc.GetBlockByNum(int64(blockNum))
	if err != nil {
		c.lggr.Error(fmt.Errorf("failed to get block by number: %w", err))
		return nil, err
	}

	// iterate over transactions
	var eventLogs []*core.TransactionInfo_Log
	for _, tx := range block.Transactions {
		for _, log := range tx.Logs {
			// check log address matches contract address
			if !bytes.Equal(log.Address, address.Bytes()) {
				continue
			}
			// check first topic in log against event topic hash
			if !bytes.Equal(log.Topics[0], eventTopicHash) {
				continue
			}
			eventLogs = append(eventLogs, log)
		}
	}

	// todo: parse event logs into struct rather than returning raw logs
	return eventLogs, nil
}
