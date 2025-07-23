package txm_test

import (
	"context"
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
		FinalityDepth:     10,
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

func createDefaultMockClient(t *testing.T) *mocks.FullNodeClient {
	fullNodeClient := mocks.NewFullNodeClient(t)

	fullNodeClient.On("Start", mock.Anything).Maybe().Return(nil)
	fullNodeClient.On("EstimateEnergy", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Maybe().Return(&soliditynode.EnergyEstimateResult{
		Result:         soliditynode.ReturnEnergyEstimate{Result: true},
		EnergyRequired: 1000,
	}, nil)
	fullNodeClient.On("GetEnergyPrices").Maybe().Return(&fullnode.EnergyPrices{Prices: "0:420"}, nil)

	txid, _ := hex.DecodeString("2a037789237971c1c1d648f7b90b70c68a9aa6b0a2892f947213286346d0210d")

	fullNodeClient.On("GetNowBlock").Maybe().Return(&soliditynode.Block{
		BlockHeader: &soliditynode.BlockHeader{
			RawData: &soliditynode.BlockHeaderRaw{
				Timestamp: 1000,
				Number:    12345,
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

	return fullNodeClient
}

func setupTxm(t *testing.T, fullNodeClient sdk.FullNodeClient, customConfig *trontxm.TronTxmConfig) (*trontxm.TronTxm, logger.Logger, *observer.ObservedLogs) {
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
		Client:                fullNodeClient,
		BroadcastChan:         make(chan *trontxm.TronTx, config.BroadcastChanSize),
		AccountStore:          trontxm.NewAccountStore(),
		Stop:                  make(chan struct{}),
	}

	require.NoError(t, txm.Start(context.Background()))
	return txm, testLogger, observedLogs
}

func waitForMaxRetryDuration() {
	time.Sleep(trontxm.MAX_BROADCAST_RETRY_DURATION + (2 * time.Second))
}

func TestTxm(t *testing.T) {
	t.Parallel()
	t.Run("Invalid input params", func(t *testing.T) {
		fullNodeClient := createDefaultMockClient(t)
		txm, _, _ := setupTxm(t, fullNodeClient, nil)
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
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil).Times(2)

		txm, lggr, observedLogs := setupTxm(t, fullNodeClient, nil)
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
		fullNodeClient := createDefaultMockClient(t)

		// mark confirmed
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12345,
		}, nil).Once()
		// reorg
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "FAILED"},
			BlockNumber: 12346,
		}, errors.New("block reorg")).Once()
		// re-confirm w/ lower block height to simulate finalization after reorg
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Times(2)

		txm, lggr, observedLogs := setupTxm(t, fullNodeClient, nil)
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
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil).Times(2)

		fullNodeClient.On("BroadcastTransaction", mock.Anything).Unset()
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Return(&fullnode.BroadcastResponse{
			Result:  false,
			Code:    "SERVER_BUSY",
			Message: "server busy",
		}, fmt.Errorf("some err"))

		txm, _, observedLogs := setupTxm(t, fullNodeClient, nil)
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
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil).Times(2)

		fullNodeClient.On("BroadcastTransaction", mock.Anything).Unset()
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Return(&fullnode.BroadcastResponse{
			Result:  false,
			Code:    "BLOCK_UNSOLIDIFIED",
			Message: "block unsolid",
		}, fmt.Errorf("some err"))

		txm, _, observedLogs := setupTxm(t, fullNodeClient, nil)
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
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Unset()
		fullNodeClient.On("BroadcastTransaction", mock.Anything).Return(&fullnode.BroadcastResponse{
			Result:  false,
			Code:    "BANDWITH_ERROR",
			Message: "some error",
		}, fmt.Errorf("some err"))

		txm, lggr, observedLogs := setupTxm(t, fullNodeClient, nil)
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
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 12300,
		}, nil).Times(2)

		txm, lggr, observedLogs := setupTxm(t, fullNodeClient, nil)
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

		testutils.WaitForInflightTxs(lggr, txm, 10*time.Second)
		finishedAfter := txm.AccountStore.GetTotalFinishedCount()

		require.Greater(t, finishedBefore, finishedAfter)
	})

		t.Run("Reap multiple transactions with different states", func(t *testing.T) {
		fullNodeClient := createDefaultMockClient(t)
		
		shortReapConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			FinalityDepth:     1,
			RetentionPeriod:   200 * time.Millisecond,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, fullNodeClient, shortReapConfig)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		finalizedTx := &trontxm.TronTx{ID: "finalized_tx", FromAddress: genesisAddress, CreateTs: time.Now()}
		fatalTx1 := &trontxm.TronTx{ID: "fatal_tx_1", FromAddress: genesisAddress, CreateTs: time.Now()}
		fatalTx2 := &trontxm.TronTx{ID: "fatal_tx_2", FromAddress: genesisAddress, CreateTs: time.Now()}

		require.NoError(t, store.OnPending("hash1", time.Now().UnixMilli()+1000, finalizedTx))
		require.NoError(t, store.OnConfirmed(finalizedTx.ID))
		require.NoError(t, store.OnFinalized(finalizedTx.ID))

		require.NoError(t, store.OnPending("hash2", time.Now().UnixMilli()+1000, fatalTx1))
		require.NoError(t, store.OnFatalError(fatalTx1.ID))

		require.NoError(t, store.OnPending("hash3", time.Now().UnixMilli()+1000, fatalTx2))
		require.NoError(t, store.OnConfirmed(fatalTx2.ID))
		require.NoError(t, store.OnFatalError(fatalTx2.ID))

		require.Equal(t, 3, store.FinishedCount())

		time.Sleep(300 * time.Millisecond)
		time.Sleep(200 * time.Millisecond)

		require.Equal(t, 0, store.FinishedCount())
		require.False(t, store.Has(finalizedTx.ID))
		require.False(t, store.Has(fatalTx1.ID))
		require.False(t, store.Has(fatalTx2.ID))
	})

		t.Run("Reap only expired transactions, keep recent ones", func(t *testing.T) {
		fullNodeClient := createDefaultMockClient(t)
		
		reapConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			FinalityDepth:     1,
			RetentionPeriod:   500 * time.Millisecond,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, fullNodeClient, reapConfig)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		oldTime := time.Now().Add(-1 * time.Second)
		oldTx := &trontxm.TronTx{
			ID:          "old_tx",
			FromAddress: genesisAddress,
			CreateTs:    oldTime,
		}

		require.NoError(t, store.OnPending("old_hash", time.Now().UnixMilli()+1000, oldTx))
		require.NoError(t, store.OnConfirmed(oldTx.ID))
		require.NoError(t, store.OnFinalized(oldTx.ID))

		newTx := &trontxm.TronTx{
			ID:          "new_tx",
			FromAddress: genesisAddress,
			CreateTs:    time.Now(),
		}

		require.NoError(t, store.OnPending("new_hash", time.Now().UnixMilli()+1000, newTx))
		require.NoError(t, store.OnConfirmed(newTx.ID))
		require.NoError(t, store.OnFinalized(newTx.ID))

		require.Equal(t, 2, store.FinishedCount())

		time.Sleep(150 * time.Millisecond)

		require.False(t, store.Has(oldTx.ID))
		require.True(t, store.Has(newTx.ID))
		require.Equal(t, 1, store.FinishedCount())
	})

		t.Run("Reap across multiple accounts", func(t *testing.T) {
		fullNodeClient := createDefaultMockClient(t)
		
		multiAccountConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			FinalityDepth:     1,
			RetentionPeriod:   150 * time.Millisecond,
			ReapInterval:      30 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, fullNodeClient, multiAccountConfig)
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

			require.NoError(t, store.OnPending(fmt.Sprintf("hash_%d", i), time.Now().UnixMilli()+1000, tx))
			require.NoError(t, store.OnConfirmed(tx.ID))
			require.NoError(t, store.OnFinalized(tx.ID))
		}

		totalFinished := 0
		for _, account := range accounts {
			store := txm.AccountStore.GetTxStore(account)
			totalFinished += store.FinishedCount()
		}
		require.Equal(t, 3, totalFinished)

		time.Sleep(200 * time.Millisecond)
		time.Sleep(100 * time.Millisecond)

		totalFinishedAfter := 0
		for _, account := range accounts {
			store := txm.AccountStore.GetTxStore(account)
			totalFinishedAfter += store.FinishedCount()
		}
		require.Equal(t, 0, totalFinishedAfter)
	})

		t.Run("No reaping when no expired transactions", func(t *testing.T) {
		fullNodeClient := createDefaultMockClient(t)
		
		noReapConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			FinalityDepth:     1,
			RetentionPeriod:   5 * time.Second,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, fullNodeClient, noReapConfig)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		recentTx := &trontxm.TronTx{
			ID:          "recent_tx",
			FromAddress: genesisAddress,
			CreateTs:    time.Now(),
		}

		require.NoError(t, store.OnPending("recent_hash", time.Now().UnixMilli()+1000, recentTx))
		require.NoError(t, store.OnConfirmed(recentTx.ID))
		require.NoError(t, store.OnFinalized(recentTx.ID))

		require.Equal(t, 1, store.FinishedCount())

		time.Sleep(200 * time.Millisecond)

		require.True(t, store.Has(recentTx.ID))
		require.Equal(t, 1, store.FinishedCount())
	})

		t.Run("Reap performance with many transactions", func(t *testing.T) {
		fullNodeClient := createDefaultMockClient(t)
		
		perfConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			FinalityDepth:     1,
			RetentionPeriod:   100 * time.Millisecond,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, fullNodeClient, perfConfig)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		numTxs := 100
		for i := 0; i < numTxs; i++ {
			tx := &trontxm.TronTx{
				ID:          fmt.Sprintf("perf_tx_%d", i),
				FromAddress: genesisAddress,
				CreateTs:    time.Now(),
			}

			require.NoError(t, store.OnPending(fmt.Sprintf("perf_hash_%d", i), time.Now().UnixMilli()+1000, tx))
			require.NoError(t, store.OnConfirmed(tx.ID))
			require.NoError(t, store.OnFinalized(tx.ID))
		}

		require.Equal(t, numTxs, store.FinishedCount())

		startTime := time.Now()
		time.Sleep(200 * time.Millisecond)
		reapDuration := time.Since(startTime)

		require.Equal(t, 0, store.FinishedCount())
		require.Less(t, reapDuration, 1*time.Second)
	})
}

func TestTxmStateTransitions(t *testing.T) {
	t.Parallel()

	t.Run("TxStore state transitions", func(t *testing.T) {
		fullNodeClient := createDefaultMockClient(t)
		txm, _, _ := setupTxm(t, fullNodeClient, nil)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		// OnPending
		tx1 := &trontxm.TronTx{ID: "id1", FromAddress: genesisAddress}
		hash1 := "hash1"
		require.NoError(t, store.OnPending(hash1, 1000, tx1))
		require.True(t, store.Has("id1"))
		require.Equal(t, trontxm.Pending, tx1.State)
		status, err := txm.GetTransactionStatus(context.Background(), tx1.ID)
		require.NoError(t, err)
		require.Equal(t, types.Pending, status)

		// duplicate hash → error
		require.Error(t, store.OnPending(hash1, 1000, &trontxm.TronTx{ID: "id1"}))

		// OnBroadcasted
		require.NoError(t, store.OnBroadcasted(tx1.ID))
		require.Equal(t, trontxm.Broadcasted, tx1.State)
		require.Error(t, store.OnBroadcasted("no-such"))

		// OnConfirmed
		require.NoError(t, store.OnConfirmed(tx1.ID))
		require.Equal(t, trontxm.Confirmed, tx1.State)
		require.Len(t, store.GetUnconfirmed(), 0)
		require.Error(t, store.OnConfirmed("no-such"))

		// Fatal error from confirmed
		require.NoError(t, store.OnPending("h2", 2000, tx1))
		require.NoError(t, store.OnConfirmed(tx1.ID))
		require.NoError(t, store.OnFatalError(tx1.ID))
		require.Equal(t, trontxm.FatallyErrored, tx1.State)
		// retain id
		require.True(t, store.Has(tx1.ID))

		tx2 := &trontxm.TronTx{ID: "id2", FromAddress: genesisAddress}
		hash2 := "hash2"
		// OnErrored
		require.Error(t, store.OnErrored("no-such"))
		require.NoError(t, store.OnPending(hash2, 2000, tx2))
		require.NoError(t, store.OnErrored(tx2.ID))
		require.Equal(t, trontxm.Errored, tx2.State)
		require.True(t, store.Has(tx2.ID))

		// OnReorg
		require.NoError(t, store.OnPending("h3", 2000, tx2))
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
	})
}

func TestTxmRaceConditions(t *testing.T) {
	t.Parallel()

	t.Run("Concurrent enqueue operations", func(t *testing.T) {
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)

		raceConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 300,
			ConfirmPollSecs:   1,
			FinalityDepth:     10,
			RetentionPeriod:   10 * time.Second,
			ReapInterval:      1 * time.Second,
		}

		txm, testLogger, _ := setupTxm(t, fullNodeClient, raceConfig)
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
		fullNodeClient := createDefaultMockClient(t)
		txm, _, _ := setupTxm(t, fullNodeClient, nil)
		defer txm.Close()

		store := txm.AccountStore.GetTxStore(genesisAddress.String())

		numGoroutines := 100
		var wg sync.WaitGroup

		tx := &trontxm.TronTx{ID: "race_test_tx", FromAddress: genesisAddress}
		hash := "race_test_hash"
		require.NoError(t, store.OnPending(hash, 2000, tx))

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(routineID int) {
				defer wg.Done()
				switch routineID % 4 {
				case 0:
					store.OnBroadcasted(tx.ID)
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
		fullNodeClient := createDefaultMockClient(t)
		txm, _, _ := setupTxm(t, fullNodeClient, nil)
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

				store.OnPending(hash, time.Now().UnixMilli()+10000, tx)
				store.OnBroadcasted(tx.ID)
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
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)
		txm, _, _ := setupTxm(t, fullNodeClient, nil)
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

		initialStatus, err := txm.GetTransactionStatus(context.Background(), txID)
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
				status, err := txm.GetTransactionStatus(context.Background(), txID)
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

func TestTxmLoadTesting(t *testing.T) {
	t.Parallel()
	t.Run("High volume transaction enqueueing", func(t *testing.T) {
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)

		loadConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 300,
			ConfirmPollSecs:   1,
			FinalityDepth:     5,
			RetentionPeriod:   5 * time.Second,
			ReapInterval:      500 * time.Millisecond,
		}

		txm, testLogger, _ := setupTxm(t, fullNodeClient, loadConfig)
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
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)

		smallConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 10,
			ConfirmPollSecs:   1,
			FinalityDepth:     5,
			RetentionPeriod:   5 * time.Second,
			ReapInterval:      500 * time.Millisecond,
		}

		txm, testLogger, _ := setupTxm(t, fullNodeClient, smallConfig)
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
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)
		txm, testLogger, _ := setupTxm(t, fullNodeClient, nil)
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
		fullNodeClient := createDefaultMockClient(t)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Maybe().Return(&soliditynode.TransactionInfo{
			Receipt:     soliditynode.ResourceReceipt{Result: "SUCCESS"},
			BlockNumber: 123,
		}, nil)

		shortConfig := &trontxm.TronTxmConfig{
			BroadcastChanSize: 100,
			ConfirmPollSecs:   1,
			FinalityDepth:     1,
			RetentionPeriod:   100 * time.Millisecond,
			ReapInterval:      50 * time.Millisecond,
		}

		txm, _, _ := setupTxm(t, fullNodeClient, shortConfig)
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
