package plugin

import (
	"errors"
	"time"

	"github.com/smartcontractkit/chainlink-common/pkg/config"
)

// Global solana defaults.
var defaultConfigSet = chainConfigSet{
	// transaction broadcast channel size
	BroadcastChanSize: 4096,
	// polling period for transaction confirmation
	ConfirmPollPeriod: 500 * time.Millisecond, // polling for tx confirmation
}

// opt: remove
type chainConfigSet struct {
	BroadcastChanSize uint64
	ConfirmPollPeriod time.Duration
}

type ChainConfig struct {
	BroadcastChanSize *uint64
	ConfirmPollPeriod *config.Duration
}

func (c *ChainConfig) SetDefaults() {
	if c.BroadcastChanSize == nil {
		c.BroadcastChanSize = &defaultConfigSet.BroadcastChanSize
	}
	if c.ConfirmPollPeriod == nil {
		c.ConfirmPollPeriod = config.MustNewDuration(defaultConfigSet.ConfirmPollPeriod)
	}
}

type NodeConfig struct {
	Name        *string
	URL         *config.URL
	SolidityURL *config.URL
	JsonRpcURL  *config.URL
}

func (n *NodeConfig) ValidateConfig() (err error) {
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
	if n.JsonRpcURL == nil {
		err = errors.Join(err, config.ErrMissing{Name: "JsonRpcURL", Msg: "required for all nodes"})
	}
	return
}
