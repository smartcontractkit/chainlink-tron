package jsonclient

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAccount(t *testing.T) {
	jsonresponse := `{
  "address": "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
  "balance": 5994521520,
  "create_time": 1637411046000,
  "latest_opration_time": 1715145693000,
  "latest_consume_time": 1717666029000,
  "latest_consume_free_time": 1715145693000,
  "net_window_size": 28800000,
  "net_window_optimized": true,
  "account_resource": {
    "latest_consume_time_for_energy": 1717675329000,
    "energy_window_size": 28800000,
    "energy_window_optimized": true
  },
  "owner_permission": {
    "permission_name": "owner",
    "threshold": 1,
    "keys": [
      {
        "address": "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
        "weight": 1
      }
    ]
  },
  "active_permission": [
    {
      "type": "Active",
      "id": 2,
      "permission_name": "active",
      "threshold": 1,
      "operations": "7fff1fc0033e0300000000000000000000000000000000000000000000000000",
      "keys": [
        {
          "address": "TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g",
          "weight": 1
        }
      ]
    }
  ],
  "frozenV2": [
    {},
    {
      "type": "ENERGY"
    },
    {
      "type": "TRON_POWER"
    }
  ],
  "asset_optimized": true
}`
	code := http.StatusOK
	jsonclient := NewTronJsonClient("baseurl", NewMockJsonClient(code, jsonresponse, nil))

	a := assert.New(t)
	r := require.New(t)

	acct, err := jsonclient.GetAccount("TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g")
	r.Nil(err, "get account failed:", err)

	a.Equal("TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g", acct.Address)
	a.Equal(int64(5994521520), acct.Balance)
	a.Equal(int64(1637411046000), acct.CreateTime)
	a.Equal(int64(1715145693000), acct.LatestOprationTime)
	a.Equal(int64(1717666029000), acct.LatestConsumeTime)
	a.Equal(int64(1715145693000), acct.LatestConsumeFreeTime)
	a.Equal(int64(28800000), acct.NetWindowSize)
	a.True(acct.NetWindowOptimized)
	a.Equal(int64(1717675329000), acct.AccountResource.LatestConsumeTimeForEnergy)
	a.Equal(int64(28800000), acct.AccountResource.EnergyWindowSize)
	a.True(acct.AccountResource.EnergyWindowOptimized)
	a.Equal("owner", acct.OwnerPermission.PermissionName)
	a.Equal(int64(1), acct.OwnerPermission.Threshold)
	a.Equal("TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g", acct.OwnerPermission.Keys[0].Address)
	a.Equal(int64(1), acct.OwnerPermission.Keys[0].Weight)
	a.Equal("Active", acct.ActivePermission[0].Type)
	a.Equal(int32(2), acct.ActivePermission[0].ID)
	a.Equal("active", acct.ActivePermission[0].PermissionName)
	a.Equal(int64(1), acct.ActivePermission[0].Threshold)
	a.Equal("7fff1fc0033e0300000000000000000000000000000000000000000000000000", acct.ActivePermission[0].Operations)
	a.Equal("TZ4UXDV5ZhNW7fb2AMSbgfAEZ7hWsnYS2g", acct.ActivePermission[0].Keys[0].Address)
	a.Equal(int64(1), acct.ActivePermission[0].Keys[0].Weight)
	a.Equal(3, len(acct.FrozenV2))
	a.Equal("", acct.FrozenV2[0].Type)
	a.Equal(int64(0), acct.FrozenV2[0].Amount)
	a.Equal("ENERGY", acct.FrozenV2[1].Type)
	a.Equal(int64(0), acct.FrozenV2[1].Amount)
	a.Equal("TRON_POWER", acct.FrozenV2[2].Type)
	a.Equal(int64(0), acct.FrozenV2[2].Amount)
}
