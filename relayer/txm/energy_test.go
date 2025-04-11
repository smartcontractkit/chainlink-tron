package txm_test

import (
	"fmt"
	"testing"

	"github.com/smartcontractkit/chainlink-tron/relayer/txm"
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
		expectedPrice   int32
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
			expectedPrice:   txm.DEFAULT_ENERGY_UNIT_PRICE,
			expectedErrMsg:  "invalid format for energy price component: expected 'timestamp:price', got [\"\"]",
		},
		{
			name:            "Invalid last price component",
			energyPricesStr: "0:100,invalid",
			expectedPrice:   txm.DEFAULT_ENERGY_UNIT_PRICE,
			expectedErrMsg:  "invalid format for energy price component: expected 'timestamp:price', got [\"invalid\"]",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			price, err := txm.ParseLatestEnergyPrice(tc.energyPricesStr)

			assert.Equal(t, tc.expectedPrice, price)
			if tc.expectedErrMsg != "" {
				assert.EqualError(t, err, tc.expectedErrMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCalculatePaddedFeeLimit(t *testing.T) {
	var DEFAULT_ENERGY_MULTIPLIER = 1.5

	tests := []struct {
		feeLimit int32
		attempt  uint32
		expected int32
	}{
		{feeLimit: 1000, attempt: 0, expected: 1500},
		{feeLimit: 1000, attempt: 1, expected: 2250},
		{feeLimit: 1000, attempt: 2, expected: 3375},
		{feeLimit: 1000, attempt: 3, expected: 5062},
	}

	for _, tt := range tests {
		t.Run("FeeLimit: "+fmt.Sprint(tt.feeLimit)+", Attempt: "+fmt.Sprint(tt.attempt), func(t *testing.T) {
			result := txm.CalculatePaddedFeeLimit(tt.feeLimit, tt.attempt, DEFAULT_ENERGY_MULTIPLIER)
			if result != tt.expected {
				t.Errorf("calculatePaddedFeeLimit(%d, %d) = %d, want %d", tt.feeLimit, tt.attempt, result, tt.expected)
			}
		})
	}
}
