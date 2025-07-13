package soliditynode

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
)

var triggerConstantContractResp = `{
  "result": {
    "result": true
  },
  "energy_used": 541,
  "constant_result": [
    "0000000000000000000000000000000000000000000000000000000198eb569b"
  ],
  "transaction": {
    "ret": [
      {}
    ],
    "visible": true,
    "txID": "b58c32274d9c54c590f4879cb941d7dc6a3f5e415bcc31c67bd2cb142a9c3269",
    "raw_data": {
      "contract": [
        {
          "parameter": {
            "value": {
              "data": "70a08231000000000000000000000000a614f803b6fd780986a42c78ec9c7f77e6ded13c",
              "owner_address": "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
              "contract_address": "TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs"
            },
            "type_url": "type.googleapis.com/protocol.TriggerSmartContract"
          },
          "type": "TriggerSmartContract"
        }
      ],
      "ref_block_bytes": "a73a",
      "ref_block_hash": "d472ab9f324cd862",
      "expiration": 1742184435000,
      "timestamp": 1742184376236
    },
    "raw_data_hex": "0a02a73a2208d472ab9f324cd86240b89aa392da325a8e01081f1289010a31747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e54726967676572536d617274436f6e747261637412540a1541fd49eda0f23ff7ec1d03b52c3a45991c24cd440e12154142a1e39aefa49290f2b3f9ed688d7cecf86cd6e0222470a08231000000000000000000000000a614f803b6fd780986a42c78ec9c7f77e6ded13c70accf9f92da32"
  }
}`

func TestTriggerConstantContract(t *testing.T) {
	httpClient := &http.Client{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, triggerConstantContractResp)
	}))
	defer testServer.Close()

	soliditynodeClient := NewClient(testServer.URL, httpClient)
	from, err := address.StringToAddress("TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g")
	assert.NoError(t, err)
	contractAddr, err := address.StringToAddress("TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs")
	assert.NoError(t, err)
	method := "test()"
	data := []any{}

	ctx := context.Background()
	res, err := soliditynodeClient.TriggerConstantContract(ctx, from, contractAddr, method, data)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 1, len(res.ConstantResult))
	assert.Equal(t, 1, len(res.Transaction.RawData.Contract))
	assert.Equal(t, "TriggerSmartContract", res.Transaction.RawData.Contract[0].Type)
}

var estimateEnergyResp = `{
  "result": {
    "result": true
  },
  "energy_required": 1082
}`

func TestEstimateEnergy(t *testing.T) {
	httpClient := &http.Client{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, estimateEnergyResp)
	}))
	defer testServer.Close()

	ctx := context.Background()
	soliditynodeClient := NewClient(testServer.URL, httpClient)
	from, err := address.StringToAddress("TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g")
	assert.NoError(t, err)
	contractAddr, err := address.StringToAddress("TG3XXyExBkPp9nzdajDZsozEu4BkaSJozs")
	assert.NoError(t, err)
	method := "test()"
	data := []any{}

	res, err := soliditynodeClient.EstimateEnergy(ctx, from, contractAddr, method, data, 0)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, true, res.Result.Result)
	assert.Equal(t, int64(1082), res.EnergyRequired)
}
