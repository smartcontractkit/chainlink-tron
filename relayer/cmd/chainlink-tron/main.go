package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/pelletier/go-toml/v2"

	"github.com/smartcontractkit/chainlink-common/pkg/beholder"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/types/core"
	"github.com/smartcontractkit/chainlink-tron/relayer/config"

	tronplugin "github.com/smartcontractkit/chainlink-tron/relayer/plugin"
)

const (
	loggerName = "PluginTron"
)

func main() {
	s := loop.MustNewStartedServer(loggerName)
	defer s.Stop()

	p := &pluginRelayer{Plugin: loop.Plugin{Logger: s.Logger}}
	defer s.Logger.ErrorIfFn(p.Close, "Failed to close")

	s.MustRegister(p)

	stopCh := make(chan struct{})
	defer close(stopCh)

	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: loop.PluginRelayerHandshakeConfig(),
		Plugins: map[string]plugin.Plugin{
			loop.PluginRelayerName: &loop.GRPCPluginRelayer{
				PluginServer: p,
				BrokerConfig: loop.BrokerConfig{
					StopCh:   stopCh,
					Logger:   s.Logger,
					GRPCOpts: s.GRPCOpts,
				},
			},
		},
		GRPCServer: s.GRPCOpts.NewServer,
	})
}

type pluginRelayer struct {
	loop.Plugin
}

var _ loop.PluginRelayer = &pluginRelayer{}

func (c *pluginRelayer) NewRelayer(ctx context.Context, configTOML string, keystore, csaKeystore core.Keystore, capabilityRegistry core.CapabilitiesRegistry) (loop.Relayer, error) {
	d := toml.NewDecoder(strings.NewReader(configTOML))
	d.DisallowUnknownFields()

	var cfg config.TOMLConfig

	if err := d.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config toml: %w:\n\t%s", err, configTOML)
	}

	if err := cfg.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid tron config: %w", err)
	}
	if !cfg.IsEnabled() {
		return nil, fmt.Errorf("cannot create new chain with ID %s: chain is disabled", *cfg.ChainID)
	}

	cfg.SetDefaults()

	rawNodes := make([]map[string]string, 0, len(cfg.Nodes))
	for _, n := range cfg.Nodes {
		if n == nil {
			continue
		}
		nodeURLs := make(map[string]string)
		if n.URL != nil {
			nodeURLs["URL"] = n.URL.String()
		}
		if n.SolidityURL != nil {
			nodeURLs["SolidityURL"] = n.SolidityURL.String()
		}
		if len(nodeURLs) == 0 {
			continue
		}
		rawNodes = append(rawNodes, nodeURLs)
	}
	chainID := ""
	if cfg.ChainID != nil {
		chainID = *cfg.ChainID
	}
	emitter := loop.NewPluginRelayerConfigEmitter(
		c.Logger,
		beholder.GetClient().Config.AuthPublicKeyHex,
		chainID,
		rawNodes,
	)
	if err := emitter.Start(ctx); err != nil {
		return nil, fmt.Errorf("failed to start plugin relayer config emitter: %w", err)
	}
	c.SubService(emitter)

	relayer, err := tronplugin.NewRelayer(&cfg, c.Logger, keystore)
	if err != nil {
		return nil, fmt.Errorf("failed to create relayer: %w", err)
	}

	c.SubService(relayer)

	return relayer, nil
}
