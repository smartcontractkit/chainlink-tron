package monitor

import (
	"testing"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBalanceMonitorUpdateProm(t *testing.T) {
	b := &balanceMonitor{
		chainID: "testChainID",
	}

	testAddr, err := tronaddress.Base58ToAddress("TJRabPrwbZy45sbavfcjinPJC18kjpRTv8")
	require.NoError(t, err, "Failed to create test address")

	// Test cases
	testCases := []struct {
		name     string
		sun      int64
		expected float64
	}{
		{"Zero balance", 0, 0},
		{"1 TRX", 1_000_000, 1},
		{"1.5 TRX", 1_500_000, 1.5},
		{"Large balance", 1_000_000_000_000, 1_000_000},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			promTronBalance.Reset()
			b.updateProm(testAddr, tc.sun)

			// Check if the metric was updated correctly
			expected := tc.expected
			actual := testutil.ToFloat64(promTronBalance.WithLabelValues(testAddr.String(), b.chainID, "tron", "TRX"))

			assert.Equal(t, expected, actual, "Unexpected metric value")
		})
	}
}
