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
	client  *http.Client
}

func NewTronJsonClient(baseURL string) *TronJsonClient {
	return &TronJsonClient{
		baseURL: baseURL,
		client:  &http.Client{},
	}
}

func (tc *TronJsonClient) request(method string, endpoint string, reqBody interface{}, responseBody interface{}) (int, []byte, error) {
	var req *http.Request

	if reqBody != nil {
		var jsonbytes []byte
		var err error
		jsonbytes, err = json.Marshal(reqBody)
		if err != nil {
			return 0, []byte{}, NewMarshalError(err)
		}

		req, err = http.NewRequest(method, endpoint, bytes.NewBuffer(jsonbytes))
		if err != nil {
			return 0, []byte{}, NewRequestCreationError(endpoint, err)
		}
	} else {
		var err error
		req, err = http.NewRequest(method, endpoint, nil)
		if err != nil {
			return 0, []byte{}, NewRequestCreationError(endpoint, err)
		}
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

func (tc *TronJsonClient) get(endpoint string, responseBody interface{}) (int, []byte, error) {
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
