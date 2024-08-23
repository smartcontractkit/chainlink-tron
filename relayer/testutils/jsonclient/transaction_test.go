package jsonclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTransactionInfoByIDRequest(t *testing.T) {
	jsonresponse := `{
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
	code := http.StatusOK
	jsonclient := NewTronJsonClient("baseurl", NewMockJsonClient(code, jsonresponse, nil))

	a := assert.New(t)
	r := require.New(t)

	txInfoById, err := jsonclient.GetTransactionInfoById("7c2d4206c03a883dd9066d620335dc1be272a8dc733cfa3f6d10308faa37facc")
	r.Nil(err, "get transaction info by id failed:", err)

	a.Equal("7c2d4206c03a883dd9066d620335dc1be272a8dc733cfa3f6d10308faa37facc", txInfoById.ID)
	a.Equal(int64(1100000), txInfoById.Fee)
	a.Equal(int64(32880248), txInfoById.BlockNumber)
	a.Equal(int64(1681368027000), txInfoById.BlockTimeStamp)
	a.Equal("", txInfoById.ContractResult[0])
	a.Equal(int64(100000), txInfoById.Receipt.NetFee)

}
