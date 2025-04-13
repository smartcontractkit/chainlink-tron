package txm

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/timeutil"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
)

var _ EnergyEstimator = &basicEnergyEstimator{}

// EnergyEstimator is an interface for estimating the energy used by a transaction
// and calculating the fee limit for a transaction.
type EnergyEstimator interface {
	services.Service
	EstimateEnergy(tx *TronTx) (int64, error)
	GetEnergyUnitPrice() (int64, error)
	CalculateFeeLimit(tx *TronTx) (int64, error)
}

// basicEnergyEstimator is a basic implementation of an EnergyEstimator.
// It estimates the energy used by a transaction calling the full node's API `/estimateEnergy`
// However, if this fails or the node doesn't support it. it will fallback to simulating the transaction and getting the energy used from the response.
// It also caches the energy unit price and updates it every minute.
type basicEnergyEstimator struct {
	services.StateMachine
	lggr                  logger.Logger
	client                sdk.FullNodeClient
	estimateEnergyEnabled bool

	// Channels
	chGetEnergyUnitPrice chan struct{}
	chStop               services.StopChan
	done                 sync.WaitGroup

	// State
	energyUnitPrice int64
}

func NewBasicEnergyEstimator(lggr logger.Logger, client sdk.FullNodeClient, estimateEnergyEnabled bool) *basicEnergyEstimator {
	return &basicEnergyEstimator{
		StateMachine:          services.StateMachine{},
		lggr:                  lggr,
		client:                client,
		estimateEnergyEnabled: estimateEnergyEnabled,
		chGetEnergyUnitPrice:  make(chan struct{}),
		chStop:                make(services.StopChan),
		done:                  sync.WaitGroup{},
	}
}

func (e *basicEnergyEstimator) CalculateFeeLimit(tx *TronTx) (int64, error) {
	energyUsed, err := e.EstimateEnergy(tx)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate energy: %w", err)
	}

	energyUnitPrice, err := e.GetEnergyUnitPrice()
	if err != nil {
		return 0, fmt.Errorf("failed to get energy unit price: %w", err)
	}

	feeLimit := energyUnitPrice * energyUsed

	return feeLimit, nil
}

// GetEnergyUnitPrice returns the energy unit price from the cache
func (e *basicEnergyEstimator) GetEnergyUnitPrice() (int64, error) {
	e.StateMachine.RLock()
	defer e.StateMachine.RUnlock()

	if e.energyUnitPrice == 0 {
		e.lggr.Warnw("Energy unit price not set, using default value", "default", DEFAULT_ENERGY_UNIT_PRICE)
		return int64(DEFAULT_ENERGY_UNIT_PRICE), nil
	}

	return e.energyUnitPrice, nil
}

// Estimates the energy used by a transaction
func (t *basicEnergyEstimator) EstimateEnergy(tx *TronTx) (int64, error) {
	if t.estimateEnergyEnabled {
		estimateEnergyMessage, err := t.callEstimateEnergy(tx.FromAddress, tx.ContractAddress, tx.Method, tx.Params)
		if err == nil {
			return estimateEnergyMessage.EnergyRequired, nil
		}
	}

	// We'll fallback to TriggerConstantContract if EstimateEnergy is not supported
	triggerResponse, err := t.triggerConstantContract(tx)
	if err != nil {
		return 0, fmt.Errorf("failed to estimate energy using EstimateEnergy and fallback TriggerConstantContract: %w", err)
	}

	// The energy used is the sum of the energy used by the contract and the penalty for operations that require
	// significant dynamic memory allocation or handling of complex data structures.
	return triggerResponse.EnergyUsed + triggerResponse.EnergyPenalty, nil
}

func (e *basicEnergyEstimator) callEstimateEnergy(fromAddress, contractAddress address.Address, method string, params []any) (*soliditynode.EnergyEstimateResult, error) {
	estimateEnergyMessage, err := e.client.EstimateEnergy(fromAddress, contractAddress, method, params /* tAmount= */, 0)
	if err != nil {
		if strings.Contains(err.Error(), "this node does not support estimate energy") {
			e.estimateEnergyEnabled = false
			e.lggr.Infow("Node does not support EstimateEnergy", "err", err)
		} else {
			e.lggr.Errorw("Failed to call EstimateEnergy", "err", err)
		}
		return nil, err
	}

	return estimateEnergyMessage, nil
}

func (e *basicEnergyEstimator) triggerConstantContract(tx *TronTx) (*soliditynode.TriggerConstantContractResponse, error) {
	triggerResponse, err := e.client.TriggerConstantContract(tx.FromAddress, tx.ContractAddress, tx.Method, tx.Params)
	if err != nil {
		e.lggr.Errorw("Failed to trigger Constant Contract", "err", err, "tx", tx)
		return nil, err
	}

	e.lggr.Debugw("Triggered Constant Contract", "triggerResponse", triggerResponse, "tx", tx)
	return triggerResponse, nil
}

func (e *basicEnergyEstimator) Name() string {
	return e.lggr.Name()
}

func (e *basicEnergyEstimator) Start(ctx context.Context) error {
	return e.StartOnce("BasicEnergyEstimator", func() error {
		e.done.Add(1)
		go e.getEnergyUnitPriceTicker(ctx, services.NewTicker(time.Minute))
		return nil
	})
}

func (e *basicEnergyEstimator) Close() error {
	return e.StopOnce("BasicEnergyEstimator", func() error {
		close(e.chStop)
		e.done.Wait()
		return nil
	})
}

func (e *basicEnergyEstimator) HealthReport() map[string]error {
	return map[string]error{e.Name(): e.Healthy()}
}

func (e *basicEnergyEstimator) getEnergyUnitPriceTicker(ctx context.Context, ticker *timeutil.Ticker) {
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if energyPrices, err := e.client.GetEnergyPrices(); err == nil {
				if parsedPrice, err := ParseLatestEnergyPrice(energyPrices.Prices); err == nil {
					e.Lock()
					e.energyUnitPrice = int64(parsedPrice)
					e.Unlock()
				} else {
					e.lggr.Errorw("error parsing energy unit price", "error", err)
				}
			} else {
				e.lggr.Errorw("failed to get energy unit price", "error", err)
			}
		case <-ctx.Done():
			return
		}
	}
}
