package plugin

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/pelletier/go-toml/v2"

	"github.com/smartcontractkit/chainlink-common/pkg/chains"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/types"
	"github.com/smartcontractkit/chainlink-common/pkg/types/core"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/ocr2"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/reader"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
)

type TronRelayer struct {
	services.StateMachine

	chainId    string
	chainIdNum *big.Int

	cfg  *TOMLConfig
	lggr logger.Logger

	client sdk.GrpcClient
	txm    *txm.TronTxm
}

var _ loop.Relayer = &TronRelayer{}

func NewRelayer(cfg *TOMLConfig, lggr logger.Logger, keystore core.Keystore) (*TronRelayer, error) {
	id := *cfg.ChainID

	var idNum *big.Int
	if strings.HasPrefix(id, "0x") {
		// TODO: we can just do new(big.Int).SetString(.., 16)
		idBytes, err := hex.DecodeString(id[2:])
		if err != nil {
			return nil, fmt.Errorf("couldn't parse hex chain id %s: %w", id, err)
		}
		idNum = new(big.Int).SetBytes(idBytes)
	} else {
		parsedNum, ok := new(big.Int).SetString(id, 10)
		if !ok {
			return nil, fmt.Errorf("couldn't parse numeric chain id %s", id)
		}
		idNum = parsedNum
	}

	nodeConfig, err := cfg.ListNodes().SelectRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to get node config: %w", err)
	}
	client, err := sdk.CreateCombinedGrpcClient(nodeConfig.URL.URL(), nodeConfig.SolidityURL.URL())
	if err != nil {
		return nil, fmt.Errorf("error in NewConfigProvider chain.Reader: %w", err)
	}

	txmgr := txm.New(lggr, keystore, client, txm.TronTxmConfig{
		// TODO: stop changing uint64 fields here to uint?
		BroadcastChanSize: uint(cfg.BroadcastChanSize()),
		ConfirmPollSecs:   uint(cfg.ConfirmPollPeriod().Seconds()),
	})

	return &TronRelayer{
		chainId:    id,
		chainIdNum: idNum,
		cfg:        cfg,
		lggr:       logger.Named(logger.With(lggr, "chainID", id, "chain", "tron"), "TronRelayer"),
		client:     client,
		txm:        txmgr,
	}, nil
}

// Service interface
func (t *TronRelayer) Name() string {
	return t.lggr.Name()
}

func (t *TronRelayer) Start(ctx context.Context) error {
	return t.StartOnce("TronRelayer", func() error {
		t.lggr.Debug("Starting")
		t.lggr.Debug("Starting txm")
		var ms services.MultiStart
		return ms.Start(ctx, t.txm)
	})
}

func (t *TronRelayer) Close() error {
	return t.StopOnce("TronRelayer", func() error {
		t.lggr.Debug("Stopping")
		t.lggr.Debug("Stopping txm")
		return services.CloseAll(t.txm)
	})
}

func (t *TronRelayer) Ready() error {
	return errors.Join(
		t.StateMachine.Ready(),
		t.txm.Ready(),
	)
}

func (t *TronRelayer) HealthReport() map[string]error {
	report := map[string]error{t.Name(): t.Healthy()}
	services.CopyHealth(report, t.txm.HealthReport())
	return report
}

// ChainService interface
func (t *TronRelayer) GetChainStatus(ctx context.Context) (types.ChainStatus, error) {
	toml, err := t.cfg.TOMLString()
	if err != nil {
		return types.ChainStatus{}, err
	}
	return types.ChainStatus{
		ID:      t.chainId,
		Enabled: t.cfg.IsEnabled(),
		Config:  toml,
	}, nil
}

func (t *TronRelayer) ListNodeStatuses(ctx context.Context, pageSize int32, pageToken string) (stats []types.NodeStatus, nextPageToken string, total int, err error) {
	return chains.ListNodeStatuses(int(pageSize), pageToken, t.listNodeStatuses)
}

func (t *TronRelayer) Transact(ctx context.Context, from, to string, amount *big.Int, balanceCheck bool) error {
	return errors.New("TODO")
}

func (t *TronRelayer) listNodeStatuses(start, end int) ([]types.NodeStatus, int, error) {
	stats := make([]types.NodeStatus, 0)
	total := len(t.cfg.Nodes)
	if start >= total {
		return stats, total, chains.ErrOutOfRange
	}
	if end > total {
		end = total
	}
	nodes := t.cfg.Nodes[start:end]
	for _, node := range nodes {
		stat, err := nodeStatus(node, t.chainId)
		if err != nil {
			return stats, total, err
		}
		stats = append(stats, stat)
	}
	return stats, total, nil
}

func nodeStatus(n *NodeConfig, id string) (types.NodeStatus, error) {
	// TODO: why does marshalling the config tell us the node status?
	var s types.NodeStatus
	s.ChainID = id
	s.Name = *n.Name
	b, err := toml.Marshal(n)
	if err != nil {
		return types.NodeStatus{}, err
	}
	s.Config = string(b)
	return s, nil
}

// Relayer interface
func (t *TronRelayer) NewContractReader(ctx context.Context, contractReaderConfig []byte) (types.ContractReader, error) {
	return nil, errors.New("TODO")
}

func (t *TronRelayer) NewConfigProvider(ctx context.Context, args types.RelayArgs) (types.ConfigProvider, error) {
	// todo: unmarshal args.RelayConfig into a struct if required
	reader := reader.NewReader(t.client, t.lggr)
	contractAddress, err := address.Base58ToAddress(args.ContractID)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse contract id %s as base58 Tron address: %w", args.ContractID, err)
	}

	configProvider, err := ocr2.NewConfigProvider(t.chainIdNum, contractAddress, reader, t.cfg, t.lggr)
	if err != nil {
		return nil, fmt.Errorf("coudln't initialize ConfigProvider: %w", err)
	}

	return configProvider, nil
}

func (t *TronRelayer) NewPluginProvider(ctx context.Context, relayargs types.RelayArgs, pluginargs types.PluginArgs) (types.PluginProvider, error) {
	// TODO: is this necessary? should we just return an error?
	return t.NewMedianProvider(ctx, relayargs, pluginargs)
}

func (t *TronRelayer) NewLLOProvider(context.Context, types.RelayArgs, types.PluginArgs) (types.LLOProvider, error) {
	return nil, errors.New("TODO")
}

// implement MedianProvider type from github.com/smartcontractkit/chainlink-common/pkg/loop/internal/types
//
// if the loop.Relayer returned by NewRelayer supports the internal loop type MedianProvider, it's called here:
// see https://github.com/smartcontractkit/chainlink-common/blob/7c11e2c2ce3677f57239c40585b04fd1c9ce1713/pkg/loop/internal/relayer/relayer.go#L493
func (t *TronRelayer) NewMedianProvider(ctx context.Context, relayargs types.RelayArgs, pluginargs types.PluginArgs) (types.MedianProvider, error) {
	// todo: unmarshal args.RelayConfig if required
	reader := reader.NewReader(t.client, t.lggr)
	contractAddress, err := address.Base58ToAddress(relayargs.ContractID)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse contract id %s as base58 Tron address: %w", relayargs.ContractID, err)
	}
	configProvider, err := ocr2.NewConfigProvider(t.chainIdNum, contractAddress, reader, t.cfg, t.lggr)
	if err != nil {
		return nil, fmt.Errorf("coudln't initialize ConfigProvider: %w", err)
	}
	senderAddress, err := address.Base58ToAddress(pluginargs.TransmitterID)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse transmitter id %s as base58 Tron address: %w", pluginargs.TransmitterID, err)
	}
	ocr2Reader := ocr2.NewOCR2Reader(reader, t.lggr)
	medianContract := ocr2.NewContractReader(contractAddress, ocr2Reader, t.lggr)
	medianProvider := ocr2.NewMedianProvider(ctx, t.cfg, medianContract, configProvider, contractAddress, senderAddress, t.txm, t.lggr)

	return medianProvider, nil
}
