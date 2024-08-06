package monitor

import (
	"context"
	"time"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/utils"
)

// Config defines the monitor configuration.
type Config interface {
	BalancePollPeriod() time.Duration
}

// Keystore provides the addresses to be monitored.
type Keystore interface {
	Accounts(ctx context.Context) ([]string, error)
}

type BalanceClient interface {
	Balance(addr tronaddress.Address) (int64, error)
}

// NewBalanceMonitor returns a balance monitoring services.Service which reports the TRX balance of all ks keys to prometheus.
func NewBalanceMonitor(chainID string, cfg Config, lggr logger.Logger, ks Keystore, newReader func() (BalanceClient, error)) services.Service {
	return newBalanceMonitor(chainID, cfg, lggr, ks, newReader)
}

func newBalanceMonitor(chainID string, cfg Config, lggr logger.Logger, ks Keystore, newReader func() (BalanceClient, error)) *balanceMonitor {
	b := balanceMonitor{
		chainID:   chainID,
		cfg:       cfg,
		lggr:      logger.Named(lggr, "BalanceMonitor"),
		ks:        ks,
		newReader: newReader,
		stop:      make(chan struct{}),
		done:      make(chan struct{}),
	}
	b.updateFn = b.updateProm
	return &b
}

type balanceMonitor struct {
	services.StateMachine
	chainID   string
	cfg       Config
	lggr      logger.Logger
	ks        Keystore
	newReader func() (BalanceClient, error)
	updateFn  func(acc tronaddress.Address, sun int64) // overridable for testing

	reader BalanceClient

	stop services.StopChan
	done chan struct{}
}

func (b *balanceMonitor) Name() string {
	return b.lggr.Name()
}

func (b *balanceMonitor) Start(context.Context) error {
	return b.StartOnce("TronBalanceMonitor", func() error {
		go b.monitor()
		return nil
	})
}

func (b *balanceMonitor) Close() error {
	return b.StopOnce("TronBalanceMonitor", func() error {
		close(b.stop)
		<-b.done
		return nil
	})
}

func (b *balanceMonitor) HealthReport() map[string]error {
	return map[string]error{b.Name(): b.Healthy()}
}

func (b *balanceMonitor) monitor() {
	defer close(b.done)
	ctx, cancel := b.stop.NewCtx()
	defer cancel()

	tick := time.After(utils.WithJitter(b.cfg.BalancePollPeriod()))
	for {
		select {
		case <-b.stop:
			return
		case <-tick:
			b.updateBalances(ctx)
			tick = time.After(utils.WithJitter(b.cfg.BalancePollPeriod()))
		}
	}
}

func (b *balanceMonitor) getReader() (BalanceClient, error) {
	if b.reader == nil {
		var err error
		b.reader, err = b.newReader()
		if err != nil {
			return nil, err
		}
	}
	return b.reader, nil
}

func (b *balanceMonitor) updateBalances(ctx context.Context) {
	keys, err := b.ks.Accounts(ctx)
	if err != nil {
		b.lggr.Errorw("Failed to get keys", "err", err)
		return
	}
	if len(keys) == 0 {
		return
	}
	reader, err := b.getReader()
	if err != nil {
		b.lggr.Errorw("Failed to get client", "err", err)
		return
	}
	var gotSomeBals bool
	for _, k := range keys {
		// Check for shutdown signal, since Balance blocks and may be slow.
		select {
		case <-b.stop:
			return
		default:
		}
		addr, err := tronaddress.Base58ToAddress(k)
		if err != nil {
			b.lggr.Errorw("Failed to parse address", "account", k, "err", err)
			continue
		}
		sun, err := reader.Balance(addr)
		if err != nil {
			b.lggr.Errorw("Failed to get balance", "account", k, "err", err)
			continue
		}
		gotSomeBals = true
		b.updateFn(addr, sun)
	}
	if !gotSomeBals {
		// Try a new client next time.
		b.reader = nil
	}
}
