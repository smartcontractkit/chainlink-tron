package soliditynode

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var blockResponse = `{
  "blockID": "000000000325a7105234af0154beb7fcb0363b809cb469fe7e0e0fd571bbd054",
  "block_header": {
    "raw_data": {
      "number": 52799248,
      "txTrieRoot": "0000000000000000000000000000000000000000000000000000000000000000",
      "witness_address": "4116c7786091fdd1bde708bc7dee96cd7371543ce4",
      "parentHash": "000000000325a70fc9a4058603644e3bfba280c418981dcf0a27d13f15ff3c66",
      "version": 31,
      "timestamp": 1742184195000
    },
    "witness_signature": "2cf3060d4904e3a61439ef44c90b283b78ac94512504749dd45b96e20d64da1f7f3ca5e57616d5c6b97363b508909ac2eecf5ea437e5effe1343ddd63d7c720500"
  }
}`

func TestGetNowBlock(t *testing.T) {
	httpClient := &http.Client{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, blockResponse)
	}))
	defer testServer.Close()

	soliditynodeClient := NewClient(testServer.URL, httpClient)
	res, err := soliditynodeClient.GetNowBlock()
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, int64(52799248), res.BlockHeader.RawData.Number)
}

func TestGetBlockByNum(t *testing.T) {
	httpClient := &http.Client{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, blockResponse)
	}))
	defer testServer.Close()

	soliditynodeClient := NewClient(testServer.URL, httpClient)
	res, err := soliditynodeClient.GetBlockByNum(52799248)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, int64(52799248), res.BlockHeader.RawData.Number)
}
