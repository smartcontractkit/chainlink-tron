package plugin

import (
	"errors"
	"fmt"
	"math/rand"
	"net/url"
	"time"

	"github.com/pelletier/go-toml/v2"
	"golang.org/x/exp/slices"

	"github.com/smartcontractkit/chainlink-common/pkg/config"
	"github.com/smartcontractkit/chainlink-tron/relayer/ocr2"
)

type TOMLConfigs []*TOMLConfig

func (cs TOMLConfigs) ValidateConfig() error {
	return cs.validateKeys()
}

func (cs TOMLConfigs) validateKeys() error {
	var err error
	// Unique chain IDs
	chainIDs := config.UniqueStrings{}
	for i, c := range cs {
		if chainIDs.IsDupe(c.ChainID) {
			err = errors.Join(err, config.NewErrDuplicate(fmt.Sprintf("%d.ChainID", i), *c.ChainID))
		}
	}

	// Unique node names
	names := config.UniqueStrings{}
	for i, c := range cs {
		for j, n := range c.Nodes {
			if names.IsDupe(n.Name) {
				err = errors.Join(err, config.NewErrDuplicate(fmt.Sprintf("%d.Nodes.%d.Name", i, j), *n.Name))
			}
		}
	}

	// Unique URLs
	urls := config.UniqueStrings{}
	for i, c := range cs {
		for j, n := range c.Nodes {
			u := (*url.URL)(n.URL)
			if urls.IsDupeFmt(u) {
				err = errors.Join(err, config.NewErrDuplicate(fmt.Sprintf("%d.Nodes.%d.URL", i, j), u.String()))
			}
		}
	}
	return err
}

func (cs *TOMLConfigs) SetFrom(fs *TOMLConfigs) error {
	if err1 := fs.validateKeys(); err1 != nil {
		return err1
	}
	for _, f := range *fs {
		if f.ChainID == nil {
			*cs = append(*cs, f)
		} else if i := slices.IndexFunc(*cs, func(c *TOMLConfig) bool {
			return c.ChainID != nil && *c.ChainID == *f.ChainID
		}); i == -1 {
			*cs = append(*cs, f)
		} else {
			(*cs)[i].SetFrom(f)
		}
	}
	return nil
}

type NodeConfigs []*NodeConfig

func (ns *NodeConfigs) SetFrom(fs *NodeConfigs) {
	for _, f := range *fs {
		if f.Name == nil {
			*ns = append(*ns, f)
		} else if i := slices.IndexFunc(*ns, func(n *NodeConfig) bool {
			return n.Name != nil && *n.Name == *f.Name
		}); i == -1 {
			*ns = append(*ns, f)
		} else {
			setFromNode((*ns)[i], f)
		}
	}
}

func (ns NodeConfigs) SelectRandom() (*NodeConfig, error) {
	if len(ns) == 0 {
		return nil, errors.New("no nodes available")
	}

	index := rand.Perm(len(ns))
	node := ns[index[0]]

	return node, nil
}

func setFromNode(n, f *NodeConfig) {
	if f.Name != nil {
		n.Name = f.Name
	}
	if f.URL != nil {
		n.URL = f.URL
	}
	if f.SolidityURL != nil {
		n.SolidityURL = f.SolidityURL
	}
}

type TOMLConfig struct {
	ChainID *string
	// Do not access directly, use [IsEnabled]
	Enabled *bool
	ChainConfig
	Nodes NodeConfigs
}

var _ ocr2.Config = (*TOMLConfig)(nil)

func (c *TOMLConfig) IsEnabled() bool {
	return c.Enabled == nil || *c.Enabled
}

func (c *TOMLConfig) SetFrom(f *TOMLConfig) {
	if f.ChainID != nil {
		c.ChainID = f.ChainID
	}
	if f.Enabled != nil {
		c.Enabled = f.Enabled
	}
	setFromChain(&c.ChainConfig, &f.ChainConfig)
	c.Nodes.SetFrom(&f.Nodes)
}

func setFromChain(c, f *ChainConfig) {
	if f.BalancePollPeriod != nil {
		c.BalancePollPeriod = f.BalancePollPeriod
	}
	if f.BroadcastChanSize != nil {
		c.BroadcastChanSize = f.BroadcastChanSize
	}
	if f.ConfirmPollPeriod != nil {
		c.ConfirmPollPeriod = f.ConfirmPollPeriod
	}
	if f.OCR2CachePollPeriod != nil {
		c.OCR2CachePollPeriod = f.OCR2CachePollPeriod
	}
	if f.OCR2CacheTTL != nil {
		c.OCR2CacheTTL = f.OCR2CacheTTL
	}
	if f.RetentionPeriod != nil {
		c.RetentionPeriod = f.RetentionPeriod
	}
	if f.ReapInterval != nil {
		c.ReapInterval = f.ReapInterval
	}
}

func (c *TOMLConfig) ValidateConfig() error {
	var err error
	if c.ChainID == nil {
		err = errors.Join(err, config.ErrMissing{Name: "ChainID", Msg: "required for all chains"})
	} else if *c.ChainID == "" {
		err = errors.Join(err, config.ErrEmpty{Name: "ChainID", Msg: "required for all chains"})
	}

	if len(c.Nodes) == 0 {
		err = errors.Join(err, config.ErrMissing{Name: "Nodes", Msg: "must have at least one node"})
	} else {
		for _, node := range c.Nodes {
			err = errors.Join(err, node.ValidateConfig())
		}
	}

	return err
}

func (c *TOMLConfig) TOMLString() (string, error) {
	b, err := toml.Marshal(c)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func (c *TOMLConfig) BalancePollPeriod() time.Duration {
	return c.ChainConfig.BalancePollPeriod.Duration()
}

func (c *TOMLConfig) BroadcastChanSize() uint64 {
	return *c.ChainConfig.BroadcastChanSize
}

func (c *TOMLConfig) ConfirmPollPeriod() time.Duration {
	return c.ChainConfig.ConfirmPollPeriod.Duration()
}

func (c *TOMLConfig) ListNodes() NodeConfigs {
	return c.Nodes
}

func (c *TOMLConfig) OCR2CachePollPeriod() time.Duration {
	return c.ChainConfig.OCR2CachePollPeriod.Duration()
}

func (c *TOMLConfig) OCR2CacheTTL() time.Duration {
	return c.ChainConfig.OCR2CacheTTL.Duration()
}

func (c *TOMLConfig) RetentionPeriod() time.Duration {
	return c.ChainConfig.RetentionPeriod.Duration()
}

func (c *TOMLConfig) ReapInterval() time.Duration {
	return c.ChainConfig.ReapInterval.Duration()
}

func (c *TOMLConfig) SetDefaults() {
	c.ChainConfig.SetDefaults()
}

func NewDefault() *TOMLConfig {
	cfg := &TOMLConfig{}
	cfg.SetDefaults()
	return cfg
}
