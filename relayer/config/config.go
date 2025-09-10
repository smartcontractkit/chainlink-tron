package config

import (
	"errors"
	"log"
	"strings"

	"github.com/smartcontractkit/chainlink-common/pkg/config"
	"github.com/smartcontractkit/chainlink-common/pkg/config/configtest"
)

var defaults TOMLConfig

func init() {
	if err := configtest.DocDefaultsOnly(strings.NewReader(docsTOML), &defaults, config.DecodeTOML); err != nil {
		log.Fatalf("Failed to initialize defaults from docs: %v", err)
	}
}

func Defaults() (c TOMLConfig) {
	c.SetFrom(&defaults)
	return
}

type ChainConfig struct {
	BalancePollPeriod   *config.Duration
	BroadcastChanSize   *uint64
	ConfirmPollPeriod   *config.Duration
	OCR2CachePollPeriod *config.Duration
	OCR2CacheTTL        *config.Duration
	RetentionPeriod     *config.Duration
	ReapInterval        *config.Duration
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
