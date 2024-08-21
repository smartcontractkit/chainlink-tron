package jsonclient

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEstimateEnergy(t *testing.T) {
	response := `{
  "result": {
    "result": true
  },
  "energy_required": 1082
}`
	code := http.StatusOK
	jsonclient := NewTronJsonClient("someurl", NewMockJsonClient(code, response, nil))

	a := assert.New(t)
	r := require.New(t)

	estimation, err := jsonclient.EstimateEnergy(&EnergyEstimateRequest{})
	r.Nil(err, "EstimateEnergy failed", "error:", err)
	a.Equal(true, estimation.Result.Result)
	a.Equal(int64(1082), estimation.EnergyRequired)
}

func TestEstimateEnergyFail(t *testing.T) {
	reqerr := &url.Error{
		Op:  "Post",
		URL: "http://endpoint/",
		Err: fmt.Errorf("request failed"),
	}

	jsonclient := NewTronJsonClient("http://endpoint", NewMockJsonClient(500, "", reqerr))
	r := require.New(t)
	est, err := jsonclient.EstimateEnergy(&EnergyEstimateRequest{})
	r.Nil(est)
	r.NotNil(err)
}
