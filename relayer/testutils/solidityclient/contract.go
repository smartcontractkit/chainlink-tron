package solidityclient

import (
	"fmt"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronSolidityClient) TriggerConstantContract(tcRequest *api.TriggerConstantContractRequest) (*api.TriggerConstantContractResponse, error) {
	triggerContractEndpoint := "/walletsolidity/triggerconstantcontract"
	contractResponse := api.TriggerConstantContractResponse{}

	err := tc.post(tc.baseURL+triggerContractEndpoint, tcRequest, &contractResponse)
	if err != nil {
		return nil, fmt.Errorf("trigger constant contract request (%s) failed: %w", tc.baseURL+triggerContractEndpoint, err)
	}

	return &contractResponse, nil
}

func (tc *TronSolidityClient) EstimateEnergy(reqBody *api.EnergyEstimateRequest) (*api.EnergyEstimateResult, error) {
	response := api.EnergyEstimateResult{}
	energyEstimateEndpoint := "/walletsolidity/estimateenergy"

	err := tc.post(tc.baseURL+energyEstimateEndpoint, reqBody, &response)

	if err != nil {
		return nil, fmt.Errorf("energy estimation request (%s) failed: %w", tc.baseURL+energyEstimateEndpoint, err)
	}

	return &response, nil
}
