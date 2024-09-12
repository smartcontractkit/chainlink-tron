package fullnodeclient

import (
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

func (tc *TronFullNodeClient) DeployContract(reqBody *api.DeployContractRequest) (*api.Transaction, error) {
	return tc.tronclient.DeployContract(reqBody)
}

func (tc *TronFullNodeClient) GetContract(address string) (*api.GetContractResponse, error) {
	return tc.tronclient.GetContract(address)
}

func (tc *TronFullNodeClient) TriggerSmartContract(tcRequest *api.TriggerSmartContractRequest) (*api.TriggerSmartContractResponse, error) {
	return tc.tronclient.TriggerSmartContract(tcRequest)
}

func (tc *TronFullNodeClient) TriggerConstantContract(tcRequest *api.TriggerConstantContractRequest) (*api.TriggerConstantContractResponse, error) {
	return tc.tronclient.TriggerConstantContract(tcRequest)
}

func (tc *TronFullNodeClient) BroadcastTransaction(reqBody *api.Transaction) (*api.BroadcastResponse, error) {
	return tc.tronclient.BroadcastTransaction(reqBody)
}

func (tc *TronFullNodeClient) EstimateEnergy(reqBody *api.EnergyEstimateRequest) (*api.EnergyEstimateResult, error) {
	return tc.tronclient.EstimateEnergy(reqBody)
}
