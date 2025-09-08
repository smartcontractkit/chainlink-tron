package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-plugin"
	"github.com/pelletier/go-toml/v2"

	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/types/core"

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

	// Add connection monitoring to detect when gRPC connection is lost
	// connLostCh := make(chan struct{})
	
	// // Monitor for connection loss and trigger cleanup
	// go func() {
	// 	select {
	// 	case <-connLostCh:
	// 		s.Logger.Infow("gRPC connection lost, triggering plugin cleanup")
	// 		s.Logger.ErrorIfFn(p.Close, "Failed to close plugin on connection loss")
	// 	case <-stopCh:
	// 		// Normal shutdown, no need to trigger cleanup
	// 	}
	// }()

	// Start the plugin server in a goroutine so we can monitor it
	// serverDone := make(chan struct{})
	// go func() {
	// 	defer close(serverDone)
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
	// }()

	// // Monitor the server - if it exits unexpectedly, it means connection was lost
	// select {
	// case <-serverDone:
	// 	s.Logger.Infow("Plugin server exited, connection likely lost")
	// 	close(connLostCh)
	// case <-stopCh:
	// 	s.Logger.Infow("Stop channel closed, normal shutdown")
	// }
}

type pluginRelayer struct {
	loop.Plugin
}

var _ loop.PluginRelayer = &pluginRelayer{}

func (c *pluginRelayer) NewRelayer(ctx context.Context, config string, keystore, csaKeystore core.Keystore, capabilityRegistry core.CapabilitiesRegistry) (loop.Relayer, error) {
	c.Logger.Infow("Creating new TronRelayer instance", "config", config)
	d := toml.NewDecoder(strings.NewReader(config))
	d.DisallowUnknownFields()

	var cfg tronplugin.TOMLConfig

	if err := d.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("failed to decode config toml: %w:\n\t%s", err, config)
	}

	if err := cfg.ValidateConfig(); err != nil {
		return nil, fmt.Errorf("invalid tron config: %w", err)
	}
	if !cfg.IsEnabled() {
		return nil, fmt.Errorf("cannot create new chain with ID %s: chain is disabled", *cfg.ChainID)
	}

	cfg.SetDefaults()

	c.Logger.Infow("Calling NewRelayer", "cfg", cfg)
	relayer, err := tronplugin.NewRelayer(&cfg, c.Logger, keystore)
	if err != nil {
		return nil, fmt.Errorf("failed to create relayer: %w", err)
	}
	c.Logger.Infow("NewRelayer returned", "relayer", relayer, "instance_pointer", fmt.Sprintf("%p", relayer))

	c.SubService(relayer)

	return relayer, nil
}
