package plugin

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/pelletier/go-toml/v2"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/smartcontractkit/chainlink-common/pkg/chains"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/types"
	"github.com/smartcontractkit/chainlink-common/pkg/types/core"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/ocr2"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
)

type TronRelayer struct {
	services.StateMachine

	id   string
	cfg  *TOMLConfig
	lggr logger.Logger

	client *client.GrpcClient
	txm    *txm.TronTxm
}

var _ loop.Relayer = &TronRelayer{}

func NewRelayer(cfg *TOMLConfig, lggr logger.Logger, keystore core.Keystore) (*TronRelayer, error) {
	id := *cfg.ChainID

	nodeConfig, err := cfg.ListNodes().SelectRandom()
	if err != nil {
		return nil, fmt.Errorf("failed to get node config: %w", err)
	}
	client, err := createClient(nodeConfig.URL.URL())
	if err != nil {
		return nil, fmt.Errorf("error in NewConfigProvider chain.Reader: %w", err)
	}

	txmgr := txm.New(lggr, keystore, client, txm.TronTxmConfig{
		// TODO: stop changing uint64 fields here to uint?
		BroadcastChanSize: uint(cfg.BroadcastChanSize()),
		ConfirmPollSecs:   uint(cfg.ConfirmPollPeriod().Seconds()),
	})

	return &TronRelayer{
		id:     id,
		cfg:    cfg,
		lggr:   logger.Named(logger.With(lggr, "chainID", id, "chain", "tron"), "TronRelayer"),
		client: client,
		txm:    txmgr,
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

func (t *TronRelayer) NewConfigProvider(ctx context.Context, args types.RelayArgs) (types.ConfigProvider, error) {
	// todo: unmarshal args.RelayConfig into a struct if required
	reader := relayer.NewReader(t.client, t.lggr)
	chainID, err := strconv.ParseUint(t.id, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse chain id %s as uint64: %w", t.id, err)
	}
	contractAddress, err := address.Base58ToAddress(args.ContractID)
	if err != nil {
		return nil, fmt.Errorf("couldn't parse contract id %s as base58 Tron address: %w", args.ContractID, err)
	}

	configProvider, err := ocr2.NewConfigProvider(chainID, contractAddress, reader, t.cfg, t.lggr)
	if err != nil {
		return nil, fmt.Errorf("coudln't initialize ConfigProvider: %w", err)
	}

	return configProvider, nil
}

func (t *TronRelayer) NewPluginProvider(context.Context, types.RelayArgs, types.PluginArgs) (types.PluginProvider, error) {
	return nil, errors.New("TODO")
}

func (t *TronRelayer) NewLLOProvider(context.Context, types.RelayArgs, types.PluginArgs) (types.LLOProvider, error) {
	return nil, errors.New("TODO")
}

// createClient returns a TRON gRPC client
func createClient(grpcUrl *url.URL) (*client.GrpcClient, error) {
	// TODO: we expect the URL in the format grpc://host:port?insecure=true, move this somewhere?
	hostname := grpcUrl.Hostname()
	port := grpcUrl.Port()
	if port == "" {
		port = "50051"
	}

	insecureTransport := false
	values := grpcUrl.Query()
	insecureValues, ok := values["insecure"]
	if ok {
		if len(insecureValues) > 0 {
			insecureValue := strings.ToLower(insecureValues[0])
			insecureTransport = insecureValue == "true" || insecureValue == "1"
		}
	}

	var transportCredentials credentials.TransportCredentials
	if insecureTransport {
		transportCredentials = insecure.NewCredentials()
	} else {
		transportCredentials = credentials.NewTLS(nil)
	}

	grpcClient := client.NewGrpcClientWithTimeout(hostname+":"+port, 15*time.Second)
	err := grpcClient.Start(grpc.WithTransportCredentials(transportCredentials))
	if err != nil {
		return nil, fmt.Errorf("failed to start GrpcClient: %+w", err)
	}

	return grpcClient, nil
}
