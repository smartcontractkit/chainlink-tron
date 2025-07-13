package fullnode

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

var getEnergyPricesResponse = `{
  "prices": "0:100,1575871200000:10,1606537680000:40,1614238080000:140,1635739080000:280,1681895880000:420"
}`

func TestGetEnergyPrices(t *testing.T) {
	httpClient := &http.Client{}
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, getEnergyPricesResponse)
	}))
	defer testServer.Close()

	ctx := context.Background()
	fullnodeClient := NewClient(testServer.URL, httpClient)
	res, err := fullnodeClient.GetEnergyPrices(ctx)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "0:100,1575871200000:10,1606537680000:40,1614238080000:140,1635739080000:280,1681895880000:420", res.Prices)
}
