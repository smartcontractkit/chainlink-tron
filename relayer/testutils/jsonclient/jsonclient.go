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

	"github.com/smartcontractkit/chainlink-common/pkg/loop"
)

const (
	errFailedToReadResponseBody  = "failed to read response body: %v"
	errFailedToUnmarshalResponse = "failed to unmarshal response: %v"
	errFailedToDecodeResponse    = "failed to decode response into %s struct: %v"
)

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

type BroadcastResponse struct {
	Result  bool   `json:"result"`
	Code    string `json:"code"`
	TxID    string `json:"txid"`
	Message string `json:"message"`
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

func (tc *TronJsonClient) request(method string, endpoint string, reqBody interface{}, responseBody interface{}) (int, []byte, error) {
	jsonbytes, err := json.Marshal(reqBody)
	if err != nil {
		return 0, []byte{}, NewMarshalError(err)
	}

	req, err := http.NewRequest(method, endpoint, bytes.NewBuffer(jsonbytes))
	if err != nil {
		return 0, []byte{}, NewRequestCreationError(endpoint, err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return 0, []byte{}, NewRequestError(endpoint, err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, []byte{}, NewResponseBodyError(endpoint, err)
	}

	if resp.StatusCode != http.StatusOK {
		return resp.StatusCode, body, NewStatusCodeError(endpoint, body, resp.StatusCode)
	}

	if err = json.Unmarshal(body, responseBody); err != nil {
		return resp.StatusCode, body, NewUnmarshalError(err)
	}

	return resp.StatusCode, body, nil

}

func (tc *TronJsonClient) post(endpoint string, reqBody, responseBody interface{}) (int, []byte, error) {
	return tc.request("POST", endpoint, reqBody, responseBody)
}

func (tc *TronJsonClient) BroadcastTransaction(reqBody *Transaction) (*BroadcastResponse, error) {
	response := BroadcastResponse{}
	broadcastEndpoint := "/wallet/broadcasttransaction"

	// response body bytes and http status ignored for now
	_, _, err := tc.post(tc.baseURL+broadcastEndpoint, reqBody, &response)

	if err != nil {
		return nil, fmt.Errorf("broadcast transaction failed: %v", err)
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
