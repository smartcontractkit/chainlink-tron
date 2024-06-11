package testutils

import (
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
)

type MockClient struct {
	client.GrpcClient
	estimateEnergyResp  api.EstimateEnergyMessage
	getEnergyPricesResp api.PricesResponseMessage
	triggerContractResp api.TransactionExtention
	broadcastResp       api.Return
	getTxInfoByIDResp   core.TransactionInfo
}

func NewMockClient() *MockClient {
	// init default responses
	return &MockClient{
		estimateEnergyResp: api.EstimateEnergyMessage{
			Result: &api.Return{
				Result: true,
			},
			EnergyRequired: 1000,
		},
		getEnergyPricesResp: api.PricesResponseMessage{
			Prices: "0:420",
		},
		triggerContractResp: api.TransactionExtention{
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
		broadcastResp: api.Return{
			Result:  true,
			Code:    api.Return_SUCCESS,
			Message: []byte("broadcast message"),
		},
		getTxInfoByIDResp: core.TransactionInfo{
			Id: []byte("txid"),
			Receipt: &core.ResourceReceipt{
				Result: core.Transaction_Result_SUCCESS,
			},
			BlockNumber: 1,
		},
	}
}

// mock client methods

func (g *MockClient) EstimateEnergy(from, contractAddress, method, jsonString string,
	tAmount int64, tTokenID string, tTokenAmount int64) (*api.EstimateEnergyMessage, error) {
	return &g.estimateEnergyResp, nil
}

func (g *MockClient) GetEnergyPrices() (*api.PricesResponseMessage, error) {
	return &g.getEnergyPricesResp, nil
}

func (g *MockClient) TriggerContract(from, contractAddress, method, jsonString string,
	feeLimit, tAmount int64, tTokenID string, tTokenAmount int64) (*api.TransactionExtention, error) {
	return &g.triggerContractResp, nil
}

func (g *MockClient) Broadcast(tx *core.Transaction) (*api.Return, error) {
	if !g.broadcastResp.GetResult() {
		return &g.broadcastResp, fmt.Errorf("result error: %s", g.broadcastResp.GetMessage())
	}
	if g.broadcastResp.GetCode() != api.Return_SUCCESS {
		return &g.broadcastResp, fmt.Errorf("result error(%s): %s", g.broadcastResp.GetCode(), g.broadcastResp.GetMessage())
	}
	return &g.broadcastResp, nil
}

func (g *MockClient) GetTransactionInfoByID(id string) (*core.TransactionInfo, error) {
	return &g.getTxInfoByIDResp, nil
}

// response setters

func (g *MockClient) SetEstimateEnergyResp(result bool, energyRequired int64) {
	g.estimateEnergyResp = api.EstimateEnergyMessage{
		Result:         &api.Return{Result: result},
		EnergyRequired: energyRequired,
	}
}

func (g *MockClient) SetGetEnergyPricesResp(prices string) {
	g.getEnergyPricesResp = api.PricesResponseMessage{
		Prices: prices,
	}
}

func (g *MockClient) SetTriggerContractResp(result bool, energyUsed int64) {
	g.triggerContractResp = api.TransactionExtention{
		Result:     &api.Return{Result: result},
		EnergyUsed: energyUsed,
	}
}

func (g *MockClient) SetBroadcastResp(result bool, code api.ReturnResponseCode, message []byte) {
	g.broadcastResp = api.Return{
		Result:  result,
		Code:    code,
		Message: message,
	}
}

func (g *MockClient) SetGetTxInfoByIDResp(result core.Transaction_ResultContractResult, blockNumber int64) {
	g.getTxInfoByIDResp = core.TransactionInfo{
		Id:          []byte("txid"),
		Receipt:     &core.ResourceReceipt{Result: result},
		BlockNumber: blockNumber,
	}
}
