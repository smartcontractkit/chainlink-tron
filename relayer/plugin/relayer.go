package plugin

import (
	"context"
	"errors"
	"math/big"

	"github.com/pelletier/go-toml/v2"

	"github.com/smartcontractkit/chainlink-common/pkg/chains"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/types"
	"github.com/smartcontractkit/chainlink-common/pkg/types/core"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
)

type TronRelayer struct {
	services.StateMachine

	id   string
	cfg  *TOMLConfig
	lggr logger.Logger

	txm *txm.TronTxm
}

var _ loop.Relayer = &TronRelayer{}

func NewRelayer(cfg *TOMLConfig, lggr logger.Logger, keystore core.Keystore) *TronRelayer {
	id := *cfg.ChainID
	return &TronRelayer{
		id:   id,
		cfg:  cfg,
		lggr: logger.Named(logger.With(lggr, "chainID", id, "chain", "tron"), "TronRelayer"),
	}
}

// Service interface
func (t *TronRelayer) Name() string {
	return t.lggr.Name()
}

func (t *TronRelayer) Start(ctx context.Context) error {
	return t.StartOnce("TronRelayer", func() error {
		t.lggr.Debug("Starting")
		t.lggr.Debug("Starting txm")
		t.lggr.Debug("Starting balance monitor")
		var ms services.MultiStart
		return ms.Start(ctx, t.txm)
	})
}

func (t *TronRelayer) Close() error {
	return t.StopOnce("Chain", func() error {
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
		ID:      t.id,
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
		stat, err := nodeStatus(node, t.id)
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

func (t *TronRelayer) NewConfigProvider(context.Context, types.RelayArgs) (types.ConfigProvider, error) {
	return nil, errors.New("TODO")
}

func (t *TronRelayer) NewPluginProvider(context.Context, types.RelayArgs, types.PluginArgs) (types.PluginProvider, error) {
	return nil, errors.New("TODO")
}

func (T *TronRelayer) NewLLOProvider(context.Context, types.RelayArgs, types.PluginArgs) (types.LLOProvider, error) {
	return nil, errors.New("TODO")
}
