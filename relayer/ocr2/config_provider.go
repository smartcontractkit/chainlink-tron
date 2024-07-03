package ocr2

import (
	"context"

	"github.com/fbsobreira/gotron-sdk/pkg/address"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	commontypes "github.com/smartcontractkit/chainlink-common/pkg/types"
	"github.com/smartcontractkit/chainlink-common/pkg/utils"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/reader"
)

// type ConfigProvider interface {
// 	Service
// 	OffchainConfigDigester() ocrtypes.OffchainConfigDigester
// 	ContractConfigTracker() ocrtypes.ContractConfigTracker
// }

var _ commontypes.ConfigProvider = (*configProvider)(nil)

type configProvider struct {
	utils.StartStopOnce

	reader        ContractReader
	contractCache *contractCache
	digester      ocrtypes.OffchainConfigDigester

	lggr logger.Logger
}

func NewConfigProvider(chainID uint64, contractAddress address.Address, readerClient reader.Reader, cfg Config, lggr logger.Logger) (*configProvider, error) {
	lggr = logger.Named(lggr, "ConfigProvider")
	client := NewOCR2Reader(readerClient, lggr)
	contractReader := NewContractReader(contractAddress, client, lggr)
	cache := NewContractCache(cfg, contractReader, lggr)

	evmContractAddress := relayer.TronToEVMAddress(contractAddress)

	// todo: investigate if there are any issues with using the evm offchain config digester for Tron
	offchainConfigDigester := evmutil.EVMOffchainConfigDigester{
		ChainID:         chainID,
		ContractAddress: evmContractAddress,
	}

	return &configProvider{
		reader:        contractReader,
		contractCache: cache,
		digester:      offchainConfigDigester,
		lggr:          lggr,
	}, nil
}

func (p *configProvider) Name() string {
	return p.lggr.Name()
}

func (p *configProvider) Start(context.Context) error {
	return p.StartOnce("ConfigProvider", func() error {
		p.lggr.Debugf("Config provider starting")
		return p.contractCache.Start()
	})
}

func (p *configProvider) Close() error {
	return p.StopOnce("ConfigProvider", func() error {
		p.lggr.Debugf("Config provider stopping")
		return p.contractCache.Close()
	})
}

func (p *configProvider) HealthReport() map[string]error {
	return map[string]error{p.Name(): p.Healthy()}
}

func (p *configProvider) ContractConfigTracker() ocrtypes.ContractConfigTracker {
	return p.contractCache
}

func (p *configProvider) OffchainConfigDigester() ocrtypes.OffchainConfigDigester {
	return p.digester
}
