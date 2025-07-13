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

func TestGetAccount(t *testing.T) {
	httpClient := &http.Client{}
	getAccountResponse := `{"address": "TVSTZkvVosqh4YHLwHmmNuqeyn967aE2iv","balance": 2059504131,"create_time": 1740420096000,"net_window_size": 28800000,"net_window_optimized": true,"account_resource": {"energy_window_size": 28800000,"energy_window_optimized": true},"owner_permission": {"permission_name": "owner","threshold": 1,"keys": [{"address": "TVSTZkvVosqh4YHLwHmmNuqeyn967aE2iv","weight": 1}]},"active_permission": [{"type": "Active","id": 2,"permission_name": "active","threshold": 1,"operations": "7fff1fc0033ec30f000000000000000000000000000000000000000000000000","keys": [{"address": "TVSTZkvVosqh4YHLwHmmNuqeyn967aE2iv","weight": 1}]}],"frozenV2": [{},{"type": "ENERGY"},{"type": "TRON_POWER"}],"assetV2": [{"key": "1005065","value": 8888880000}],"free_asset_net_usageV2": [{"key": "1005065","value": 0}],"asset_optimized": true}`
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, getAccountResponse)
	}))
	defer testServer.Close()

	ctx := context.Background()
	soliditynodeClient := NewClient(testServer.URL, httpClient)
	addr, err := address.StringToAddress("TVSTZkvVosqh4YHLwHmmNuqeyn967aE2iv")
	assert.NoError(t, err)
	res, err := soliditynodeClient.GetAccount(ctx, addr)
	assert.NoError(t, err)
	assert.NotNil(t, res)
	assert.Equal(t, "TVSTZkvVosqh4YHLwHmmNuqeyn967aE2iv", res.Address)
	assert.Equal(t, int64(2059504131), res.Balance)
}
