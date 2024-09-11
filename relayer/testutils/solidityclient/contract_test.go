package solidityclient

import (
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils/api"
)

var expectedEstimateEnergy = &api.EnergyEstimateResult{
	Result: api.ReturnEnergyEstimate{
		Result: true,
	},
	EnergyRequired: 1082,
}

var expectedTriggerConstantContract = &api.TriggerConstantContractResponse{
	Result: api.TriggerConstantContractResult{
		Result: true,
	},
	EnergyUsed: 541,
	ConstantResult: []string{
		"00000000000000000000000000000000000000000000000000000001663ea8d6",
	},
	Transaction: api.TriggerConstantTransaction{
		RawData: api.TriggerRawData{
			Expiration: 1724412825000,
			Timestamp:  1724412768022,
			Contract: []api.TriggerContract{
				{
					Parameter: api.TriggerParameter{
						Value: api.TriggerValue{
							Data:            "70a08231000000000000000000000000a614f803b6fd780986a42c78ec9c7f77e6ded13c",
							OwnerAddress:    "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
							ContractAddress: "TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs",
						},
						TypeUrl: "type.googleapis.com/protocol.TriggerSmartContract",
					},
					Type: "TriggerSmartContract",
				},
			},
			RefBlockBytes: "2adf",
			RefBlockHash:  "62a72c7aa9af9c65",
		},
		RawDataHex: "0a022adf220862a72c7aa9af9c6540a89b8ef897325a8e01081f1289010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412540a1541fd49eda0f23ff7ec1d03b52c3a45991c24cd440e12154142a1e39aefa49290f2b3f9ed688d7cecf86cd6e0222470a08231000000000000000000000000a614f803b6fd780986a42c78ec9c7f77e6ded13c7096de8af89732",
		Ret: []api.ConstantRet{
			{},
		},
		Visible: true,
		TxID:    "bf5d8b1917f8c6d31793ac0c0428b23a0a711143860fcf9be34b53801fd17454",
	},
}
