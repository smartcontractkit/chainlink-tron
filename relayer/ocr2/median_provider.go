package ocr2

import (
	"context"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	relaytypes "github.com/smartcontractkit/chainlink-common/pkg/types"
	"github.com/smartcontractkit/chainlink-tron/relayer/txm"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median/evmreportcodec"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"
)

var _ relaytypes.MedianProvider = (*medianProvider)(nil)

type medianProvider struct {
	*configProvider
	transmitter        ocrtypes.ContractTransmitter
	transmissionsCache *transmissionsCache
	reportCodec        median.ReportCodec
}

func NewMedianProvider(
	ctx context.Context,
	cfg Config,
	medianContract median.MedianContract,
	configProvider *configProvider,
	contractAddress address.Address,
	senderAddress address.Address,
	txm *txm.TronTxm,
	lggr logger.Logger,
) *medianProvider {
	lggr = logger.Named(lggr, "MedianProvider")
	cache := NewTransmissionsCache(cfg, medianContract, lggr)
	//transmitter := NewOCRContractTransmitter(ctx, cache, contractAddress, senderAddress, txm, lggr)

	return &medianProvider{
		configProvider: configProvider,
		//transmitter:        transmitter,
		transmissionsCache: cache,
		reportCodec:        evmreportcodec.ReportCodec{},
	}
}

func (p *medianProvider) Name() string {
	return p.lggr.Name()
}

func (p *medianProvider) Start(context.Context) error {
	return p.StartOnce("MedianProvider", func() error {
		p.lggr.Debugf("Median provider starting")
		// starting both cache services here
		// todo: find a better way
		if err := p.configProvider.contractCache.Start(); err != nil {
			return fmt.Errorf("couldn't start contractCache: %w", err)
		}
		return p.transmissionsCache.Start()
	})
}

func (p *medianProvider) Close() error {
	return p.StopOnce("MedianProvider", func() error {
		p.lggr.Debugf("Median provider stopping")
		// stopping both cache services here
		// todo: find a better way
		if err := p.configProvider.contractCache.Close(); err != nil {
			return fmt.Errorf("coulnd't stop contractCache: %w", err)
		}
		return p.transmissionsCache.Close()
	})
}

func (p *medianProvider) HealthReport() map[string]error {
	return map[string]error{p.Name(): p.Healthy()}
}

func (p *medianProvider) ContractTransmitter() ocrtypes.ContractTransmitter {
	return p.transmitter
}

func (p *medianProvider) ReportCodec() median.ReportCodec {
	return p.reportCodec
}

func (p *medianProvider) MedianContract() median.MedianContract {
	return p.transmissionsCache
}

func (p *medianProvider) OnchainConfigCodec() median.OnchainConfigCodec {
	return median.StandardOnchainConfigCodec{}
}

func (p *medianProvider) ContractReader() relaytypes.ContractReader {
	return nil
}

func (p *medianProvider) Codec() relaytypes.Codec {
	return nil
}
