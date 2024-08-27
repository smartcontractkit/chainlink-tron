package jsonclient

import "fmt"

type DeployContractRequest struct {
	OwnerAddress               string `json:"owner_address"`
	ABI                        string `json:"abi"`
	Bytecode                   string `json:"bytecode"`
	Parameter                  string `json:"parameter"`
	Name                       string `json:"name"`
	CallValue                  int    `json:"call_value"`
	FeeLimit                   int    `json:"fee_limit"`
	ConsumeUserResourcePercent int    `json:"consume_user_resource_percent"`
	OriginEnergyLimit          int    `json:"origin_energy_limit"`
	Visible                    bool   `json:"visible"`
}

type JSONABI struct {
	Entrys []struct {
		Anonymous bool `json:"anonymous"`
		Constant  bool `json:"constant"`
		Inputs    []struct {
			Indexed bool   `json:"indexed"`
			Name    string `json:"name"`
			Type    string `json:"type"`
		} `json:"inputs"`
		Name    string `json:"name"`
		Outputs []struct {
			Indexed bool   `json:"indexed"`
			Name    string `json:"name"`
			Type    string `json:"type"`
		} `json:"outputs"`
		Payable         bool   `json:"payable"`
		StateMutability string `json:"stateMutability"`
		Type            string `json:"type"`
	} `json:"entrys"`
}

type NewContract struct {
	OriginAddress              string  `json:"origin_address"`                // Contract creator address
	ContractAddress            string  `json:"contract_address"`              // Contract address
	ABI                        JSONABI `json:"abi"`                           // ABI
	Bytecode                   string  `json:"bytecode"`                      // Bytecode
	CallValue                  int64   `json:"call_value"`                    // The amount of TRX passed into the contract when deploying the contract
	ConsumeUserResourcePercent int64   `json:"consume_user_resource_percent"` // Proportion of user energy consumption
	Name                       string  `json:"name"`                          // contract name
	OriginEnergyLimit          int64   `json:"origin_energy_limit"`           // Each transaction is allowed to consume the maximum energy of the contract creator
	CodeHash                   string  `json:"code_hash"`                     // code hash
}

type ParameterValue struct {
	OwnerAddress string      `json:"owner_address"`
	NewContract  NewContract `json:"new_contract"`
}

type Parameter struct {
	Value   ParameterValue `json:"value"`
	TypeUrl string         `json:"type_url"`
}

type Contract struct {
	Parameter Parameter `json:"parameter"`
	Type      string    `json:"type"`
}

type RawData struct {
	Contract      []Contract `json:"contract,omitempty"`
	RefBlockBytes string     `json:"ref_block_bytes,omitempty"`
	RefBlockHash  string     `json:"ref_block_hash,omitempty"`
	Expiration    int64      `json:"expiration,omitempty"`
	FeeLimit      int64      `json:"fee_limit,omitempty"`
	Timestamp     int64      `json:"timestamp,omitempty"`
}

type Transaction struct {
	Visible    bool     `json:"visible"`
	TxID       string   `json:"txID"`
	RawData    RawData  `json:"raw_data"`
	RawDataHex string   `json:"raw_data_hex"`
	Signature  []string `json:"signature"`
}

func (tc *TronJsonClient) DeployContract(reqBody *DeployContractRequest) (*Transaction, error) {
	transaction := Transaction{}
	deployEndpoint := "/wallet/deploycontract"

	err := tc.post(tc.baseURL+deployEndpoint, reqBody, &transaction)
	if err != nil {
		return nil, fmt.Errorf("deploy contract request (%s) failed: %w", tc.baseURL+deployEndpoint, err)
	}

	return &transaction, nil
}

type GetContractRequest struct {
	Value   string `json:"value"`
	Visible bool   `json:"visible"`
}

type GetContractResponse struct {
	OriginAddress              string  `json:"origin_address"`                // Contract creator address
	ContractAddress            string  `json:"contract_address"`              // Contract address
	ABI                        JSONABI `json:"abi"`                           // ABI
	Bytecode                   string  `json:"bytecode"`                      // Bytecode
	CallValue                  int64   `json:"call_value"`                    // The amount of TRX passed into the contract when deploying the contract
	ConsumeUserResourcePercent int64   `json:"consume_user_resource_percent"` // Proportion of user energy consumption
	Name                       string  `json:"name"`                          // contract name
	OriginEnergyLimit          int64   `json:"origin_energy_limit"`           // Each transaction is allowed to consume the maximum energy of the contract creator
	CodeHash                   string  `json:"code_hash"`                     // code hash
}

func (tc *TronJsonClient) GetContract(address string) (*GetContractResponse, error) {

	getContractEndpoint := "/wallet/getcontract"
	var contractInfo GetContractResponse

	err := tc.post(tc.baseURL+getContractEndpoint,
		&GetContractRequest{
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

type TriggerSmartContractRequest struct {
	OwnerAddress     string `json:"owner_address"`     // Address that triggers the contract, converted to a hex string
	ContractAddress  string `json:"contract_address"`  // Contract address, converted to a hex string
	FunctionSelector string `json:"function_selector"` // Function call, must not be left blank
	Parameter        string `json:"parameter"`
	Data             string `json:"data"`             // The data for interacting with smart contracts, including the contract function and parameters
	FeeLimit         int32  `json:"fee_limit"`        // Maximum TRX consumption, measured in SUN
	CallValue        int64  `json:"call_value"`       // Amount of TRX transferred with this transaction, measured in SUN
	CallTokenValue   int64  `json:"call_token_value"` // Amount of TRC10 token transferred with this transaction
	TokenId          int64  `json:"token_id"`         // TRC 10 token id
	// typo in spec? json:"Permission_id" https://developers.tron.network/reference/triggersmartcontract
	PermissionId int32 `json:"permission_id"` // for multi-signature
	Visible      bool  `json:"visible"`       // Whether the address is in base58check format
}

type TriggerResult struct {
	Result bool `json:"result"`
}

type TriggerValue struct {
	Data            string `json:"data"`
	OwnerAddress    string `json:"owner_address"`
	ContractAddress string `json:"contract_address"`
}

type TriggerParameter struct {
	Value   TriggerValue `json:"value"`
	TypeUrl string       `json:"type_url"`
}

type TriggerContract struct {
	Parameter TriggerParameter `json:"parameter"`
	Type      string           `json:"type"`
}

type TriggerRawData struct {
	Contract      []TriggerContract `json:"contract,omitempty"`
	RefBlockBytes string            `json:"ref_block_bytes,omitempty"`
	RefBlockHash  string            `json:"ref_block_hash,omitempty"`
	Expiration    int64             `json:"expiration,omitempty"`
	FeeLimit      int64             `json:"fee_limit,omitempty"`
	Timestamp     int64             `json:"timestamp,omitempty"`
}

type TriggerTransaction struct {
	Visible    bool           `json:"visible"`
	TxID       string         `json:"txID"`
	RawData    TriggerRawData `json:"raw_data"`
	RawDataHex string         `json:"raw_data_hex"`
}

type TriggerSmartContractResponse struct {
	Result      TriggerResult      `json:"result"`
	Transaction TriggerTransaction `json:"transaction"`
}

func (tc *TronJsonClient) TriggerSmartContract(tcRequest *TriggerSmartContractRequest) (*TriggerSmartContractResponse, error) {
	triggerContractEndpoint := "/wallet/triggersmartcontract"
	contractResponse := TriggerSmartContractResponse{}

	err := tc.post(tc.baseURL+triggerContractEndpoint, tcRequest, &contractResponse)
	if err != nil {
		return nil, fmt.Errorf("trigger smart contract request (%s) failed: %w", tc.baseURL+triggerContractEndpoint, err)

	}

	return &contractResponse, nil
}

type TriggerConstantContractRequest struct {
	OwnerAddress     string `json:"owner_address"`     // Owner address that triggers the contract. If visible=true, use base58check format, otherwise use hex format
	ContractAddress  string `json:"contract_address"`  // Smart contract address. If visible=true, use base58check format, otherwise use hex format
	FunctionSelector string `json:"function_selector"` // Function call, must not be left blank
	Parameter        string `json:"parameter"`
	Data             string `json:"data"`             // The bytecode of the contract or the data for interacting with smart contracts, including the contract function and parameters
	CallValue        int64  `json:"call_value"`       // Amount of TRX transferred with this transaction, measured in SUN
	CallTokenValue   int64  `json:"call_token_value"` // Amount of TRC10 token transferred with this transaction
	TokenId          int64  `json:"token_id"`         // TRC10 token id
	Visible          bool   `json:"visible"`          // Whether the address is in base58check format
}

type ReturnResponseCode int32

const (
	Return_SUCCESS                         ReturnResponseCode = 0
	Return_SIGERROR                        ReturnResponseCode = 1 // error in signature
	Return_CONTRACT_VALIDATE_ERROR         ReturnResponseCode = 2
	Return_CONTRACT_EXE_ERROR              ReturnResponseCode = 3
	Return_BANDWITH_ERROR                  ReturnResponseCode = 4
	Return_DUP_TRANSACTION_ERROR           ReturnResponseCode = 5
	Return_TAPOS_ERROR                     ReturnResponseCode = 6
	Return_TOO_BIG_TRANSACTION_ERROR       ReturnResponseCode = 7
	Return_TRANSACTION_EXPIRATION_ERROR    ReturnResponseCode = 8
	Return_SERVER_BUSY                     ReturnResponseCode = 9
	Return_NO_CONNECTION                   ReturnResponseCode = 10
	Return_NOT_ENOUGH_EFFECTIVE_CONNECTION ReturnResponseCode = 11
	Return_OTHER_ERROR                     ReturnResponseCode = 20
)

// Enum value maps for ReturnResponseCode.
var (
	ReturnResponseCode_name = map[int32]string{
		0:  "SUCCESS",
		1:  "SIGERROR",
		2:  "CONTRACT_VALIDATE_ERROR",
		3:  "CONTRACT_EXE_ERROR",
		4:  "BANDWITH_ERROR",
		5:  "DUP_TRANSACTION_ERROR",
		6:  "TAPOS_ERROR",
		7:  "TOO_BIG_TRANSACTION_ERROR",
		8:  "TRANSACTION_EXPIRATION_ERROR",
		9:  "SERVER_BUSY",
		10: "NO_CONNECTION",
		11: "NOT_ENOUGH_EFFECTIVE_CONNECTION",
		20: "OTHER_ERROR",
	}
	ReturnResponseCode_value = map[string]int32{
		"SUCCESS":                         0,
		"SIGERROR":                        1,
		"CONTRACT_VALIDATE_ERROR":         2,
		"CONTRACT_EXE_ERROR":              3,
		"BANDWITH_ERROR":                  4,
		"DUP_TRANSACTION_ERROR":           5,
		"TAPOS_ERROR":                     6,
		"TOO_BIG_TRANSACTION_ERROR":       7,
		"TRANSACTION_EXPIRATION_ERROR":    8,
		"SERVER_BUSY":                     9,
		"NO_CONNECTION":                   10,
		"NOT_ENOUGH_EFFECTIVE_CONNECTION": 11,
		"OTHER_ERROR":                     20,
	}
)

type TriggerConstantContractResult struct {
	Result bool `json:"result"`
}

type ConstantRet struct{}

type TriggerConstantTransaction struct {
	Ret        []ConstantRet  `json:"ret"`
	Visible    bool           `json:"visible"`
	TxID       string         `json:"txID"`
	RawData    TriggerRawData `json:"raw_data"`
	RawDataHex string         `json:"raw_data_hex"`
}

type TriggerConstantContractResponse struct {
	Result         TriggerConstantContractResult `json:"result"`          // Run result, for detailed parameter definition, refer to EstimateEnergy
	EnergyUsed     int64                         `json:"energy_used"`     // Estimated energy consumption, including the basic energy consumption and penalty energy consumption
	EnergyPenalty  int64                         `json:"energy_penalty"`  // The penalty energy consumption
	ConstantResult []string                      `json:"constant_result"` // []	Result list
	Transaction    TriggerConstantTransaction    `json:"transaction"`     // Transaction information, refer to GetTransactionByID
}

func (tc *TronJsonClient) TriggerConstantContract(tcRequest *TriggerConstantContractRequest) (*TriggerConstantContractResponse, error) {
	triggerContractEndpoint := "/wallet/triggerconstantcontract"
	contractResponse := TriggerConstantContractResponse{}

	err := tc.post(tc.baseURL+triggerContractEndpoint, tcRequest, &contractResponse)
	if err != nil {
		return nil, fmt.Errorf("trigger constant contract request (%s) failed: %w", tc.baseURL+triggerContractEndpoint, err)
	}

	return &contractResponse, nil
}

type BroadcastResponse struct {
	Result  bool   `json:"result"`
	Code    string `json:"code"`
	TxID    string `json:"txid"`
	Message string `json:"message"`
}

func (tc *TronJsonClient) BroadcastTransaction(reqBody *Transaction) (*BroadcastResponse, error) {
	response := BroadcastResponse{}
	broadcastEndpoint := "/wallet/broadcasttransaction"

	err := tc.post(tc.baseURL+broadcastEndpoint, reqBody, &response)

	if err != nil {
		return nil, fmt.Errorf("broadcast transaction request (%s) failed: %w", tc.baseURL+broadcastEndpoint, err)
	}

	return &response, nil
}

type EnergyEstimateRequest struct {
	OwnerAddress     string `json:"owner_address"`     // Owner address that triggers the contract. If visible=true, use base58check format, otherwise use hex format
	ContractAddress  string `json:"contract_address"`  // Smart contract address. If visible=true, use base58check format, otherwise use hex format
	FunctionSelector string `json:"function_selector"` // Function call, must not be left blank
	Parameter        string `json:"parameter"`
	Data             string `json:"data"`             // The bytecode of the contract or the data for interacting with smart contracts, including the contract function and parameters
	CallValue        int64  `json:"call_value"`       // Amount of TRX transferred with this transaction, measured in SUN
	CallTokenValue   int64  `json:"call_token_value"` // Amount of TRC10 token transferred with this transaction
	TokenId          int64  `json:"token_id"`         // TRC10 token id
	Visible          bool   `json:"visible"`          // Whether the address is in base58check format
}

type ReturnEnergyEstimate struct {
	Result  bool               `json:"result"`  // Is the estimate successful
	Code    ReturnResponseCode `json:"code"`    // (enum)   response code, an enum type
	Message string             `json:"message"` // Result message
}

type EnergyEstimateResult struct {
	Result         ReturnEnergyEstimate `json:"result"`          // Run result
	EnergyRequired int64                `json:"energy_required"` // Estimated energy to run the contract
}

func (tc *TronJsonClient) EstimateEnergy(reqBody *EnergyEstimateRequest) (*EnergyEstimateResult, error) {
	response := EnergyEstimateResult{}
	energyEstimateEndpoint := "/wallet/estimateenergy"

	err := tc.post(tc.baseURL+energyEstimateEndpoint, reqBody, &response)

	if err != nil {
		return nil, fmt.Errorf("energy estimation request (%s) failed: %w", tc.baseURL+energyEstimateEndpoint, err)
	}

	return &response, nil
}
