package monitor

import (
	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var promTronBalance = promauto.NewGaugeVec(
	prometheus.GaugeOpts{Name: "tron_balance", Help: "Tron account balances"},
	[]string{"account", "chainID", "chainSet", "denomination"},
)

// updateProm updates the prometheus metric
func (b *balanceMonitor) updateProm(acc tronaddress.Address, sun int64) {
	v := sunToTrx(sun)
	promTronBalance.WithLabelValues(acc.String(), b.chainID, "tron", "TRX").Set(v)
}

// sunToTrx converts SUN to TRX
func sunToTrx(sun int64) float64 {
	return float64(sun) / 1_000_000 // 1 TRX = 1,000,000 SUN
}
