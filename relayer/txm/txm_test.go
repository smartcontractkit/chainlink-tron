package txm

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
	"github.com/stretchr/testify/require"
)

var log logger.Logger
var observedLogs observer.ObservedLogs
var config = TronTxmConfig{
	RPCAddress:        "",
	RPCInsecure:       true,
	BroadcastChanSize: 100,
	ConfirmPollSecs:   2,
}
var genesisAccountKey = testutils.CreateKey(rand.Reader)
var genesisAddress = genesisAccountKey.Address.String()
var genesisPrivateKey = genesisAccountKey.PrivateKey

func setup(t *testing.T, client *testutils.MockClient) (*TronTxm, *observer.ObservedLogs) {
	testLogger, observedlogs := logger.TestObserved(t, zapcore.DebugLevel)
	log = testLogger
	keystore := testutils.NewTestKeystore(genesisAddress, genesisPrivateKey)
	txm := TronTxm{
		logger:                log,
		keystore:              keystore,
		config:                config,
		estimateEnergyEnabled: true,

		client:        client,
		broadcastChan: make(chan *TronTx, config.BroadcastChanSize),
		accountStore:  newAccountStore(),
		stop:          make(chan struct{}),
	}

	err := txm.Start(context.Background())
	require.NoError(t, err)
	return &txm, observedlogs
}

func WaitForInflightTxs(txm *TronTxm, timeout time.Duration) {
	start := time.Now()
	for {
		queueLen, unconfirmedLen := txm.InflightCount()
		log.Debugw("Inflight count", "queued", queueLen, "unconfirmed", unconfirmedLen)
		if queueLen == 0 && unconfirmedLen == 0 {
			break
		}
		if time.Since(start) > timeout {
			panic("Timeout waiting for inflight txs")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func TestTxm_InvalidInputParams(t *testing.T) {
	txm, _ := setup(t, testutils.NewMockClient())
	err := txm.Enqueue(genesisAddress, genesisAddress, "foo()", "param1")
	require.Error(t, err)
	require.ErrorContains(t, err, "odd number of params")
}

func TestTxm_Success(t *testing.T) {
	txm, logs := setup(t, testutils.NewMockClient())

	err := txm.Enqueue(genesisAddress, genesisAddress, "foo()")
	require.NoError(t, err)

	WaitForInflightTxs(txm, 10*time.Second)

	require.Equal(t, logs.FilterMessageSnippet("retry").Len(), 0)
	require.Equal(t, logs.FilterMessageSnippet("confirmed transaction").Len(), 1)
}

func TestTxm_RetryOnBroadcastServerBusy(t *testing.T) {
	grpcClient := testutils.NewMockClient()
	grpcClient.SetBroadcastResp(false, api.Return_SERVER_BUSY, []byte("server busy"))
	txm, logs := setup(t, grpcClient)

	err := txm.Enqueue(genesisAddress, genesisAddress, "foo()")
	require.NoError(t, err)

	WaitForInflightTxs(txm, 10*time.Second)

	require.Equal(t, logs.FilterMessageSnippet("SERVER_BUSY or BLOCK_UNSOLIDIFIED: adding transaction to retry queue").Len(), 5)
	require.Equal(t, logs.FilterMessageSnippet("not retrying, already reached max retries").Len(), 1)
}

func TestTxm_RetryOnBroadcastBlockUnsolidifed(t *testing.T) {
	grpcClient := testutils.NewMockClient()
	grpcClient.SetBroadcastResp(false, api.Return_BLOCK_UNSOLIDIFIED, []byte("block unsolid"))
	txm, logs := setup(t, grpcClient)

	err := txm.Enqueue(genesisAddress, genesisAddress, "foo()")
	require.NoError(t, err)

	WaitForInflightTxs(txm, 10*time.Second)

	require.Equal(t, logs.FilterMessageSnippet("SERVER_BUSY or BLOCK_UNSOLIDIFIED: adding transaction to retry queue").Len(), 5)
	require.Equal(t, logs.FilterMessageSnippet("not retrying, already reached max retries").Len(), 1)
}

func TestTxm_NoRetryOnOtherBroadcastErr(t *testing.T) {
	grpcClient := testutils.NewMockClient()
	grpcClient.SetBroadcastResp(false, api.Return_BANDWITH_ERROR, []byte("some error"))
	txm, logs := setup(t, grpcClient)

	err := txm.Enqueue(genesisAddress, genesisAddress, "foo()")
	require.NoError(t, err)

	WaitForInflightTxs(txm, 10*time.Second)

	require.Equal(t, logs.FilterMessageSnippet("not retrying, already reached max retries").Len(), 0)
	require.Equal(t, logs.FilterMessageSnippet("adding transaction to retry queue").Len(), 0)
	require.Equal(t, logs.FilterMessageSnippet("transaction failed to broadcast").Len(), 1)
}
