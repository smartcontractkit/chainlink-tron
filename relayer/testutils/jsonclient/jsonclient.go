package jsonclient

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mitchellh/mapstructure"

	"github.com/smartcontractkit/chainlink-common/pkg/loop"
)

const (
	deployEndpoint      = "/wallet/deploycontract"
	broadcastEndpoint   = "/wallet/broadcasttransaction"
	getContractEndpoint = "/wallet/getcontract"
)

const (
	errFailedToReadResponseBody  = "failed to read response body: %v"
	errFailedToUnmarshalResponse = "failed to unmarshal response: %v"
	errFailedToDecodeResponse    = "failed to decode response into %s struct: %v"
)

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

type Transaction struct {
	Visible bool   `json:"visible" mapstructure:"visible"`
	TxID    string `json:"txID" mapstructure:"txID"`
	RawData struct {
		Contract      []map[string]interface{} `json:"contract,omitempty" mapstructure:"contract"`
		RefBlockBytes string                   `json:"ref_block_bytes,omitempty" mapstructure:"ref_block_bytes"`
		RefBlockHash  string                   `json:"ref_block_hash,omitempty" mapstructure:"ref_block_hash"`
		Expiration    int64                    `json:"expiration,omitempty" mapstructure:"expiration"`
		FeeLimit      int64                    `json:"fee_limit,omitempty" mapstructure:"fee_limit"`
		Timestamp     int64                    `json:"timestamp,omitempty" mapstructure:"timestamp"`
	} `json:"raw_data" mapstructure:"raw_data"`
	RawDataHex string   `json:"raw_data_hex" mapstructure:"raw_data_hex"`
	Signature  []string `json:"signature" mapstructure:"signature"`
}

type GetContractRequest struct {
	Value   string `json:"value"`
	Visible bool   `json:"visible"`
}

type GetContractResponse struct {
	ContractAddress string `json:"contract_address"`
}

type BroadcastResponse struct {
	Result  bool   `mapstructure:"result"`
	Code    string `mapstructure:"code"`
	TxID    string `mapstructure:"txid"`
	Message string `mapstructure:"message"`
}

type TronJsonClient struct {
	baseURL string
	client  *http.Client
}

func NewTronJsonClient(baseURL string) *TronJsonClient {
	return &TronJsonClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (tc *TronJsonClient) DeployContract(request *DeployContractRequest) (*Transaction, error) {
	payloadBytes, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", tc.baseURL+deployEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(errFailedToReadResponseBody, err)
	}

	var responseMap map[string]interface{}
	if err = json.Unmarshal(body, &responseMap); err != nil {
		return nil, fmt.Errorf(errFailedToUnmarshalResponse, err)
	}

	if errorMsg, ok := responseMap["Error"]; ok {
		return nil, fmt.Errorf("deploy contract failed: %v", errorMsg)
	}

	var transaction Transaction
	if err = mapstructure.Decode(responseMap, &transaction); err != nil {
		return nil, fmt.Errorf(errFailedToDecodeResponse, "Transaction", err)
	}

	return &transaction, nil
}

func (tc *TronJsonClient) BroadcastTransaction(transaction *Transaction) (*BroadcastResponse, error) {
	payloadBytes, err := json.Marshal(transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal payload: %v", err)
	}

	req, err := http.NewRequest("POST", tc.baseURL+broadcastEndpoint, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf(errFailedToReadResponseBody, err)
	}

	var responseMap map[string]interface{}
	if err = json.Unmarshal(body, &responseMap); err != nil {
		return nil, fmt.Errorf(errFailedToUnmarshalResponse, err)
	}

	if errorMsg, ok := responseMap["Error"]; ok {
		return nil, fmt.Errorf("broadcast transaction failed: %v", errorMsg)
	}

	var response BroadcastResponse
	if err = mapstructure.Decode(responseMap, &response); err != nil {
		return nil, fmt.Errorf(errFailedToDecodeResponse, "BroadcastResponse", err)
	}

	if !response.Result {
		return nil, fmt.Errorf("broadcasting failed. Code: %s, Message: %s", response.Code, response.Message)
	}

	return &response, nil
}

func (t *Transaction) SignWithKey(privateKey *ecdsa.PrivateKey) error {
	txIdBytes, err := hex.DecodeString(t.TxID)
	if err != nil {
		return fmt.Errorf("failed to decode raw_data_hex: %v", err)
	}

	signature, err := crypto.Sign(txIdBytes, privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	signatureHex := hex.EncodeToString(signature)

	t.Signature = []string{signatureHex}

	return nil
}

func (t *Transaction) Sign(fromAddress string, keystore loop.Keystore) error {
	txIdBytes, err := hex.DecodeString(t.TxID)
	if err != nil {
		return fmt.Errorf("failed to decode raw_data_hex: %v", err)
	}

	signature, err := keystore.Sign(context.Background(), fromAddress, txIdBytes)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	signatureHex := hex.EncodeToString(signature)

	t.Signature = []string{signatureHex}

	return nil
}

func (tc *TronJsonClient) GetContract(address string) (bool, error) {
	reqBody := GetContractRequest{
		Value:   address,
		Visible: true,
	}
	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("POST", tc.baseURL+getContractEndpoint, bytes.NewBuffer(reqBytes))
	if err != nil {
		return false, fmt.Errorf("failed to create request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return false, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to get contract info, status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return false, fmt.Errorf(errFailedToReadResponseBody, err)
	}

	var contractInfo GetContractResponse
	if err = json.Unmarshal(body, &contractInfo); err != nil {
		return false, fmt.Errorf(errFailedToUnmarshalResponse, err)
	}

	return contractInfo.ContractAddress != "", nil
}
