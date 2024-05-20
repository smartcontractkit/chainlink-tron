package txm

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseLatestEnergyPrice(t *testing.T) {
	// example json string
	// {
	// "prices": "0:100,1575871200000:10,1606537680000:40,1614238080000:140,1635739080000:280,1681895880000:420"
	// }

	testCases := []struct {
		name            string
		energyPricesStr string
		expectedPrice   int64
		expectedErrMsg  string
	}{
		{
			name:            "Valid energy prices",
			energyPricesStr: "0:100,1575871200000:10,1606537680000:40,1614238080000:140,1635739080000:280,1681895880000:420",
			expectedPrice:   420,
		},
		{
			name:            "Empty energy prices",
			energyPricesStr: "",
			expectedPrice:   DEFAULT_ENERGY_UNIT_PRICE,
			expectedErrMsg:  "invalid format for energy price component: expected 'timestamp:price', got [\"\"]",
		},
		{
			name:            "Invalid last price component",
			energyPricesStr: "0:100,invalid",
			expectedPrice:   DEFAULT_ENERGY_UNIT_PRICE,
			expectedErrMsg:  "invalid format for energy price component: expected 'timestamp:price', got [\"invalid\"]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			price, err := parseLatestEnergyPrice(tc.energyPricesStr)

			assert.Equal(t, tc.expectedPrice, price)
			if tc.expectedErrMsg != "" {
				assert.EqualError(t, err, tc.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
