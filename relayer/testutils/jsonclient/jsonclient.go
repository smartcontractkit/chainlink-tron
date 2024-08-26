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

type TronJsonClient struct {
	baseURL string
	client  JsonHttpClient
}

func NewTronJsonClient(baseURL string, client JsonHttpClient) *TronJsonClient {
	return &TronJsonClient{
		baseURL: baseURL,
		client:  client,
	}
}

func (tc *TronJsonClient) request(method string, endpoint string, reqBody interface{}, responseBody interface{}) error {
	var req *http.Request

	if reqBody != nil {
		var jsonbytes []byte
		var err error
		jsonbytes, err = json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("marshalling request failed: %v", err)
		}

		req, err = http.NewRequest(method, endpoint, bytes.NewBuffer(jsonbytes))
		if err != nil {
			return fmt.Errorf("creating http request failed: %v", err)
		}
	} else {
		var err error
		req, err = http.NewRequest(method, endpoint, nil)
		if err != nil {
			return fmt.Errorf("creating http request failed: %v", err)
		}
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := tc.client.Do(req)
	if err != nil {
		return fmt.Errorf("http request failed: %v", err)
	}

	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response body failed: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("invalid http status: %d", resp.StatusCode)
	}

	if err = json.Unmarshal(body, responseBody); err != nil {
		return fmt.Errorf("unmarshalling response body failed: %v", err)
	}

	return nil

}

func (tc *TronJsonClient) post(endpoint string, reqBody, responseBody interface{}) error {
	return tc.request("POST", endpoint, reqBody, responseBody)
}

func (tc *TronJsonClient) get(endpoint string, responseBody interface{}) error {
	return tc.request("GET", endpoint, nil, responseBody)
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
