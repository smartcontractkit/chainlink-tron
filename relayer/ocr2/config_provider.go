package ocr2

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	commontypes "github.com/smartcontractkit/chainlink-common/pkg/types"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/libocr/offchainreporting2/chains/evmutil"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-common/pkg/utils"
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

func NewConfigProvider(chainID string, contractAddress address.Address, reader relayer.Reader, cfg Config, lggr logger.Logger) (*configProvider, error) {
	lggr = logger.Named(lggr, "ConfigProvider")
	client := NewOCR2Reader(reader, lggr)
	contractReader := NewContractReader(contractAddress, client, lggr)
	cache := NewContractCache(cfg, contractReader, lggr)

	// todo: accept tron addr format and convert to EVM addr
	isEVMAddr := common.IsHexAddress(contractAddress.Hex())
	if !isEVMAddr {
		return nil, fmt.Errorf("contract address is not a valid EVM address: %s", contractAddress)
	}
	offchainConfigDigester := evmutil.EVMOffchainConfigDigester{
		ChainID:         1, // todo
		ContractAddress: common.HexToAddress(contractAddress.Hex()),
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
