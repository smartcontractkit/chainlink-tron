package txm_test

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"

	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/fullnode"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-tron/relayer/mocks"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-tron/relayer/testutils"
	trontxm "github.com/smartcontractkit/chainlink-tron/relayer/txm"
)

var keystore *testutils.TestKeystore
var config = trontxm.TronTxmConfig{
	BroadcastChanSize: 100,
	ConfirmPollSecs:   2,
}
var genesisAccountKey = testutils.CreateKey(rand.Reader)
var genesisAddress = genesisAccountKey.Address
var genesisPrivateKey = genesisAccountKey.PrivateKey

func waitForMaxRetryDuration() {
	time.Sleep(trontxm.MAX_BROADCAST_RETRY_DURATION + (2 * time.Second))
}

func setupTxm(t *testing.T, fullNodeClient sdk.FullNodeClient) (*trontxm.TronTxm, logger.Logger, *observer.ObservedLogs) {
	testLogger, observedLogs := logger.TestObserved(t, zapcore.DebugLevel)
	txm := trontxm.TronTxm{
		Logger:                testLogger,
		Keystore:              keystore,
		Config:                config,
		EstimateEnergyEnabled: true,

		Client:        fullNodeClient,
		BroadcastChan: make(chan *trontxm.TronTx, config.BroadcastChanSize),
		AccountStore:  trontxm.NewAccountStore(),
		Stop:          make(chan struct{}),
	}
	txm.Start(context.Background())
	return &txm, testLogger, observedLogs
}

func TestTxm(t *testing.T) {
	// setup
	fullNodeClient := mocks.NewFullNodeClient(t)
	fullNodeClient.On("Start", mock.Anything).Maybe().Return(nil)
	fullNodeClient.On("EstimateEnergy", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(&soliditynode.EnergyEstimateResult{
		Result:         soliditynode.ReturnEnergyEstimate{Result: true},
		EnergyRequired: 1000,
	}, nil)
	fullNodeClient.On("GetEnergyPrices").Maybe().Return(&fullnode.EnergyPrices{Prices: "0:420"}, nil)
	txid, err := hex.DecodeString("2a037789237971c1c1d648f7b90b70c68a9aa6b0a2892f947213286346d0210d")
	require.NoError(t, err)

	fullNodeClient.On("GetNowBlock").Maybe().Return(&soliditynode.Block{
		BlockHeader: &soliditynode.BlockHeader{
			RawData: &soliditynode.BlockHeaderRaw{
				Timestamp: 1000,
			},
		},
	}, nil)

	fullNodeClient.On("TriggerSmartContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(&fullnode.TriggerSmartContractResponse{
		Transaction: &common.Transaction{
			TxID: hex.EncodeToString(txid),
			RawData: common.RawData{
				Timestamp:    123,
				Expiration:   2000,
				RefBlockHash: "abc",
				FeeLimit:     789,
			},
		},
		Result: fullnode.TriggerResult{Result: true},
	}, nil)

	fullNodeClient.On("BroadcastTransaction", mock.Anything).Maybe().Return(&fullnode.BroadcastResponse{
		Result:  true,
		Code:    "SUCCESS",
		Message: "broadcast message",
	}, nil)

	fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
		Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
		BlockNumber: 12345,
	}, nil)
	keystore = testutils.NewTestKeystore(genesisAddress.String(), genesisPrivateKey)

	t.Run("Invalid input params", func(t *testing.T) {
		txm, _, _ := setupTxm(t, fullNodeClient)
		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{"param1"},
		})
		require.Error(t, err)
		require.ErrorContains(t, err, "odd number of params")
	})

	t.Run("Success", func(t *testing.T) {
		txm, lggr, observedLogs := setupTxm(t, fullNodeClient)
		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 10*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("retry").Len(), 0)
		require.Equal(t, observedLogs.FilterMessageSnippet("confirmed transaction").Len(), 1)
	})

	t.Run("Retry on broadcast server busy", func(t *testing.T) {
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Unset()
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Return(&fullnode.BroadcastResponse{
			Result:  false,
			Code:    "SERVER_BUSY",
			Message: "server busy",
		}, fmt.Errorf("some err"))

		txm, _, observedLogs := setupTxm(t, fullNodeClient)
		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
		})
		require.NoError(t, err)

		waitForMaxRetryDuration()

		queueLen, unconfirmedLen := txm.InflightCount()
		require.Equal(t, queueLen, 0)
		require.Equal(t, unconfirmedLen, 0)
		require.Greater(t, observedLogs.FilterMessageSnippet("SERVER_BUSY or BLOCK_UNSOLIDIFIED: retry broadcast after timeout").Len(), int(trontxm.MAX_BROADCAST_RETRY_DURATION/trontxm.BROADCAST_DELAY_DURATION)-1)
		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed to broadcast").Len(), 1)
	})

	t.Run("Retry on broadcast block unsolidified", func(t *testing.T) {
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Unset()
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Return(&fullnode.BroadcastResponse{
			Result:  false,
			Code:    "BLOCK_UNSOLIDIFIED",
			Message: "block unsolid",
		}, fmt.Errorf("some err"))
		txm, _, observedLogs := setupTxm(t, fullNodeClient)
		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
		})
		require.NoError(t, err)

		waitForMaxRetryDuration()

		queueLen, unconfirmedLen := txm.InflightCount()
		require.Equal(t, queueLen, 0)
		require.Equal(t, unconfirmedLen, 0)
		require.Greater(t, observedLogs.FilterMessageSnippet("SERVER_BUSY or BLOCK_UNSOLIDIFIED: retry broadcast after timeout").Len(), int(trontxm.MAX_BROADCAST_RETRY_DURATION/trontxm.BROADCAST_DELAY_DURATION)-1)
		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed to broadcast").Len(), 1)
	})

	t.Run("No retry on other broadcast err", func(t *testing.T) {
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Unset()
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Return(&fullnode.BroadcastResponse{
			Result:  false,
			Code:    "BANDWITH_ERROR",
			Message: "some error",
		}, fmt.Errorf("some err"))
		txm, lggr, observedLogs := setupTxm(t, fullNodeClient)
		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 10*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("SERVER_BUSY or BLOCK_UNSOLIDIFIED: retry broadcast after timeout").Len(), 0)
		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed to broadcast").Len(), 1)
	})
}
