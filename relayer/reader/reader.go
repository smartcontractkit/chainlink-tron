package reader

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"fmt"

	tronabi "github.com/fbsobreira/gotron-sdk/pkg/abi"
	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
)

//go:generate mockery --name Reader --output ../mocks/
type Reader interface {
	CallContract(tronaddress.Address, string, []map[string]string) (map[string]interface{}, error)
	LatestBlockHeight() (blockHeight uint64, err error)
	GetEventsFromBlock(address tronaddress.Address, eventName string, blockNum uint64) ([]map[string]interface{}, error)

	BaseClient() sdk.GrpcClient
}

var _ Reader = (*ReaderClient)(nil)

type ReaderClient struct {
	rpc  sdk.GrpcClient
	lggr logger.Logger
	abi  map[string]*core.SmartContract_ABI
}

func NewReader(rpc sdk.GrpcClient, lggr logger.Logger) *ReaderClient {
	return &ReaderClient{
		rpc:  rpc,
		lggr: lggr,
		abi:  map[string]*core.SmartContract_ABI{},
	}
}

func (c *ReaderClient) BaseClient() sdk.GrpcClient {
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
	methodSignature, err := relayer.GetFunctionSignature(abi, method)
	if err != nil {
		return result, fmt.Errorf("failed to get method sighash: %w", err)
	}

	// call triggerconstantcontract
	res, err := c.rpc.TriggerConstantContract(
		/* from= */ sdk.TRON_ZERO_ADDR_B58,
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

func (c *ReaderClient) GetEventsFromBlock(address tronaddress.Address, eventName string, blockNum uint64) ([]map[string]interface{}, error) {
	// get abi
	abi, err := c.getContractABI(address)
	if err != nil {
		c.lggr.Error(fmt.Errorf("failed to get contract abi: %w", err))
		return nil, err
	}

	// get event topic hash
	eventSignature, err := relayer.GetFunctionSignature(abi, eventName)
	if err != nil {
		c.lggr.Error(fmt.Errorf("failed to get event signature: %w", err))
		return nil, err
	}
	eventTopicHash := relayer.GetEventTopicHash(eventSignature)

	// get block
	block, err := c.rpc.GetBlockByNum(int64(blockNum))
	if err != nil {
		c.lggr.Error(fmt.Errorf("failed to get block by number: %w", err))
		return nil, err
	}

	// iterate over transactions
	eventLogs := []*core.TransactionInfo_Log{}
	for _, tx := range block.Transactions {
		contract := tx.Transaction.RawData.Contract
		// This should be exactly 1 for any contract transaction.
		if contract == nil || len(contract) < 1 {
			continue
		}
		if contract[0].Parameter.TypeUrl != "type.googleapis.com/protocol.TriggerSmartContract" {
			continue
		}
		triggerSmartContract := &core.TriggerSmartContract{}
		if err := contract[0].Parameter.UnmarshalTo(triggerSmartContract); err != nil {
			c.lggr.Error(fmt.Sprintf("failed to unmarshal TriggerSmartContract transaction %s", hex.EncodeToString(tx.Txid)))
			continue
		}

		if !bytes.Equal(address.Bytes(), triggerSmartContract.ContractAddress) {
			continue
		}

		transactionInfo, err := c.rpc.GetTransactionInfoByID(hex.EncodeToString(tx.Txid))
		if err != nil {
			c.lggr.Error(fmt.Errorf("failed to fetch transaction info: %w", err))
			continue
		}

		for _, log := range transactionInfo.Log {
			// TODO: do we need this check since we already checked contract aaddress above?
			if !bytes.Equal(log.Address, address.Bytes()) {
				continue
			}
			// check first topic in log against event topic hash
			if len(log.Topics) == 0 || !bytes.Equal(log.Topics[0], eventTopicHash) {
				continue
			}
			eventLogs = append(eventLogs, log)
		}
	}

	parser, err := tronabi.GetInputsParser(abi, eventName)
	if err != nil {
		return nil, fmt.Errorf("failed to get input parser for event %s: %w", eventName, err)
	}

	var events = []map[string]interface{}{}
	for _, log := range eventLogs {
		event := make(map[string]interface{})
		err = parser.UnpackIntoMap(event, log.Data)
		if err != nil {
			return nil, fmt.Errorf("failed to unpack event log: %w", err)
		}
		events = append(events, event)
	}

	return events, nil
}
