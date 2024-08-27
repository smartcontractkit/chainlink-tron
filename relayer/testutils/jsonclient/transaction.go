package jsonclient

import "fmt"

type Transaction_ResultContractResult int32

const (
	Transaction_Result_DEFAULT              Transaction_ResultContractResult = 0
	Transaction_Result_SUCCESS              Transaction_ResultContractResult = 1
	Transaction_Result_REVERT               Transaction_ResultContractResult = 2
	Transaction_Result_BAD_JUMP_DESTINATION Transaction_ResultContractResult = 3
	Transaction_Result_OUT_OF_MEMORY        Transaction_ResultContractResult = 4
	Transaction_Result_PRECOMPILED_CONTRACT Transaction_ResultContractResult = 5
	Transaction_Result_STACK_TOO_SMALL      Transaction_ResultContractResult = 6
	Transaction_Result_STACK_TOO_LARGE      Transaction_ResultContractResult = 7
	Transaction_Result_ILLEGAL_OPERATION    Transaction_ResultContractResult = 8
	Transaction_Result_STACK_OVERFLOW       Transaction_ResultContractResult = 9
	Transaction_Result_OUT_OF_ENERGY        Transaction_ResultContractResult = 10
	Transaction_Result_OUT_OF_TIME          Transaction_ResultContractResult = 11
	Transaction_Result_JVM_STACK_OVER_FLOW  Transaction_ResultContractResult = 12
	Transaction_Result_UNKNOWN              Transaction_ResultContractResult = 13
	Transaction_Result_TRANSFER_FAILED      Transaction_ResultContractResult = 14
	Transaction_Result_INVALID_CODE         Transaction_ResultContractResult = 15
)

// Enum value maps for Transaction_ResultContractResult.
var (
	Transaction_ResultContractResult_name = map[int32]string{
		0:  "DEFAULT",
		1:  "SUCCESS",
		2:  "REVERT",
		3:  "BAD_JUMP_DESTINATION",
		4:  "OUT_OF_MEMORY",
		5:  "PRECOMPILED_CONTRACT",
		6:  "STACK_TOO_SMALL",
		7:  "STACK_TOO_LARGE",
		8:  "ILLEGAL_OPERATION",
		9:  "STACK_OVERFLOW",
		10: "OUT_OF_ENERGY",
		11: "OUT_OF_TIME",
		12: "JVM_STACK_OVER_FLOW",
		13: "UNKNOWN",
		14: "TRANSFER_FAILED",
		15: "INVALID_CODE",
	}
	Transaction_ResultContractResult_value = map[string]int32{
		"DEFAULT":              0,
		"SUCCESS":              1,
		"REVERT":               2,
		"BAD_JUMP_DESTINATION": 3,
		"OUT_OF_MEMORY":        4,
		"PRECOMPILED_CONTRACT": 5,
		"STACK_TOO_SMALL":      6,
		"STACK_TOO_LARGE":      7,
		"ILLEGAL_OPERATION":    8,
		"STACK_OVERFLOW":       9,
		"OUT_OF_ENERGY":        10,
		"OUT_OF_TIME":          11,
		"JVM_STACK_OVER_FLOW":  12,
		"UNKNOWN":              13,
		"TRANSFER_FAILED":      14,
		"INVALID_CODE":         15,
	}
)

type ResourceReceipt struct {
	EnergyUsage        int64                            `json:"energy_usage,omitempty"`
	EnergyFee          int64                            `json:"energy_fee,omitempty"`
	OriginEnergyUsage  int64                            `json:"origin_energy_usage,omitempty"`
	EnergyUsageTotal   int64                            `json:"energy_usage_total,omitempty"`
	NetUsage           int64                            `json:"net_usage,omitempty"`
	NetFee             int64                            `json:"net_fee,omitempty"`
	Result             Transaction_ResultContractResult `json:"result,omitempty"`
	EnergyPenaltyTotal int64                            `json:"energy_penalty_total,omitempty"`
}

type Log struct {
	Address string   `json:"address,omitempty"`
	Topics  []string `json:"topics,omitempty"`
	Data    string   `json:"data,omitempty"`
}

type InternalTransaction_CallValueInfo struct {
	CallValue int64  `json:"callValue,omitempty"` // trx (TBD: or token) value
	TokenId   string `json:"tokenId,omitempty"`   // TBD: tokenName, trx should be empty
}

type InternalTransaction struct {
	Hash              string                               `json:"hash,omitempty"`               // internalTransaction identity, the root InternalTransaction hash should equals to root transaction id.
	CallerAddress     string                               `json:"caller_address,omitempty"`     // the one send trx (TBD: or token) via function
	TransferToAddress string                               `json:"transferTo_address,omitempty"` // the one recieve trx (TBD: or token) via function
	CallValueInfo     []*InternalTransaction_CallValueInfo `json:"callValueInfo,omitempty"`
	Note              string                               `json:"note,omitempty"`
	Rejected          bool                                 `json:"rejected,omitempty"`
	Extra             string                               `json:"extra,omitempty"`
}

type TransactionInfo struct {
	ID                     string                `json:"id"`                       // Transaction ID
	Fee                    int64                 `json:"fee"`                      // The total number of TRX burned in this transaction, including TRX burned for bandwidth/energy, memo fee, account activation fee, multi-signature fee and other fees
	BlockNumber            int64                 `json:"blockNumber"`              // The block number
	BlockTimeStamp         int64                 `json:"blockTimeStamp"`           // The block timestamp, the unit is millisecond
	ContractResult         []string              `json:"contractResult"`           // Transaction Execution Results
	ContractAddress        string                `json:"contract_address"`         // Contract address
	Receipt                ResourceReceipt       `json:"receipt"`                  // Transaction receipt, including transaction execution result and transaction fee details, which contains the following fields:
	Log                    []Log                 `json:"log"`                      // The log of events triggered during the smart contract call, each log includes the following information:
	Result                 int                   `json:"result"`                   // Execution results. If the execution is successful, the field will not be displayed in the returned value, if the execution fails, the field will be "FAILED"
	ResMessage             string                `json:"resMessage"`               // When the transaction execution fails, the details of the failure will be returned through this field. Hex format, you can convert it to a string to get plaintext information.
	WithdrawAmount         int64                 `json:"withdraw_amount"`          // For the withdrawal reward transaction、unfreeze transaction, they will withdraw the vote reward to account. The number of rewards withdrawn to the account is returned through this field, and the unit is sun
	UnfreezeAmount         int64                 `json:"unfreeze_amount"`          // In the Stake1.0 stage, for unstaking transactions, this field returns the amount of unstaked TRX, the unit is sun
	InternalTransactions   []InternalTransaction `json:"internal_transactions"`    // []	Internal transaction
	WithdrawExpireAmount   int64                 `json:"withdraw_expire_amount"`   // In the Stake2.0 stage, for unstaking transaction and withdrawing unfrozen balance transaction, and cancelling all unstakes transaction, this field returns the amount of unfrozen TRX withdrawn to the account in this transaction, the unit is sun
	CancelUnfreezev2Amount map[string]int64      `json:"cancel_unfreezeV2_amount"` // 	The amount of TRX re-staked to obtain various types of resources, in sun, that is, the amount of unstaked principal that has been canceled, the key is: "BANDWIDTH" or "ENERGY" or "TRON_POWER"
}

type GetTransactionInfoByIDRequest struct {
	Value string `json:"value"` // Transaction hash, i.e. transaction id
}

func (tc *TronJsonClient) GetTransactionInfoById(txhash string) (*TransactionInfo, error) {
	transactionInfo := TransactionInfo{}
	getTransactionInfoByIdEndpoint := "/wallet/gettransactioninfobyid"

	err := tc.post(tc.baseURL+getTransactionInfoByIdEndpoint,
		&GetTransactionInfoByIDRequest{
			Value: txhash,
		}, &transactionInfo)

	if err != nil {
		return nil, fmt.Errorf("get transaction info by id request (%s) failed: %w", tc.baseURL+getTransactionInfoByIdEndpoint, err)
	}

	return &transactionInfo, nil
}
