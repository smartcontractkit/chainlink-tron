package testutils

import (
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

type estimateEnergyResp struct {
	res *api.EstimateEnergyMessage
	err error
}

type getEnergyPricesResp struct {
	res *api.PricesResponseMessage
	err error
}

type txExtentionResp struct {
	res *api.TransactionExtention
	err error
}

type broadcastResp struct {
	res *api.Return
	err error
}

type getTxInfoByIDResp struct {
	res *core.TransactionInfo
	err error
}

type getBlockResp struct {
	res *api.BlockExtention
	err error
}

type getContractAbiResp struct {
	res *core.SmartContract_ABI
	err error
}

type MockClient struct {
	client.GrpcClient
	estimateEnergyResp          estimateEnergyResp
	getEnergyPricesResp         getEnergyPricesResp
	triggerContractResp         txExtentionResp
	triggerConstantContractResp txExtentionResp
	broadcastResp               broadcastResp
	getTxInfoByIDResp           getTxInfoByIDResp
	getNowBlockResp             getBlockResp
	getContractAbiResp          getContractAbiResp
	getBlockByNumResp           getBlockResp
}

func NewMockClient() *MockClient {
	defaultTxExtention := api.TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Timestamp:    123,
				Expiration:   456,
				RefBlockHash: []byte("abc"),
				FeeLimit:     789,
			},
		},
		Txid:           []byte("txid"),
		ConstantResult: [][]byte{{0x01}},
		Result:         &api.Return{Result: true},
		EnergyUsed:     1000,
		Logs: []*core.TransactionInfo_Log{
			{
				Address: []byte{0, 1, 2, 3},
				Topics:  [][]byte{{0x02}},
				Data:    []byte("data"),
			},
		},
	}
	defaultBlockExtention := api.BlockExtention{
		BlockHeader: &core.BlockHeader{
			RawData: &core.BlockHeaderRaw{
				Number: 1,
			},
		},
		Transactions: []*api.TransactionExtention{
			&defaultTxExtention,
		},
	}

	// init default responses
	return &MockClient{
		estimateEnergyResp: estimateEnergyResp{
			res: &api.EstimateEnergyMessage{
				Result: &api.Return{
					Result: true,
				},
				EnergyRequired: 1000,
			},
			err: nil,
		},
		getEnergyPricesResp: getEnergyPricesResp{
			res: &api.PricesResponseMessage{
				Prices: "0:420",
			},
			err: nil,
		},
		triggerContractResp: txExtentionResp{
			res: &defaultTxExtention,
			err: nil,
		},
		triggerConstantContractResp: txExtentionResp{
			res: &defaultTxExtention,
			err: nil,
		},
		broadcastResp: broadcastResp{
			res: &api.Return{
				Result:  true,
				Code:    api.Return_SUCCESS,
				Message: []byte("broadcast message"),
			},
			err: nil,
		},
		getTxInfoByIDResp: getTxInfoByIDResp{
			res: &core.TransactionInfo{
				Id: []byte("txid"),
				Receipt: &core.ResourceReceipt{
					Result: core.Transaction_Result_SUCCESS,
				},
				BlockNumber: 1,
			},
			err: nil,
		},
		getNowBlockResp: getBlockResp{
			res: &defaultBlockExtention,
			err: nil,
		},
		getBlockByNumResp: getBlockResp{
			res: &defaultBlockExtention,
			err: nil,
		},
		getContractAbiResp: getContractAbiResp{
			res: &core.SmartContract_ABI{
				Entrys: []*core.SmartContract_ABI_Entry{
					{
						Name: "foo",
						Type: core.SmartContract_ABI_Entry_Function,
						Inputs: []*core.SmartContract_ABI_Entry_Param{
							{
								Name: "a",
								Type: "uint64",
							},
							{
								Name: "b",
								Type: "uint64",
							},
						},
						Outputs: []*core.SmartContract_ABI_Entry_Param{
							{
								Name: "a",
								Type: "uint64",
							},
							{
								Name: "b",
								Type: "uint64",
							},
						},
					},
				},
			},
			err: nil,
		},
	}
}

// mock client methods

func (g *MockClient) EstimateEnergy(from, contractAddress, method, jsonString string,
	tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error) {
	return g.estimateEnergyResp.res, g.estimateEnergyResp.err
}

func (g *MockClient) GetEnergyPrices() (*api.PricesResponseMessage, error) {
	return g.getEnergyPricesResp.res, g.getEnergyPricesResp.err
}

func (g *MockClient) TriggerContract(from, contractAddress, method, jsonString string,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	return g.triggerContractResp.res, g.triggerContractResp.err
}

func (g *MockClient) TriggerConstantContract(from, contractAddress, method, jsonString string) (*api.TransactionExtention, error) {
	return g.triggerConstantContractResp.res, g.triggerConstantContractResp.err
}

func (g *MockClient) Broadcast(tx *core.Transaction) (*api.Return, error) {
	return g.broadcastResp.res, g.broadcastResp.err
}

func (g *MockClient) GetTransactionInfoByID(id string) (*core.TransactionInfo, error) {
	return g.getTxInfoByIDResp.res, g.getTxInfoByIDResp.err
}

func (g *MockClient) GetNowBlock() (*api.BlockExtention, error) {
	return g.getNowBlockResp.res, g.getNowBlockResp.err
}

func (g *MockClient) GetBlockByNum(num int64) (*api.BlockExtention, error) {
	return g.getBlockByNumResp.res, g.getBlockByNumResp.err
}

func (g *MockClient) GetContractABI(address string) (*core.SmartContract_ABI, error) {
	return g.getContractAbiResp.res, g.getContractAbiResp.err
}

// response setters

func (g *MockClient) SetEstimateEnergyResp(result bool, energyRequired int64, err error) {
	g.estimateEnergyResp = estimateEnergyResp{
		res: &api.EstimateEnergyMessage{
			Result:         &api.Return{Result: result},
			EnergyRequired: energyRequired,
		},
		err: err,
	}
}

func (g *MockClient) SetGetEnergyPricesResp(prices string, err error) {
	g.getEnergyPricesResp = getEnergyPricesResp{
		res: &api.PricesResponseMessage{
			Prices: prices,
		},
		err: err,
	}
}

func (g *MockClient) SetTriggerContractResp(result bool, energyUsed int64, err error) {
	g.triggerContractResp = txExtentionResp{
		res: &api.TransactionExtention{
			Result:     &api.Return{Result: result},
			EnergyUsed: energyUsed,
		},
		err: err,
	}
}

func (g *MockClient) SetTriggerConstantContractResp(constantResult [][]byte, err error) {
	g.triggerConstantContractResp = txExtentionResp{
		res: &api.TransactionExtention{
			Result:         &api.Return{Result: true},
			ConstantResult: constantResult,
		},
		err: err,
	}
}

func (g *MockClient) SetBroadcastResp(result bool, code api.ReturnResponseCode, message []byte, err error) {
	g.broadcastResp = broadcastResp{
		res: &api.Return{
			Result:  result,
			Code:    code,
			Message: message,
		},
		err: err,
	}
}

func (g *MockClient) SetGetTxInfoByIDResp(result core.Transaction_ResultContractResult, blockNumber int64, err error) {
	g.getTxInfoByIDResp = getTxInfoByIDResp{
		res: &core.TransactionInfo{
			Id:          []byte("txid"),
			Receipt:     &core.ResourceReceipt{Result: result},
			BlockNumber: blockNumber,
		},
		err: err,
	}
}

func (g *MockClient) SetGetNowBlockResp(blockNumber int64, err error) {
	g.getNowBlockResp = getBlockResp{
		res: &api.BlockExtention{
			BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{
					Number: blockNumber,
				},
			},
		},
		err: err,
	}
}

func (g *MockClient) SetGetBlockByNumResp(txExtention *api.TransactionExtention, err error) {
	g.getBlockByNumResp = getBlockResp{
		res: &api.BlockExtention{
			BlockHeader:  g.getNowBlockResp.res.BlockHeader,
			Transactions: append(g.getNowBlockResp.res.Transactions, txExtention),
		},
		err: err,
	}
}

func (g *MockClient) SetGetContractAbiResp(abi *core.SmartContract_ABI, err error) {
	g.getContractAbiResp = getContractAbiResp{
		res: abi,
		err: err,
	}
}
