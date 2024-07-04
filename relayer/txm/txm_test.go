package txm_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/mocks"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
	trontxm "github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var keystore *testutils.TestKeystore
var config = trontxm.TronTxmConfig{
	BroadcastChanSize: 100,
	ConfirmPollSecs:   2,
}
var genesisAccountKey = testutils.CreateKey(rand.Reader)
var genesisAddress = genesisAccountKey.Address.String()
var genesisPrivateKey = genesisAccountKey.PrivateKey

func WaitForInflightTxs(logger logger.Logger, txm *trontxm.TronTxm, timeout time.Duration) {
	start := time.Now()
	for {
		queueLen, unconfirmedLen := txm.InflightCount()
		logger.Debugw("Inflight count", "queued", queueLen, "unconfirmed", unconfirmedLen)
		if queueLen == 0 && unconfirmedLen == 0 {
			break
		}
		if time.Since(start) > timeout {
			panic("Timeout waiting for inflight txs")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func waitForMaxRetryDuration() {
	time.Sleep(trontxm.MAX_BROADCAST_RETRY_DURATION + (2 * time.Second))
}

func setupTxm(t *testing.T, grpcClient sdk.GrpcClient) (*trontxm.TronTxm, logger.Logger, *observer.ObservedLogs) {
	testLogger, observedLogs := logger.TestObserved(t, zapcore.DebugLevel)
	txm := trontxm.TronTxm{
		Logger:                testLogger,
		Keystore:              keystore,
		Config:                config,
		EstimateEnergyEnabled: true,

		Client:        grpcClient,
		BroadcastChan: make(chan *trontxm.TronTx, config.BroadcastChanSize),
		AccountStore:  trontxm.NewAccountStore(),
		Stop:          make(chan struct{}),
	}
	txm.Start(context.Background())
	return &txm, testLogger, observedLogs
}

func TestTxm(t *testing.T) {
	// setup
	grpcClient := mocks.NewGrpcClient(t)
	grpcClient.On("Start", mock.Anything).Maybe().Return(nil)
	grpcClient.On("EstimateEnergy", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(&api.EstimateEnergyMessage{
		Result: &api.Return{
			Result: true,
		},
		EnergyRequired: 1000,
	}, nil)
	grpcClient.On("GetEnergyPrices").Maybe().Return(&api.PricesResponseMessage{Prices: "0:420"}, nil)
	grpcClient.On("TriggerContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(&api.TransactionExtention{
		Transaction: &core.Transaction{
			RawData: &core.TransactionRaw{
				Timestamp:    123,
				Expiration:   456,
				RefBlockHash: []byte("abc"),
				FeeLimit:     789,
			},
		},
		Txid:           []byte("txid"),
		ConstantResult: [][]byte{{0x01}},
		Result:         &api.Return{Result: true},
		EnergyUsed:     1000,
	}, nil)
	grpcClient.On("Broadcast", mock.Anything).Maybe().Return(&api.Return{
		Result:  true,
		Code:    api.Return_SUCCESS,
		Message: []byte("broadcast message"),
	}, nil)
	grpcClient.On("GetTransactionInfoByID", mock.Anything).Maybe().Return(&core.TransactionInfo{
		Receipt:     &core.ResourceReceipt{Result: core.Transaction_Result_SUCCESS},
		BlockNumber: 123,
	}, nil)
	keystore = testutils.NewTestKeystore(genesisAddress, genesisPrivateKey)

	t.Run("Invalid input params", func(t *testing.T) {
		txm, _, _ := setupTxm(t, grpcClient)
		err := txm.Enqueue(genesisAddress, genesisAddress, "foo()", "param1")
		require.Error(t, err)
		require.ErrorContains(t, err, "odd number of params")
	})

	t.Run("Success", func(t *testing.T) {
		txm, lggr, observedLogs := setupTxm(t, grpcClient)
		err := txm.Enqueue(genesisAddress, genesisAddress, "foo()")
		require.NoError(t, err)

		WaitForInflightTxs(lggr, txm, 10*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("retry").Len(), 0)
		require.Equal(t, observedLogs.FilterMessageSnippet("confirmed transaction").Len(), 1)
	})

	t.Run("Retry on broadcast server busy", func(t *testing.T) {
		grpcClient.On("Broadcast", mock.Anything).Unset()
		grpcClient.On("Broadcast", mock.Anything).Return(&api.Return{
			Result:  false,
			Code:    api.Return_SERVER_BUSY,
			Message: []byte("server busy"),
		}, fmt.Errorf("some err"))
		txm, _, observedLogs := setupTxm(t, grpcClient)
		err := txm.Enqueue(genesisAddress, genesisAddress, "foo()")
		require.NoError(t, err)

		waitForMaxRetryDuration()

		queueLen, unconfirmedLen := txm.InflightCount()
		require.Equal(t, queueLen, 0)
		require.Equal(t, unconfirmedLen, 0)
		require.Greater(t, observedLogs.FilterMessageSnippet("SERVER_BUSY or BLOCK_UNSOLIDIFIED: retry broadcast after timeout").Len(), int(trontxm.MAX_BROADCAST_RETRY_DURATION/trontxm.BROADCAST_DELAY_DURATION)-1)
		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed to broadcast").Len(), 1)
	})

	t.Run("Retry on broadcast block unsolidified", func(t *testing.T) {
		grpcClient.On("Broadcast", mock.Anything).Unset()
		grpcClient.On("Broadcast", mock.Anything).Return(&api.Return{
			Result:  false,
			Code:    api.Return_BLOCK_UNSOLIDIFIED,
			Message: []byte("block unsolid"),
		}, fmt.Errorf("some err"))
		txm, _, observedLogs := setupTxm(t, grpcClient)
		err := txm.Enqueue(genesisAddress, genesisAddress, "foo()")
		require.NoError(t, err)

		waitForMaxRetryDuration()

		queueLen, unconfirmedLen := txm.InflightCount()
		require.Equal(t, queueLen, 0)
		require.Equal(t, unconfirmedLen, 0)
		require.Greater(t, observedLogs.FilterMessageSnippet("SERVER_BUSY or BLOCK_UNSOLIDIFIED: retry broadcast after timeout").Len(), int(trontxm.MAX_BROADCAST_RETRY_DURATION/trontxm.BROADCAST_DELAY_DURATION)-1)
		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed to broadcast").Len(), 1)
	})

	t.Run("No retry on other broadcast err", func(t *testing.T) {
		grpcClient.On("Broadcast", mock.Anything).Unset()
		grpcClient.On("Broadcast", mock.Anything).Return(&api.Return{
			Result:  false,
			Code:    api.Return_BANDWITH_ERROR,
			Message: []byte("some error"),
		}, fmt.Errorf("some err"))
		txm, lggr, observedLogs := setupTxm(t, grpcClient)
		err := txm.Enqueue(genesisAddress, genesisAddress, "foo()")
		require.NoError(t, err)

		WaitForInflightTxs(lggr, txm, 10*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("SERVER_BUSY or BLOCK_UNSOLIDIFIED: retry broadcast after timeout").Len(), 0)
		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed to broadcast").Len(), 1)
	})
}
