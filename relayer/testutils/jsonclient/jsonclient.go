package jsonclient

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/mitchellh/mapstructure"
)

const (
	deployEndpoint    = "/wallet/deploycontract"
	broadcastEndpoint = "/wallet/broadcasttransaction"
)

type DeployContractRequest struct {
	OwnerAddress               string `json:"owner_address"`
	ABI                        string `json:"abi"`
	Bytecode                   string `json:"bytecode"`
	Parameter                  string `json:"parameter"`
	Name                       string `json:"name"`
	Value                      int    `json:"value"`
	FeeLimit                   int    `json:"fee_limit"`
	ConsumeUserResourcePercent int    `json:"consume_user_resource_percent"`
	OriginEnergyLimit          int    `json:"origin_energy_limit"`
	Visible                    bool   `json:"visible"`
}

type Transaction struct {
	Visible bool   `json:"visible" mapstructure:"visible"`
	TxID    string `json:"txID" mapstructure:"txID"`
	RawData struct {
		Contract []struct {
			Parameter struct {
				Value struct {
					Data            string `json:"data" mapstructure:"data"`
					OwnerAddress    string `json:"owner_address" mapstructure:"owner_address"`
					ContractAddress string `json:"contract_address" mapstructure:"contract_address"`
				} `json:"value" mapstructure:"value"`
				TypeUrl string `json:"type_url" mapstructure:"type_url"`
			} `json:"parameter" mapstructure:"parameter"`
			Type string `json:"type" mapstructure:"type"`
		} `json:"contract" mapstructure:"contract"`
		RefBlockBytes string `json:"ref_block_bytes" mapstructure:"ref_block_bytes"`
		RefBlockHash  string `json:"ref_block_hash" mapstructure:"ref_block_hash"`
		Expiration    int64  `json:"expiration" mapstructure:"expiration"`
		FeeLimit      int64  `json:"fee_limit" mapstructure:"fee_limit"`
		Timestamp     int64  `json:"timestamp" mapstructure:"timestamp"`
	} `json:"raw_data" mapstructure:"raw_data"`
	RawDataHex string   `json:"raw_data_hex" mapstructure:"raw_data_hex"`
	Signature  []string `json:"signature" mapstructure:"signature"`
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

func (tc *TronJsonClient) CreateDeployContractTransaction(request *DeployContractRequest) (*Transaction, error) {
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
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var responseMap map[string]interface{}
	if err = json.Unmarshal(body, &responseMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if errorMsg, ok := responseMap["Error"]; ok {
		return nil, fmt.Errorf("deploy contract failed: %v", errorMsg)
	}

	var transaction Transaction
	if err = mapstructure.Decode(responseMap, &transaction); err != nil {
		return nil, fmt.Errorf("failed to decode response into Transaction struct: %v", err)
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
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	var responseMap map[string]interface{}
	if err = json.Unmarshal(body, &responseMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %v", err)
	}

	if errorMsg, ok := responseMap["Error"]; ok {
		return nil, fmt.Errorf("broadcast transaction failed: %v", errorMsg)
	}

	var response BroadcastResponse
	if err = mapstructure.Decode(responseMap, &response); err != nil {
		return nil, fmt.Errorf("failed to decode response into BroadcastResponse struct: %v", err)
	}

	if !response.Result {
		return nil, fmt.Errorf("broadcasting failed. Code: %s, Message: %s", response.Code, response.Message)
	}

	return &response, nil
}

func (t *Transaction) Sign(privateKey *ecdsa.PrivateKey) error {
	rawDataBytes, err := hex.DecodeString(t.RawDataHex)
	if err != nil {
		return fmt.Errorf("failed to decode raw_data_hex: %v", err)
	}

	hash := sha256.Sum256(rawDataBytes)

	signature, err := crypto.Sign(hash[:], privateKey)
	if err != nil {
		return fmt.Errorf("failed to sign transaction: %v", err)
	}

	signatureHex := hex.EncodeToString(signature)

	t.Signature = []string{signatureHex}

	return nil
}
