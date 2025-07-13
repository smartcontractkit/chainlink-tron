package fullnode

import "context"

type EnergyPrices struct {
	Prices string `json:"prices"` // All historical energy unit price information. Each unit price change is separated by a comma. Before the colon is the millisecond timestamp, and after the colon is the energy unit price in sun.
}

func (tc *Client) GetEnergyPrices(ctx context.Context) (*EnergyPrices, error) {
	energyPrices := EnergyPrices{}
	err := tc.Get(ctx, "/getenergyprices", &energyPrices)
	if err != nil {
		return nil, err
	}
	return &energyPrices, nil
}
