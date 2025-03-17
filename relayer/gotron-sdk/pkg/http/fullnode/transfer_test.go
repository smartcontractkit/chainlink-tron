package fullnode

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
)

var createTransactionResponse = `{
  "visible": true,
  "txID": "7182680f7aee0892aea0b87c783e32f8f43d50a5ffbfc014aa468892ffccb205",
  "raw_data": {
    "contract": [
      {
        "parameter": {
          "value": {
            "amount": 1000,
            "owner_address": "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
            "to_address": "TPswDDCAWhJAZGdHPidFg5nEf8TkNToDX1"
          },
          "type_url": "type.googleapis.com/protocol.TransferContract"
        },
        "type": "TransferContract"
      }
    ],
    "ref_block_bytes": "a2bc",
    "ref_block_hash": "46cc5c608ac2ad52",
    "expiration": 1742180955000,
    "timestamp": 1742180897646
  },
  "raw_data_hex": "0a02a2bc220846cc5c608ac2ad5240f8e6ce90da325a66080112620a2d747970652e676f6f676c65617069732e636f6d2f70726f746f636f6c2e5472616e73666572436f6e747261637412310a1541fd49eda0f23ff7ec1d03b52c3a45991c24cd440e12154198927ffb9f554dc4a453c64b2e553a02d6df514b18e80770eea6cb90da32"
}`

func TestTransfer(t *testing.T) {
	httpClient := &http.Client{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, createTransactionResponse)
	}))
	defer testServer.Close()

	fullnodeClient := NewClient(testServer.URL, httpClient)
	from, err := address.StringToAddress("TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g")
	assert.NoError(t, err)
	to, err := address.StringToAddress("TPswDDCAWhJAZGdHPidFg5nEf8TkNToDX1")
	assert.NoError(t, err)
	amount := int64(1000)

	res, err := fullnodeClient.Transfer(from, to, amount)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, 1, len(res.RawData.Contract))
	assert.Equal(t, "TransferContract", res.RawData.Contract[0].Type)
	assert.Equal(t, int64(1000), res.RawData.Contract[0].Parameter.Value.Amount)
}
