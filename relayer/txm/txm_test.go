package txm_test

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
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
	"github.com/smartcontractkit/chainlink-common/pkg/types"

	"github.com/smartcontractkit/chainlink-tron/relayer/mocks"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-tron/relayer/testutils"
	trontxm "github.com/smartcontractkit/chainlink-tron/relayer/txm"
)

// Test configuration and setup helpers
var (
	defaultConfig = trontxm.TronTxmConfig{
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
		RetentionPeriod:   10 * time.Second,
		ReapInterval:      1 * time.Second,
	}

	genesisAccountKey = testutils.CreateKey(rand.Reader)
	genesisAddress    = genesisAccountKey.Address
	genesisPrivateKey = genesisAccountKey.PrivateKey
)

func createTestKeystore() *testutils.TestKeystore {
	return testutils.NewTestKeystore(genesisAddress.String(), genesisPrivateKey)
}

func createDefaultMockClient(t *testing.T) *mocks.CombinedClient {
	combinedClient := mocks.NewCombinedClient(t)

	combinedClient.On("Start", mock.Anything).Maybe().Return(nil)
	combinedClient.On("EstimateEnergy", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(&soliditynode.EnergyEstimateResult{
		Result:         soliditynode.ReturnEnergyEstimate{Result: true},
		EnergyRequired: 1000,
	}, nil)
	combinedClient.On("GetEnergyPrices").Maybe().Return(&fullnode.EnergyPrices{Prices: "0:420"}, nil)

	txid, _ := hex.DecodeString("2a037789237971c1c1d648f7b90b70c68a9aa6b0a2892f947213286346d0210d")

	combinedClient.On("GetNowBlockFullNode").Maybe().Return(&soliditynode.Block{
		BlockHeader: &soliditynode.BlockHeader{
			RawData: &soliditynode.BlockHeaderRaw{
				Timestamp: 1000,
				Number:    12345,
			},
		},
	}, nil)

	combinedClient.On("TriggerSmartContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(&fullnode.TriggerSmartContractResponse{
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

	combinedClient.On("BroadcastTransaction", mock.Anything).Maybe().Return(&fullnode.BroadcastResponse{
		Result:  true,
		Code:    "SUCCESS",
		Message: "broadcast message",
	}, nil)

	return combinedClient
}

func setupTxm(t *testing.T, combinedClient sdk.CombinedClient, customConfig *trontxm.TronTxmConfig) (*trontxm.TronTxm, logger.Logger, *observer.ObservedLogs) {
	testLogger, observedLogs := logger.TestObserved(t, zapcore.DebugLevel)
	keystore := createTestKeystore()

	config := defaultConfig
	if customConfig != nil {
		config = *customConfig
	}

	txm := &trontxm.TronTxm{
		Logger:                testLogger,
		Keystore:              keystore,
		Config:                config,
		EstimateEnergyEnabled: true,
		Client:                combinedClient,
		BroadcastChan:         make(chan *trontxm.TronTx, config.BroadcastChanSize),
		AccountStore:          trontxm.NewAccountStore(),
		Stop:                  make(chan struct{}),
	}

	require.NoError(t, txm.Start(t.Context()))
	return txm, testLogger, observedLogs
}

func waitForMaxRetryDuration() {
	time.Sleep(trontxm.MAX_BROADCAST_RETRY_DURATION + (2 * time.Second))
}

func TestTxm(t *testing.T) {
	t.Parallel()
	t.Run("Invalid input params", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		txm, _, _ := setupTxm(t, combinedClient, nil)
		defer txm.Close()

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
		combinedClient := createDefaultMockClient(t)
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil).Once()
		combinedClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil).Once()

		txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

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

	t.Run("Reorg success", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		// mark confirmed
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12345,
		}, nil).Once()
		// finalization not found at first
		combinedClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "FAILED"},
			BlockNumber: 12346,
		}, errors.New("block reorg")).Once()
		// reorg - account for retry logic (1 initial + 3 retries = 4 calls per reorg detection)
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "FAILED"},
			BlockNumber: 12346,
		}, errors.New("block reorg")).Times(4)
		// re-confirm w/ lower block height to simulate finalization after reorg
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Once()
		// finalized
		combinedClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Once()

		txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 10*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("tx missing after reorg, moving back to unconfirmed").Len(), 1)
		require.Equal(t, observedLogs.FilterMessageSnippet("finalized transaction").Len(), 1)
	})
}

func TestTxmRetryLogic(t *testing.T) {
	t.Parallel()
	t.Run("Retry on broadcast server busy", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		combinedClient.On("BroadcastTransaction", mock.Anything).Unset()
		combinedClient.On("BroadcastTransaction", mock.Anything).Return(&fullnode.BroadcastResponse{
			Result:  false,
			Code:    "SERVER_BUSY",
			Message: "server busy",
		}, fmt.Errorf("some err"))

		txm, _, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

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
		combinedClient := createDefaultMockClient(t)

		combinedClient.On("BroadcastTransaction", mock.Anything).Unset()
		combinedClient.On("BroadcastTransaction", mock.Anything).Return(&fullnode.BroadcastResponse{
			Result:  false,
			Code:    "BLOCK_UNSOLIDIFIED",
			Message: "block unsolid",
		}, fmt.Errorf("some err"))

		txm, _, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

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

	t.Run("No retry on other broadcast error", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		combinedClient.On("BroadcastTransaction", mock.Anything).Unset()
		combinedClient.On("BroadcastTransaction", mock.Anything).Return(&fullnode.BroadcastResponse{
			Result:  false,
			Code:    "BANDWITH_ERROR",
			Message: "some error",
		}, fmt.Errorf("some err"))

		txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

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

func TestTxmTransactionReaping(t *testing.T) {
	t.Parallel()
	t.Run("Reap expired transactions", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil).Once()
		combinedClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil).Once()

		txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 5*time.Second)
		finishedBefore := txm.AccountStore.GetTotalFinishedCount()

		require.Equal(t, observedLogs.FilterMessageSnippet("finalized transaction").Len(), 1)

		time.Sleep(10 * time.Second)
		finishedAfter := txm.AccountStore.GetTotalFinishedCount()

		require.Greater(t, finishedBefore, finishedAfter)
	})

	t.Run("Reap multiple transactions with different states", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		shortReapConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			RetentionPeriod:   200 * time.Millisecond,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, combinedClient, shortReapConfig)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		finalizedTx := &trontxm.TronTx{ID: "finalized_tx", FromAddress: genesisAddress, CreateTs: time.Now()}
		fatalTx1 := &trontxm.TronTx{ID: "fatal_tx_1", FromAddress: genesisAddress, CreateTs: time.Now()}
		fatalTx2 := &trontxm.TronTx{ID: "fatal_tx_2", FromAddress: genesisAddress, CreateTs: time.Now()}

		require.NoError(t, store.OnPending(finalizedTx))
		require.NoError(t, store.OnBroadcasted("hash1", time.Now().UnixMilli()+1000, finalizedTx))
		require.NoError(t, store.OnConfirmed(finalizedTx.ID))
		require.NoError(t, store.OnFinalized(finalizedTx.ID))

		require.NoError(t, store.OnPending(fatalTx1))
		require.NoError(t, store.OnBroadcasted("hash2", time.Now().UnixMilli()+1000, fatalTx1))
		require.NoError(t, store.OnFatalError(fatalTx1.ID))

		require.NoError(t, store.OnPending(fatalTx2))
		require.NoError(t, store.OnBroadcasted("hash3", time.Now().UnixMilli()+1000, fatalTx2))
		require.NoError(t, store.OnConfirmed(fatalTx2.ID))
		require.NoError(t, store.OnFatalError(fatalTx2.ID))

		require.Equal(t, 3, store.FinishedCount())

		time.Sleep(300 * time.Millisecond)
		time.Sleep(200 * time.Millisecond)

		require.Equal(t, 0, store.FinishedCount())
		require.Equal(t, 0, len(txm.AccountStore.GetHashToIdMap()))
		require.False(t, store.Has(finalizedTx.ID))
		require.False(t, store.Has(fatalTx1.ID))
		require.False(t, store.Has(fatalTx2.ID))
	})

	t.Run("Reap only expired transactions, keep recent ones", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		reapConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			RetentionPeriod:   500 * time.Millisecond,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, combinedClient, reapConfig)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		oldTx := &trontxm.TronTx{
			ID:          "old_tx",
			FromAddress: genesisAddress,
			CreateTs:    time.Now(),
		}

		require.NoError(t, store.OnPending(oldTx))
		require.NoError(t, store.OnBroadcasted("old_hash", time.Now().UnixMilli()+1000, oldTx))
		require.NoError(t, store.OnConfirmed(oldTx.ID))
		require.NoError(t, store.OnFinalized(oldTx.ID))

		newTx := &trontxm.TronTx{
			ID:          "new_tx",
			FromAddress: genesisAddress,
			CreateTs:    time.Now(),
		}

		require.NoError(t, store.OnPending(newTx))
		require.NoError(t, store.OnBroadcasted("new_hash", time.Now().UnixMilli()+1000, newTx))
		require.NoError(t, store.OnConfirmed(newTx.ID))

		require.Equal(t, 2, len(txm.AccountStore.GetHashToIdMap()))
		require.Equal(t, 1, store.FinishedCount())

		time.Sleep(1 * time.Second)
		require.NoError(t, store.OnFinalized(newTx.ID))

		require.False(t, store.Has(oldTx.ID))
		require.True(t, store.Has(newTx.ID))
		require.Equal(t, 1, store.FinishedCount())
		require.Equal(t, 1, len(txm.AccountStore.GetHashToIdMap()))
	})

	t.Run("Reap across multiple accounts", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		multiAccountConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			RetentionPeriod:   150 * time.Millisecond,
			ReapInterval:      30 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, combinedClient, multiAccountConfig)
		defer txm.Close()

		accounts := make([]string, 3)
		for i := 0; i < 3; i++ {
			key := testutils.CreateKey(rand.Reader)
			accounts[i] = key.Address.String()
		}

		for i, account := range accounts {
			store := txm.AccountStore.GetTxStore(account)

			tx := &trontxm.TronTx{
				ID:          fmt.Sprintf("account_%d_tx", i),
				FromAddress: genesisAddress,
				CreateTs:    time.Now(),
			}

			require.NoError(t, store.OnPending(tx))
			require.NoError(t, store.OnBroadcasted(fmt.Sprintf("hash_%d", i), time.Now().UnixMilli()+1000, tx))
			require.NoError(t, store.OnConfirmed(tx.ID))
			require.NoError(t, store.OnFinalized(tx.ID))
		}

		totalFinished := 0
		for _, account := range accounts {
			store := txm.AccountStore.GetTxStore(account)
			totalFinished += store.FinishedCount()
		}
		require.Equal(t, 3, totalFinished)
		require.Equal(t, 3, len(txm.AccountStore.GetHashToIdMap()))

		time.Sleep(200 * time.Millisecond)
		time.Sleep(100 * time.Millisecond)

		totalFinishedAfter := 0
		for _, account := range accounts {
			store := txm.AccountStore.GetTxStore(account)
			totalFinishedAfter += store.FinishedCount()
		}
		require.Equal(t, 0, totalFinishedAfter)
		require.Equal(t, 0, len(txm.AccountStore.GetHashToIdMap()))
	})

	t.Run("No reaping when no expired transactions", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		noReapConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			RetentionPeriod:   5 * time.Second,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, combinedClient, noReapConfig)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		recentTx := &trontxm.TronTx{
			ID:          "recent_tx",
			FromAddress: genesisAddress,
			CreateTs:    time.Now(),
		}

		require.NoError(t, store.OnPending(recentTx))
		require.NoError(t, store.OnBroadcasted("recent_hash", time.Now().UnixMilli()+1000, recentTx))
		require.NoError(t, store.OnConfirmed(recentTx.ID))
		require.NoError(t, store.OnFinalized(recentTx.ID))

		require.Equal(t, 1, store.FinishedCount())
		require.Equal(t, 1, len(txm.AccountStore.GetHashToIdMap()))

		time.Sleep(200 * time.Millisecond)

		require.True(t, store.Has(recentTx.ID))
		require.Equal(t, 1, store.FinishedCount())
		require.Equal(t, 1, len(txm.AccountStore.GetHashToIdMap()))
	})

	t.Run("Reap performance with many transactions", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		perfConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			RetentionPeriod:   100 * time.Millisecond,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, combinedClient, perfConfig)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		numTxs := 100
		for i := 0; i < numTxs; i++ {
			tx := &trontxm.TronTx{
				ID:          fmt.Sprintf("perf_tx_%d", i),
				FromAddress: genesisAddress,
				CreateTs:    time.Now(),
			}

			require.NoError(t, store.OnPending(tx))
			require.NoError(t, store.OnBroadcasted(fmt.Sprintf("perf_hash_%d", i), time.Now().UnixMilli()+1000, tx))
			require.NoError(t, store.OnConfirmed(tx.ID))
			require.NoError(t, store.OnFinalized(tx.ID))
		}

		require.Equal(t, numTxs, store.FinishedCount())
		require.Equal(t, numTxs, len(txm.AccountStore.GetHashToIdMap()))

		startTime := time.Now()
		time.Sleep(200 * time.Millisecond)
		reapDuration := time.Since(startTime)

		require.Equal(t, 0, store.FinishedCount())
		require.Equal(t, 0, len(txm.AccountStore.GetHashToIdMap()))
		require.Less(t, reapDuration, 1*time.Second)
	})
}

func TestTxmStateTransitions(t *testing.T) {
	t.Parallel()

	t.Run("TxStore state transitions", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		txm, _, _ := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		// OnPending
		tx1 := &trontxm.TronTx{ID: "id1", FromAddress: genesisAddress}
		hash1 := "hash1"
		require.NoError(t, store.OnPending(tx1))
		require.True(t, store.Has("id1"))
		require.Equal(t, trontxm.Pending, tx1.State)
		status, err := txm.GetTransactionStatus(t.Context(), tx1.ID)
		require.NoError(t, err)
		require.Equal(t, types.Pending, status)

		// duplicate hash → error
		require.Error(t, store.OnPending(&trontxm.TronTx{ID: "id1"}))

		// OnBroadcasted
		require.NoError(t, store.OnBroadcasted(hash1, 1000, tx1))
		require.Equal(t, trontxm.Broadcasted, tx1.State)
		err = store.OnBroadcasted(hash1, 1000, tx1)
		require.ErrorContains(t, err, "hash already exists")
		require.Error(t, store.OnBroadcasted("no-such", 1000, &trontxm.TronTx{ID: "no-such"}))

		// OnConfirmed
		require.NoError(t, store.OnConfirmed(tx1.ID))
		require.Equal(t, trontxm.Confirmed, tx1.State)
		// can't confirm again bc not in broadcasted state / unconfirmed txs map
		require.Error(t, store.OnConfirmed(tx1.ID))
		require.Len(t, store.GetUnconfirmed(), 0)
		require.Error(t, store.OnConfirmed("no-such"))

		// Fatal error from confirmed
		require.NoError(t, store.OnFatalError(tx1.ID))
		require.Equal(t, trontxm.FatallyErrored, tx1.State)
		// retain id
		require.True(t, store.Has(tx1.ID))

		tx2 := &trontxm.TronTx{ID: "id2", FromAddress: genesisAddress}
		// OnErrored
		require.Error(t, store.OnErrored("no-such"))
		require.NoError(t, store.OnPending(tx2))
		require.NoError(t, store.OnBroadcasted("h3", 2000, tx2))
		require.NoError(t, store.OnErrored(tx2.ID))
		require.Equal(t, trontxm.Errored, tx2.State)
		require.True(t, store.Has(tx2.ID))

		// OnReorg
		require.NoError(t, store.OnBroadcasted("h3", 2000, tx2))
		require.NoError(t, store.OnConfirmed(tx2.ID))
		require.NoError(t, store.OnReorg(tx2.ID))
		require.Equal(t, trontxm.Broadcasted, tx2.State)

		// OnFinalized
		require.NoError(t, store.OnConfirmed(tx2.ID))
		require.NoError(t, store.OnFinalized(tx2.ID))
		require.Equal(t, trontxm.Finalized, tx2.State)
		require.Len(t, store.GetUnconfirmed(), 0)

		// fatal tx + finalized tx
		require.Equal(t, store.FinishedCount(), 2)

		// ensure finalized can't be changed
		require.Error(t, store.OnPending(tx2))
		require.Error(t, store.OnBroadcasted("h3", 2000, tx2))
		require.Error(t, store.OnConfirmed(tx2.ID))
		require.Error(t, store.OnFinalized(tx2.ID))
		require.Error(t, store.OnFatalError(tx2.ID))

		// ensure fatal can't be changed
		require.Error(t, store.OnPending(tx1))
		require.Error(t, store.OnBroadcasted("h2", 2000, tx1))
		require.Error(t, store.OnConfirmed(tx1.ID))
		require.Error(t, store.OnFinalized(tx1.ID))
		require.Error(t, store.OnFatalError(tx1.ID))
	})
}

func TestTxmRaceConditions(t *testing.T) {
	t.Parallel()

	t.Run("Concurrent enqueue operations", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)
		combinedClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)

		raceConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 300,
			ConfirmPollSecs:   1,
			RetentionPeriod:   10 * time.Second,
			ReapInterval:      1 * time.Second,
		}

		txm, testLogger, _ := setupTxm(t, combinedClient, raceConfig)
		defer func() {
			done := make(chan struct{})
			go func() {
				txm.Close()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(10 * time.Second):
				t.Log("Warning: TXM close timed out after 10 seconds")
			}
		}()

		numGoroutines := 10
		numTxsPerGoroutine := 5
		var wg sync.WaitGroup
		var successCount int32
		var errorCount int32

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()
				time.Sleep(time.Duration(routineID) * 10 * time.Millisecond)

				for j := 0; j < numTxsPerGoroutine; j++ {
					err := txm.Enqueue(trontxm.TronTxmRequest{
						FromAddress:     genesisAddress,
						ContractAddress: genesisAddress,
						Method:          "foo()",
						Params:          []any{},
						ID:              fmt.Sprintf("tx_%d_%d", routineID, j),
					})

					if err == nil {
						atomic.AddInt32(&successCount, 1)
					} else {
						atomic.AddInt32(&errorCount, 1)
					}
					time.Sleep(1 * time.Millisecond)
				}
			}(i)
		}

		wg.Wait()

		totalExpected := int32(numGoroutines * numTxsPerGoroutine)
		actualSuccess := atomic.LoadInt32(&successCount)
		actualErrors := atomic.LoadInt32(&errorCount)

		require.Greater(t, actualSuccess, totalExpected*7/10, "Too many enqueue failures")
		require.Equal(t, totalExpected, actualSuccess+actualErrors, "Total count mismatch")

		testutils.WaitForInflightTxs(testLogger, txm, 30*time.Second)
	})

	t.Run("Concurrent state transitions", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		txm, _, _ := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		numGoroutines := 100
		var wg sync.WaitGroup

		tx := &trontxm.TronTx{ID: "race_test_tx", FromAddress: genesisAddress}
		hash := "race_test_hash"
		require.NoError(t, store.OnPending(tx))

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()
				switch routineID % 4 {
				case 0:
					store.OnBroadcasted(hash, 2000, tx)
				case 1:
					store.OnConfirmed(tx.ID)
				case 2:
					store.OnReorg(tx.ID)
				case 3:
					store.GetStatus(tx.ID)
				}
			}(i)
		}

		wg.Wait()

		state, exists := store.GetStatus(tx.ID)
		require.True(t, exists)
		require.NotEqual(t, trontxm.TxState(0), state)
	})

	t.Run("Concurrent account access", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		txm, _, _ := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		numGoroutines := 50
		numAccounts := 10
		var wg sync.WaitGroup

		accounts := make([]string, numAccounts)
		for i := 0; i < numAccounts; i++ {
			key := testutils.CreateKey(rand.Reader)
			accounts[i] = key.Address.String()
		}

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()

				accountIdx := routineID % numAccounts
				store := txm.AccountStore.GetTxStore(accounts[accountIdx])

				tx := &trontxm.TronTx{
					ID:          fmt.Sprintf("tx_%d", routineID),
					FromAddress: genesisAddress,
				}
				hash := fmt.Sprintf("hash_%d", routineID)

				store.OnPending(tx)
				store.OnBroadcasted(hash, time.Now().UnixMilli()+10000, tx)
				store.GetUnconfirmed()
				store.Has(tx.ID)
			}(i)
		}

		wg.Wait()

		totalTxs := 0
		for _, account := range accounts {
			store := txm.AccountStore.GetTxStore(account)
			totalTxs += len(store.GetUnconfirmed())
		}
		require.Greater(t, totalTxs, 0)
	})

	t.Run("Concurrent transaction status checks", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)
		combinedClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)
		txm, _, _ := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		txID := "concurrent_status_test"
		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
			ID:              txID,
		})
		require.NoError(t, err)

		time.Sleep(100 * time.Millisecond)

		initialStatus, err := txm.GetTransactionStatus(t.Context(), txID)
		require.NoError(t, err)
		require.NotEqual(t, types.Unknown, initialStatus, "Transaction should be trackable before concurrent testing")

		numGoroutines := 50
		var wg sync.WaitGroup
		statusResults := make([]types.TransactionStatus, numGoroutines)
		var errorCount int32

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				status, err := txm.GetTransactionStatus(t.Context(), txID)
				if err == nil {
					statusResults[idx] = status
				} else {
					atomic.AddInt32(&errorCount, 1)
				}
			}(i)
		}

		wg.Wait()

		successCount := 0
		for _, status := range statusResults {
			if status != types.Unknown {
				successCount++
			}
		}

		require.Greater(t, successCount, numGoroutines*7/10, "Most concurrent status checks should succeed")
	})
}

func TestTxmTransactionFailureScenarios(t *testing.T) {
	t.Parallel()

	t.Run("OUT_OF_ENERGY failure with energy bump retry", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "OUT_OF_ENERGY"},
			BlockNumber: 12345,
		}, nil).Once()

		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Once()
		combinedClient.On("GetTransactionInfoById", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Once()

		txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
			ID:              "energy_test",
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 10*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed due to out of energy").Len(), 1)
		require.Equal(t, observedLogs.FilterMessageSnippet("retrying transaction").Len(), 1)
		require.Equal(t, observedLogs.FilterMessageSnippet("finalized transaction").Len(), 1)

		status, err := txm.GetTransactionStatus(t.Context(), "energy_test")
		require.NoError(t, err)
		require.Equal(t, types.Finalized, status)
	})

	t.Run("OUT_OF_TIME failure with retry limit", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "OUT_OF_TIME"},
			BlockNumber: 12345,
		}, nil)

		txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
			ID:              "timeout_test",
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 10*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed due to out of time").Len(), 3)
		require.Equal(t, observedLogs.FilterMessageSnippet("not retrying, multiple OUT_OF_TIME errors").Len(), 1)

		status, err := txm.GetTransactionStatus(t.Context(), "timeout_test")
		require.NoError(t, err)
		require.Equal(t, types.Fatal, status)
	})

	t.Run("REVERT failure marked as fatal immediately", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "REVERT"},
			BlockNumber: 12345,
		}, nil).Once()

		txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
			ID:              "revert_test",
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 10*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed with fatal error").Len(), 1)
		require.Equal(t, observedLogs.FilterMessageSnippet("retrying transaction").Len(), 0)

		status, err := txm.GetTransactionStatus(t.Context(), "revert_test")
		require.NoError(t, err)
		require.Equal(t, types.Fatal, status)
	})

	t.Run("UNKNOWN failure with retry", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "UNKNOWN"},
			BlockNumber: 12345,
		}, nil).Once()

		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Once()
		combinedClient.On("GetTransactionInfoById", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Once()

		txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
			ID:              "unknown_test",
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 10*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed due to unknown error").Len(), 1)
		require.Equal(t, observedLogs.FilterMessageSnippet("retrying transaction").Len(), 1)
		require.Equal(t, observedLogs.FilterMessageSnippet("finalized transaction").Len(), 1)

		status, err := txm.GetTransactionStatus(t.Context(), "unknown_test")
		require.NoError(t, err)
		require.Equal(t, types.Finalized, status)
	})

	t.Run("Multiple fatal error types", func(t *testing.T) {
		fatalResults := []string{
			"BAD_JUMP_DESTINATION",
			"OUT_OF_MEMORY",
			"STACK_TOO_SMALL",
			"STACK_TOO_LARGE",
			"ILLEGAL_OPERATION",
			"STACK_OVERFLOW",
			"JVM_STACK_OVER_FLOW",
			"TRANSFER_FAILED",
			"INVALID_CODE",
		}

		for _, result := range fatalResults {
			t.Run("fatal_"+result, func(t *testing.T) {
				combinedClient := createDefaultMockClient(t)

				combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
					Receipt:     soliditynode.ResourceReceipt{Result: result},
					BlockNumber: 12345,
				}, nil).Once()

				txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
				defer txm.Close()

				testID := "fatal_" + result + "_test"
				err := txm.Enqueue(trontxm.TronTxmRequest{
					FromAddress:     genesisAddress,
					ContractAddress: genesisAddress,
					Method:          "foo()",
					Params:          []any{},
					ID:              testID,
				})
				require.NoError(t, err)

				testutils.WaitForInflightTxs(lggr, txm, 10*time.Second)

				require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed with fatal error").Len(), 1)
				require.Equal(t, observedLogs.FilterMessageSnippet("retrying transaction").Len(), 0)

				status, err := txm.GetTransactionStatus(t.Context(), testID)
				require.NoError(t, err)
				require.Equal(t, types.Fatal, status)
			})
		}
	})

	t.Run("Max retry attempts reached", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		maxRetryConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   2,
			RetentionPeriod:   60 * time.Second,
			ReapInterval:      1 * time.Second,
		}

		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "OUT_OF_ENERGY"},
			BlockNumber: 12345,
		}, nil)

		txm, lggr, observedLogs := setupTxm(t, combinedClient, maxRetryConfig)
		defer txm.Close()

		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
			ID:              "max_retry_test",
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 15*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed due to out of energy").Len(), 5)
		require.Equal(t, observedLogs.FilterMessageSnippet("not retrying, already reached max retries").Len(), 1)

		status, err := txm.GetTransactionStatus(t.Context(), "max_retry_test")
		require.NoError(t, err)
		require.Equal(t, types.Fatal, status)
	})

	t.Run("Energy bump progression", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)

		combinedClient.On("EstimateEnergy", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&soliditynode.EnergyEstimateResult{
			Result:         soliditynode.ReturnEnergyEstimate{Result: true},
			EnergyRequired: 1000,
		}, nil)

		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "OUT_OF_ENERGY"},
			BlockNumber: 12345,
		}, nil).Times(3)

		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Once()
		combinedClient.On("GetTransactionInfoById", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Once()

		txm, lggr, observedLogs := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		err := txm.Enqueue(trontxm.TronTxmRequest{
			FromAddress:     genesisAddress,
			ContractAddress: genesisAddress,
			Method:          "foo()",
			Params:          []any{},
			ID:              "energy_bump_test",
		})
		require.NoError(t, err)

		testutils.WaitForInflightTxs(lggr, txm, 15*time.Second)

		require.Equal(t, observedLogs.FilterMessageSnippet("transaction failed due to out of energy").Len(), 3)
		require.Equal(t, observedLogs.FilterMessageSnippet("retrying transaction").Len(), 3)
		require.Equal(t, observedLogs.FilterMessageSnippet("finalized transaction").Len(), 1)

		status, err := txm.GetTransactionStatus(t.Context(), "energy_bump_test")
		require.NoError(t, err)
		require.Equal(t, types.Finalized, status)
	})
}

func TestTxmLoadTesting(t *testing.T) {
	t.Parallel()
	t.Run("High volume transaction enqueueing", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil)
		combinedClient.On("GetTransactionInfoById", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil)

		loadConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 300,
			ConfirmPollSecs:   1,
			RetentionPeriod:   5 * time.Second,
			ReapInterval:      500 * time.Millisecond,
		}

		txm, testLogger, _ := setupTxm(t, combinedClient, loadConfig)
		defer txm.Close()

		numTransactions := 200
		startTime := time.Now()
		successCount := 0

		for i := 0; i < numTransactions; i++ {
			err := txm.Enqueue(trontxm.TronTxmRequest{
				FromAddress:     genesisAddress,
				ContractAddress: genesisAddress,
				Method:          "foo()",
				Params:          []any{},
				ID:              fmt.Sprintf("load_test_%d", i),
			})
			if err == nil {
				successCount++
			}

			if i > 0 && i%50 == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		}

		require.Greater(t, successCount, numTransactions*8/10, "Too many enqueue failures")

		testutils.WaitForInflightTxs(testLogger, txm, 60*time.Second)

		processingTime := time.Since(startTime)
		if processingTime.Seconds() > 0 {
			throughput := float64(successCount) / processingTime.Seconds()
			require.Greater(t, throughput, 5.0, "Throughput too low: %.2f tx/sec", throughput)
		}
	})

	t.Run("Channel capacity stress test", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil)
		combinedClient.On("GetTransactionInfoById", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil)

		smallConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 10,
			ConfirmPollSecs:   1,
			RetentionPeriod:   5 * time.Second,
			ReapInterval:      500 * time.Millisecond,
		}

		txm, testLogger, _ := setupTxm(t, combinedClient, smallConfig)
		defer txm.Close()

		numGoroutines := 20
		numTxsPerGoroutine := 5
		var wg sync.WaitGroup
		var enqueueErrors int32

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()
				for j := 0; j < numTxsPerGoroutine; j++ {
					err := txm.Enqueue(trontxm.TronTxmRequest{
						FromAddress:     genesisAddress,
						ContractAddress: genesisAddress,
						Method:          "foo()",
						Params:          []any{},
						ID:              fmt.Sprintf("stress_tx_%d_%d", routineID, j),
					})
					if err != nil {
						atomic.AddInt32(&enqueueErrors, 1)
					}
				}
			}(i)
		}

		wg.Wait()
		testutils.WaitForInflightTxs(testLogger, txm, 30*time.Second)
	})

	t.Run("Multiple accounts high volume", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil)
		combinedClient.On("GetTransactionInfoById", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil)
		txm, testLogger, _ := setupTxm(t, combinedClient, nil)
		defer txm.Close()

		numAccounts := 5
		numTxsPerAccount := 10

		var wg sync.WaitGroup
		var totalSuccessful int32

		for i := 0; i < numAccounts; i++ {
			wg.Add(1)
			go func(accountIdx int) {
				defer wg.Done()

				successful := 0
				for j := 0; j < numTxsPerAccount; j++ {
					err := txm.Enqueue(trontxm.TronTxmRequest{
						FromAddress:     genesisAddress,
						ContractAddress: genesisAddress,
						Method:          "foo()",
						Params:          []any{},
						ID:              fmt.Sprintf("multi_account_%d_%d", accountIdx, j),
					})
					if err == nil {
						successful++
					}

					if j%5 == 0 {
						time.Sleep(time.Millisecond)
					}
				}

				atomic.AddInt32(&totalSuccessful, int32(successful))
			}(i)
		}

		wg.Wait()

		expectedTotal := int32(numAccounts * numTxsPerAccount)
		require.Greater(t, totalSuccessful, expectedTotal*7/10, "Too many enqueue failures")

		testutils.WaitForInflightTxs(testLogger, txm, 60*time.Second)
	})

	t.Run("Concurrent reaping and finalization", func(t *testing.T) {
		combinedClient := createDefaultMockClient(t)
		combinedClient.On("GetTransactionInfoByIdFullNode", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil)
		combinedClient.On("GetTransactionInfoById", mock.Anything).Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil)

		shortConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			RetentionPeriod:   100 * time.Millisecond,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, combinedClient, shortConfig)
		defer txm.Close()

		numTransactions := 50

		for i := 0; i < numTransactions; i++ {
			err := txm.Enqueue(trontxm.TronTxmRequest{
				FromAddress:     genesisAddress,
				ContractAddress: genesisAddress,
				Method:          "foo()",
				Params:          []any{},
				ID:              fmt.Sprintf("reap_test_%d", i),
			})
			require.NoError(t, err)
		}

		time.Sleep(2 * time.Second)

		queueLen, unconfirmedLen := txm.InflightCount()
		t.Logf("After reaping test - Queue: %d, Unconfirmed: %d", queueLen, unconfirmedLen)
	})
}
