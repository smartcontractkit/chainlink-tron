package ocr2

import (
	"context"

	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/types"
)

type pluginProvider struct {
	services.Service
	chainReader         types.ContractReader
	codec               types.Codec
	contractTransmitter ocrtypes.ContractTransmitter
	configProvider      types.ConfigProvider
	lggr                logger.Logger
	ms                  services.MultiStart
}

var _ types.PluginProvider = (*pluginProvider)(nil)

func NewPluginProvider(
	chainReader types.ContractReader,
	codec types.Codec,
	contractTransmitter ocrtypes.ContractTransmitter,
	configProvider types.ConfigProvider,
	lggr logger.Logger,
) *pluginProvider {
	return &pluginProvider{
		chainReader:         chainReader,
		codec:               codec,
		contractTransmitter: contractTransmitter,
		configProvider:      configProvider,
		lggr:                lggr,
		ms:                  services.MultiStart{},
	}
}

func (p *pluginProvider) Name() string { return p.lggr.Name() }

func (p *pluginProvider) Ready() error { return nil }

func (p *pluginProvider) HealthReport() map[string]error {
	hp := map[string]error{p.Name(): p.Ready()}
	services.CopyHealth(hp, p.configProvider.HealthReport())
	return hp
}

func (p *pluginProvider) ContractTransmitter() ocrtypes.ContractTransmitter {
	return p.contractTransmitter
}

func (p *pluginProvider) OffchainConfigDigester() ocrtypes.OffchainConfigDigester {
	return p.configProvider.OffchainConfigDigester()
}

func (p *pluginProvider) ContractConfigTracker() ocrtypes.ContractConfigTracker {
	return p.configProvider.ContractConfigTracker()
}

func (p *pluginProvider) ContractReader() types.ContractReader {
	return p.chainReader
}

func (p *pluginProvider) Codec() types.Codec {
	return p.codec
}

func (p *pluginProvider) Start(ctx context.Context) error {
	return p.configProvider.Start(ctx)
}

func (p *pluginProvider) Close() error {
	return p.configProvider.Close()
}
