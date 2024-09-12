package httpclient

import (
	"fmt"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (thc *TronHttpClient) DeployContract(reqBody *api.DeployContractRequest) (*api.Transaction, error) {
	transaction := api.Transaction{}
	deployEndpoint := thc.urlPrefix + "/deploycontract"

	err := thc.post(deployEndpoint, reqBody, &transaction)
	if err != nil {
		return nil, fmt.Errorf("deploy contract request (%s) failed: %w", deployEndpoint, err)
	}

	return &transaction, nil
}

func (thc *TronHttpClient) GetContract(address string) (*api.GetContractResponse, error) {

	getContractEndpoint := thc.urlPrefix + "/getcontract"
	var contractInfo api.GetContractResponse

	err := thc.post(getContractEndpoint,
		&api.GetContractRequest{
			Value:   address,
			Visible: true,
		}, &contractInfo)

	if err != nil {
		return nil, fmt.Errorf("get contract request (%s) failed: %w", getContractEndpoint, err)
	}

	if len(contractInfo.ContractAddress) < 1 {
		return nil, fmt.Errorf("get contract failed: contract address empty")
	}

	return &contractInfo, nil
}

func (thc *TronHttpClient) TriggerSmartContract(tcRequest *api.TriggerSmartContractRequest) (*api.TriggerSmartContractResponse, error) {
	triggerContractEndpoint := thc.urlPrefix + "/triggersmartcontract"
	contractResponse := api.TriggerSmartContractResponse{}

	err := thc.post(triggerContractEndpoint, tcRequest, &contractResponse)
	if err != nil {
		return nil, fmt.Errorf("trigger smart contract request (%s) failed: %w", triggerContractEndpoint, err)
	}

	return &contractResponse, nil
}

func (thc *TronHttpClient) TriggerConstantContract(tcRequest *api.TriggerConstantContractRequest) (*api.TriggerConstantContractResponse, error) {
	triggerContractEndpoint := thc.urlPrefix + "/triggerconstantcontract"
	contractResponse := api.TriggerConstantContractResponse{}

	err := thc.post(triggerContractEndpoint, tcRequest, &contractResponse)
	if err != nil {
		return nil, fmt.Errorf("trigger constant contract request (%s) failed: %w", triggerContractEndpoint, err)
	}

	return &contractResponse, nil
}

func (thc *TronHttpClient) BroadcastTransaction(reqBody *api.Transaction) (*api.BroadcastResponse, error) {
	response := api.BroadcastResponse{}
	broadcastEndpoint := thc.urlPrefix + "/broadcasttransaction"

	err := thc.post(broadcastEndpoint, reqBody, &response)

	if err != nil {
		return nil, fmt.Errorf("broadcast transaction request (%s) failed: %w", broadcastEndpoint, err)
	}

	if !response.Result {
		return nil, fmt.Errorf("broadcasting failed. Code: %s, Message: %s", response.Code, response.Message)
	}

	return &response, nil
}

func (thc *TronHttpClient) EstimateEnergy(reqBody *api.EnergyEstimateRequest) (*api.EnergyEstimateResult, error) {
	response := api.EnergyEstimateResult{}
	energyEstimateEndpoint := thc.urlPrefix + "/estimateenergy"

	err := thc.post(energyEstimateEndpoint, reqBody, &response)

	if err != nil {
		return nil, fmt.Errorf("energy estimation request (%s) failed: %w", energyEstimateEndpoint, err)
	}

	return &response, nil
}
