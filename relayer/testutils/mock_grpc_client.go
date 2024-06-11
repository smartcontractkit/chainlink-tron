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

type triggerContractResp struct {
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

type MockClient struct {
	client.GrpcClient
	estimateEnergyResp  estimateEnergyResp
	getEnergyPricesResp getEnergyPricesResp
	triggerContractResp triggerContractResp
	broadcastResp       broadcastResp
	getTxInfoByIDResp   getTxInfoByIDResp
}

func NewMockClient() *MockClient {
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
		triggerContractResp: triggerContractResp{
			res: &api.TransactionExtention{
				Transaction: &core.Transaction{
					RawData: &core.TransactionRaw{
						Timestamp:    123,
						Expiration:   456,
						RefBlockHash: []byte("abc"),
						FeeLimit:     789,
					},
				},
				Txid:           []byte("txid"),
				ConstantResult: [][]byte{},
				Result:         &api.Return{},
				EnergyUsed:     1000,
			},
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

func (g *MockClient) Broadcast(tx *core.Transaction) (*api.Return, error) {
	return g.broadcastResp.res, g.broadcastResp.err
}

func (g *MockClient) GetTransactionInfoByID(id string) (*core.TransactionInfo, error) {
	return g.getTxInfoByIDResp.res, g.getTxInfoByIDResp.err
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
	g.triggerContractResp = triggerContractResp{
		res: &api.TransactionExtention{
			Result:     &api.Return{Result: result},
			EnergyUsed: energyUsed,
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
