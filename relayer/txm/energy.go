package txm

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

const DEFAULT_ENERGY_UNIT_PRICE int64 = 420

func parseLatestEnergyPrice(energyPricesStr string) (int64, error) {
	energyPricesList := strings.Split(energyPricesStr, ",")
	if len(energyPricesList) == 0 {
		return DEFAULT_ENERGY_UNIT_PRICE, errors.New("empty energy prices")
	}

	lastPriceParts := strings.Split(energyPricesList[len(energyPricesList)-1], ":")
	if len(lastPriceParts) != 2 {
		return DEFAULT_ENERGY_UNIT_PRICE, fmt.Errorf("invalid format for energy price component: expected 'timestamp:price', got %q", lastPriceParts)
	}

	energyUnitPrice, err := strconv.ParseInt(lastPriceParts[1], 10, 64)
	if err != nil {
		return DEFAULT_ENERGY_UNIT_PRICE, fmt.Errorf("failed to parse energy unit price: %w", err)
	}

	return energyUnitPrice, nil
}

func calculatePaddedFeeLimit(feeLimit int64, bumpTimes uint64) int64 {
	return int64(float64(feeLimit) * math.Pow(1.5, float64(bumpTimes)))
}
