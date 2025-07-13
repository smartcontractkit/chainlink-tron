package soliditynode

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var getTxInfoResp = `{
  "id": "7c2d4206c03a883dd9066d620335dc1be272a8dc733cfa3f6d10308faa37facc",
  "fee": 1100000,
  "blockNumber": 32880248,
  "blockTimeStamp": 1681368027000,
  "contractResult": [
    ""
  ],
  "receipt": {
    "net_fee": 100000
  }
}`

func TestGetTransactionInfoById(t *testing.T) {
	httpClient := &http.Client{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, getTxInfoResp)
	}))
	defer testServer.Close()

	ctx := context.Background()
	soliditynodeClient := NewClient(testServer.URL, httpClient)
	res, err := soliditynodeClient.GetTransactionInfoById(ctx, "abcde")
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, int64(32880248), res.BlockNumber)
	assert.Equal(t, int64(100000), res.Receipt.NetFee)
}

var getNonExistentTxInfoResp = `{}`

func TestGetTransactionInfoById_NonExistent(t *testing.T) {
	httpClient := &http.Client{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, getNonExistentTxInfoResp)
	}))
	defer testServer.Close()

	ctx := context.Background()
	soliditynodeClient := NewClient(testServer.URL, httpClient)
	_, err := soliditynodeClient.GetTransactionInfoById(ctx, "abcde")
	assert.Error(t, err)
	assert.ErrorContains(t, err, "transaction not found")
}
