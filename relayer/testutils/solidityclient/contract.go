package solidityclient

import (
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronSolidityClient) TriggerConstantContract(tcRequest *api.TriggerConstantContractRequest) (*api.TriggerConstantContractResponse, error) {
	return tc.tronclient.TriggerConstantContract(tcRequest)
}

func (tc *TronSolidityClient) EstimateEnergy(reqBody *api.EnergyEstimateRequest) (*api.EnergyEstimateResult, error) {
	return tc.tronclient.EstimateEnergy(reqBody)
}
