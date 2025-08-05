package plugin

import (
	"errors"
	"time"

	"github.com/smartcontractkit/chainlink-common/pkg/config"
)

// Global tron defaults.
var defaultConfigSet = chainConfigSet{
	// poll period for balance monitoring
	BalancePollPeriod: 5 * time.Second,
	// transaction broadcast channel size
	BroadcastChanSize: 4096,
	// polling period for transaction confirmation
	ConfirmPollPeriod: 500 * time.Millisecond, // polling for tx confirmation
	// polling period for OCR2 contract cache
	OCR2CachePollPeriod: 5 * time.Second,
	// time to live for OCR2 contract cache
	OCR2CacheTTL: time.Minute,
	// retention period for transaction manager
	RetentionPeriod: 0,
	// reap interval for transaction manager
	ReapInterval: 1 * time.Minute,
}

// opt: remove
type chainConfigSet struct {
	BroadcastChanSize   uint64
	ConfirmPollPeriod   time.Duration
	OCR2CachePollPeriod time.Duration
	OCR2CacheTTL        time.Duration
	BalancePollPeriod   time.Duration
	RetentionPeriod     time.Duration
	ReapInterval        time.Duration
}

type ChainConfig struct {
	BroadcastChanSize   *uint64
	ConfirmPollPeriod   *config.Duration
	OCR2CachePollPeriod *time.Duration
	OCR2CacheTTL        *time.Duration
	BalancePollPeriod   *time.Duration
	RetentionPeriod     *time.Duration
	ReapInterval        *time.Duration
}

func (c *ChainConfig) SetDefaults() {
	if c.BroadcastChanSize == nil {
		c.BroadcastChanSize = &defaultConfigSet.BroadcastChanSize
	}
	if c.ConfirmPollPeriod == nil {
		c.ConfirmPollPeriod = config.MustNewDuration(defaultConfigSet.ConfirmPollPeriod)
	}
	if c.OCR2CachePollPeriod == nil {
		c.OCR2CachePollPeriod = &defaultConfigSet.OCR2CachePollPeriod
	}
	if c.OCR2CacheTTL == nil {
		c.OCR2CacheTTL = &defaultConfigSet.OCR2CacheTTL
	}
	if c.BalancePollPeriod == nil {
		c.BalancePollPeriod = &defaultConfigSet.BalancePollPeriod
	}
	if c.RetentionPeriod == nil {
		c.RetentionPeriod = &defaultConfigSet.RetentionPeriod
	}
	if c.ReapInterval == nil {
		c.ReapInterval = &defaultConfigSet.ReapInterval
	}
}

type NodeConfig struct {
	Name        *string
	URL         *config.URL
	SolidityURL *config.URL
}

func (n *NodeConfig) ValidateConfig() error {
	var err error
	if n.Name == nil {
		err = errors.Join(err, config.ErrMissing{Name: "Name", Msg: "required for all nodes"})
	} else if *n.Name == "" {
		err = errors.Join(err, config.ErrEmpty{Name: "Name", Msg: "required for all nodes"})
	}
	if n.URL == nil {
		err = errors.Join(err, config.ErrMissing{Name: "URL", Msg: "required for all nodes"})
	}
	if n.SolidityURL == nil {
		err = errors.Join(err, config.ErrMissing{Name: "SolidityURL", Msg: "required for all nodes"})
	}
	return err
}
