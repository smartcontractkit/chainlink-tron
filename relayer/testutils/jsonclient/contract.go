package jsonclient

import (
	"fmt"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronJsonClient) DeployContract(reqBody *api.DeployContractRequest) (*apiTransaction, error) {
	transaction := api.Transaction{}
	deployEndpoint := "/wallet/deploycontract"

	err := tc.post(tc.baseURL+deployEndpoint, reqBody, &transaction)
	if err != nil {
		return nil, fmt.Errorf("deploy contract request (%s) failed: %w", tc.baseURL+deployEndpoint, err)
	}

	return &transaction, nil
}

func (tc *TronJsonClient) GetContract(address string) (*api.GetContractResponse, error) {

	getContractEndpoint := "/wallet/getcontract"
	var contractInfo api.GetContractResponse

	err := tc.post(tc.baseURL+getContractEndpoint,
		&api.GetContractRequest{
			Value:   address,
			Visible: true,
		}, &contractInfo)

	if err != nil {
		return nil, fmt.Errorf("get contract request (%s) failed: %w", tc.baseURL+getContractEndpoint, err)
	}

	if len(contractInfo.ContractAddress) < 1 {
		return nil, fmt.Errorf("get contract failed: contract address empty")
	}

	return &contractInfo, nil
}

func (tc *TronJsonClient) TriggerSmartContract(tcRequest *api.TriggerSmartContractRequest) (*api.TriggerSmartContractResponse, error) {
	triggerContractEndpoint := "/wallet/triggersmartcontract"
	contractResponse := api.TriggerSmartContractResponse{}

	err := tc.post(tc.baseURL+triggerContractEndpoint, tcRequest, &contractResponse)
	if err != nil {
		return nil, fmt.Errorf("trigger smart contract request (%s) failed: %w", tc.baseURL+triggerContractEndpoint, err)

	}

	return &contractResponse, nil
}

func (tc *TronJsonClient) TriggerConstantContract(tcRequest *api.TriggerConstantContractRequest) (*api.TriggerConstantContractResponse, error) {
	triggerContractEndpoint := "/wallet/triggerconstantcontract"
	contractResponse := api.TriggerConstantContractResponse{}

	err := tc.post(tc.baseURL+triggerContractEndpoint, tcRequest, &contractResponse)
	if err != nil {
		return nil, fmt.Errorf("trigger constant contract request (%s) failed: %w", tc.baseURL+triggerContractEndpoint, err)
	}

	return &contractResponse, nil
}

func (tc *TronJsonClient) BroadcastTransaction(reqBody *api.Transaction) (*api.BroadcastResponse, error) {
	response := api.BroadcastResponse{}
	broadcastEndpoint := "/wallet/broadcasttransaction"

	err := tc.post(tc.baseURL+broadcastEndpoint, reqBody, &response)

	if err != nil {
		return nil, fmt.Errorf("broadcast transaction request (%s) failed: %w", tc.baseURL+broadcastEndpoint, err)
	}

	if !response.Result {
		return nil, fmt.Errorf("broadcasting failed. Code: %s, Message: %s", response.Code, response.Message)
	}

	return &response, nil
}

func (tc *TronJsonClient) EstimateEnergy(reqBody *api.EnergyEstimateRequest) (*api.EnergyEstimateResult, error) {
	response := api.EnergyEstimateResult{}
	energyEstimateEndpoint := "/wallet/estimateenergy"

	err := tc.post(tc.baseURL+energyEstimateEndpoint, reqBody, &response)

	if err != nil {
		return nil, fmt.Errorf("energy estimation request (%s) failed: %w", tc.baseURL+energyEstimateEndpoint, err)
	}

	return &response, nil
}
