package txm

import (
	"context"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/fullnode"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
	"github.com/google/uuid"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	commontypes "github.com/smartcontractkit/chainlink-common/pkg/types"
	"github.com/smartcontractkit/chainlink-common/pkg/utils"

	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
)

var _ services.Service = &TronTxm{}

const (
	MAX_RETRY_ATTEMPTS           = 5
	MAX_BROADCAST_RETRY_DURATION = 30 * time.Second
	BROADCAST_DELAY_DURATION     = 2 * time.Second
	DEFAULT_ENERGY_MULTIPLIER    = 1.5
	REORG_RETRY_COUNT            = 3
	REORG_RETRY_DELAY            = 500 * time.Millisecond
)

type TronTxm struct {
	Logger                logger.Logger
	Keystore              loop.Keystore
	Config                TronTxmConfig
	EstimateEnergyEnabled bool // TODO: Move this to a NodeState/Config struct when we move to MultiNode

	Client        sdk.CombinedClient
	BroadcastChan chan *TronTx
	AccountStore  *AccountStore
	Starter       utils.StartStopOnce
	Done          sync.WaitGroup
	Stop          chan struct{}
}

type TronTxmRequest struct {
	FromAddress     address.Address
	ContractAddress address.Address
	Method          string
	Params          []any
	ID              string
}

func New(lgr logger.Logger, keystore loop.Keystore, client sdk.CombinedClient, config TronTxmConfig) *TronTxm {
	txm := &TronTxm{
		Logger:                logger.Named(lgr, "TronTxm"),
		Keystore:              keystore,
		Config:                config,
		EstimateEnergyEnabled: true,
		Client:                client,
		BroadcastChan:         make(chan *TronTx, config.BroadcastChanSize),
		AccountStore:          NewAccountStore(),
		Stop:                  make(chan struct{}),
	}

	// Set defaults for missing config values
	txm.setDefaults()

	return txm
}

func (t *TronTxm) setDefaults() {
	if t.Config.EnergyMultiplier == 0 || t.Config.EnergyMultiplier < 1.0 {
		t.Logger.Warnw("Energy multiplier is not set, using default value", "default", DEFAULT_ENERGY_MULTIPLIER)
		t.Config.EnergyMultiplier = DEFAULT_ENERGY_MULTIPLIER
	}
}

func (t *TronTxm) Name() string {
	return t.Logger.Name()
}

func (t *TronTxm) Ready() error {
	return t.Starter.Ready()
}

func (t *TronTxm) HealthReport() map[string]error {
	return map[string]error{t.Name(): t.Starter.Healthy()}
}

func (t *TronTxm) GetClient() sdk.CombinedClient {
	return t.Client
}

func (t *TronTxm) Start(ctx context.Context) error {
	return t.Starter.StartOnce("TronTxm", func() error {
		t.Done.Add(3) // waitgroup: broadcast loop, confirm loop, and reap loop
		go t.broadcastLoop()
		go t.confirmLoop()
		go t.reapLoop()

		return nil
	})
}

func (t *TronTxm) Close() error {
	return t.Starter.StopOnce("TronTxm", func() error {
		close(t.Stop)
		t.Done.Wait()
		return nil
	})
}

// Enqueues a transaction for broadcasting.
// Each item in the params array should be a map with a single key-value pair, where
// the key is the ABI type.
func (t *TronTxm) Enqueue(request TronTxmRequest) error {
	if _, err := t.Keystore.Sign(context.Background(), request.FromAddress.String(), nil); err != nil {
		return fmt.Errorf("failed to sign: %+w", err)
	}

	if len(request.Params)%2 == 1 {
		return fmt.Errorf("odd number of params")
	}

	for i := 0; i < len(request.Params); i += 2 {
		paramType := request.Params[i]
		_, ok := paramType.(string)
		if !ok {
			return fmt.Errorf("non-string param type")
		}
	}

	if request.ID == "" {
		request.ID = uuid.New().String()
	} else {
		// don’t enqueue twice for the same key
		txStore := t.AccountStore.GetTxStore(request.FromAddress.String())
		if txStore.Has(request.ID) {
			t.Logger.Warnw("transaction with ID already exists, ignoring", "txID", request.ID)
			return nil
		}
	}

	// Construct the transaction
	tx := &TronTx{FromAddress: request.FromAddress, ContractAddress: request.ContractAddress, Method: request.Method, Params: request.Params, Attempt: 1, ID: request.ID, CreateTs: time.Now()}
	txStore := t.AccountStore.GetTxStore(tx.FromAddress.String())
	txStore.OnPending(tx, false)

	select {
	case t.BroadcastChan <- tx:
	default:
		return fmt.Errorf("failed to enqueue transaction: %+v", tx)
	}

	return nil
}

func (t *TronTxm) broadcastLoop() {
	defer t.Done.Done()

	ctx, cancel := utils.ContextFromChan(t.Stop)
	defer cancel()

	t.Logger.Debugw("broadcastLoop: started")
	for {
		select {
		case tx := <-t.BroadcastChan:
			triggerResponse, err := t.TriggerSmartContract(ctx, tx)
			if err != nil {
				// TODO: is it ok to leave this transaction unmarked as fatal?
				t.Logger.Errorw("failed to trigger smart contract", "error", err, "tx", tx, "txID", tx.ID)
				continue
			}

			coreTx := triggerResponse.Transaction
			txHash := coreTx.TxID

			// RefBlockNum is optional and does not seem in use anymore.
			t.Logger.Debugw("created transaction", "method", tx.Method, "txHash", txHash, "timestampMs", coreTx.RawData.Timestamp, "expirationMs", coreTx.RawData.Expiration, "refBlockHash", coreTx.RawData.RefBlockHash, "feeLimit", coreTx.RawData.FeeLimit, "txID", tx.ID)
			txStore := t.AccountStore.GetTxStore(tx.FromAddress.String())

			_, err = t.SignAndBroadcast(ctx, tx.FromAddress, coreTx)
			if err != nil {
				t.Logger.Errorw("transaction failed to broadcast", "txHash", txHash, "error", err, "tx", tx, "triggerResponse", triggerResponse, "txID", tx.ID)
				txStore.OnFatalError(tx.ID)
				continue
			}

			t.Logger.Infow("transaction broadcasted", "method", tx.Method, "txHash", txHash, "timestampMs", coreTx.RawData.Timestamp, "expirationMs", coreTx.RawData.Expiration, "refBlockHash", coreTx.RawData.RefBlockHash, "feeLimit", coreTx.RawData.FeeLimit, "txID", tx.ID)

			txStore.OnBroadcasted(txHash, coreTx.RawData.Expiration, tx)
		case <-t.Stop:
			t.Logger.Debugw("broadcastLoop: stopped")
			return
		}
	}
}

func (t *TronTxm) TriggerSmartContract(ctx context.Context, tx *TronTx) (*fullnode.TriggerSmartContractResponse, error) {
	energyUsed, err := t.estimateEnergy(tx)
	if err != nil {
		return nil, fmt.Errorf("failed to estimate energy: %+w", err)
	}

	energyUnitPrice := DEFAULT_ENERGY_UNIT_PRICE

	if energyPrices, err := t.GetClient().GetEnergyPrices(); err == nil {
		if parsedPrice, err := ParseLatestEnergyPrice(energyPrices.Prices); err == nil {
			energyUnitPrice = parsedPrice
		} else {
			t.Logger.Errorw("error parsing energy unit price", "error", err, "txID", tx.ID)
		}
	} else {
		t.Logger.Errorw("failed to get energy unit price", "error", err, "txID", tx.ID)
	}

	feeLimit := energyUnitPrice * int32(energyUsed)
	paddedFeeLimit := CalculatePaddedFeeLimit(feeLimit, tx.EnergyBumpTimes, t.Config.EnergyMultiplier)

	t.Logger.Debugw("Trigger smart contract", "energyBumpTimes", tx.EnergyBumpTimes, "energyUnitPrice", energyUnitPrice, "feeLimit", feeLimit, "paddedFeeLimit", paddedFeeLimit, "txID", tx.ID)

	txExtention, err := t.GetClient().TriggerSmartContract(
		tx.FromAddress,
		tx.ContractAddress,
		tx.Method,
		tx.Params,
		paddedFeeLimit,
		/* tAmount= (TRX amount) */ 0)
	if err != nil {
		return nil, fmt.Errorf("failed to call TriggerSmartContract: %+w", err)
	}

	return txExtention, nil
}

func (t *TronTxm) SignAndBroadcast(ctx context.Context, fromAddress address.Address, coreTx *common.Transaction) (*fullnode.BroadcastResponse, error) {
	txIdBytes, err := hex.DecodeString(coreTx.TxID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction id: %+w", err)
	}

	signature, err := t.Keystore.Sign(ctx, fromAddress.String(), txIdBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %+w", err)
	}

	coreTx.AddSignatureBytes(signature)

	// the broadcast response code and error message is already checked by the full node client's BroadcastTranssaction function,
	// and embedded inside `err`.
	broadcastResponse, err := t.broadcastTx(coreTx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %+w", err)
	}

	return broadcastResponse, nil
}

func (t *TronTxm) broadcastTx(tx *common.Transaction) (*fullnode.BroadcastResponse, error) {
	var broadcastResponse *fullnode.BroadcastResponse
	var err error
	startTime := time.Now()
	attempt := 1
	for time.Since(startTime) < MAX_BROADCAST_RETRY_DURATION {
		broadcastResponse, err = t.GetClient().BroadcastTransaction(tx)
		if err == nil {
			break
		}

		// unsuccessful, check response code
		if !broadcastResponse.Result {
			if broadcastResponse.Code == common.ResponseCodeServerBusy || broadcastResponse.Code == common.ResponseCodeBlockUnsolidified {
				// wait and retry tx broadcast upon SERVER_BUSY and BLOCK_UNSOLIDIFIED error responses
				t.Logger.Debugw("SERVER_BUSY or BLOCK_UNSOLIDIFIED: retry broadcast after timeout", "attempt", attempt)
				time.Sleep(BROADCAST_DELAY_DURATION)
				attempt = attempt + 1
				continue
			} else {
				// do not retry on other broadcast errors. the error message and code is encoded in `err`.
				return nil, err
			}
		}
	}
	if err != nil {
		return nil, fmt.Errorf("SERVER_BUSY or BLOCK_UNSOLIDIFIED: max retry duration reached, error: %w", err)
	}
	return broadcastResponse, nil
}

func (t *TronTxm) confirmLoop() {
	defer t.Done.Done()

	_, cancel := utils.ContextFromChan(t.Stop)
	defer cancel()

	pollDuration := time.Duration(t.Config.ConfirmPollSecs) * time.Second
	tick := time.After(pollDuration)

	t.Logger.Debugw("confirmLoop: started")

	for {
		select {
		case <-tick:
			start := time.Now()

			t.checkUnconfirmed()
			t.checkFinalized()

			remaining := pollDuration - time.Since(start)
			tick = time.After(utils.WithJitter(remaining.Abs()))
		case <-t.Stop:
			t.Logger.Debugw("confirmLoop: stopped")
			return
		}
	}
}

func (t *TronTxm) checkUnconfirmed() {
	allUnconfirmedTxs := t.AccountStore.GetAllUnconfirmed()
	for fromAddress, unconfirmedTxs := range allUnconfirmedTxs {
		nowBlock, err := t.GetClient().GetNowBlockFullNode()
		if err != nil {
			t.Logger.Errorw("could not get latest block", "error", err)
			continue
		}
		if nowBlock.BlockHeader == nil || nowBlock.BlockHeader.RawData == nil {
			t.Logger.Errorw("could not read latest block header")
			continue
		}
		timestampMs := nowBlock.BlockHeader.RawData.Timestamp
		for _, unconfirmedTx := range unconfirmedTxs {
			// use fullnode endpoint for unfinalized data
			txInfo, err := t.GetClient().GetTransactionInfoByIdFullNode(unconfirmedTx.Hash)
			txStore := t.AccountStore.GetTxStore(fromAddress)

			if err != nil {
				// if the transaction has expired and we still can't find the hash, rebroadcast.
				if unconfirmedTx.ExpirationMs < timestampMs {
					t.Logger.Debugw("transaction missing after expiry", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "timestampMs", timestampMs, "expirationMs", unconfirmedTx.ExpirationMs, "txID", unconfirmedTx.Tx.ID)
					t.maybeRetry(unconfirmedTx, false, false, txStore)
				}
				continue
			}

			receipt := txInfo.Receipt
			contractResult := receipt.Result

			if contractResult == soliditynode.TransactionResultSuccess {
				err = txStore.OnConfirmed(unconfirmedTx.Tx.ID)
				if err != nil {
					t.Logger.Errorw("could not confirm transaction locally", "error", err, "txID", unconfirmedTx.Tx.ID)
					continue
				}
				t.Logger.Infow("confirmed transaction", "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "contractResult", contractResult, "txID", unconfirmedTx.Tx.ID)
				continue
			}

			switch contractResult {
			case soliditynode.TransactionResultOutOfEnergy:
				t.Logger.Errorw("transaction failed due to out of energy", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "txID", unconfirmedTx.Tx.ID)
				t.maybeRetry(unconfirmedTx, true, false, txStore)
				continue
			case soliditynode.TransactionResultOutOfTime:
				t.Logger.Errorw("transaction failed due to out of time", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "txID", unconfirmedTx.Tx.ID)
				t.maybeRetry(unconfirmedTx, false, true, txStore)
				continue
			case soliditynode.TransactionResultRevert,
				soliditynode.TransactionResultBadJumpDestination,
				soliditynode.TransactionResultOutOfMemory,
				soliditynode.TransactionResultStackTooSmall,
				soliditynode.TransactionResultStackTooLarge,
				soliditynode.TransactionResultIllegalOperation,
				soliditynode.TransactionResultStackOverflow,
				soliditynode.TransactionResultJvmStackOverflow,
				soliditynode.TransactionResultTransferFailed,
				soliditynode.TransactionResultInvalidCode:
				// fatal error
				t.Logger.Errorw("transaction failed with fatal error", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "contractResult", contractResult, "txID", unconfirmedTx.Tx.ID)
				if err := txStore.OnFatalError(unconfirmedTx.Tx.ID); err != nil {
					t.Logger.Errorw("failed to mark transaction as fatally errored", "txID", unconfirmedTx.Tx.ID, "error", err)
				}
				continue
			case soliditynode.TransactionResultUnknown, soliditynode.TransactionResultDefault:
				// retry unknown error
				t.Logger.Errorw("transaction failed due to unknown error", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "txID", unconfirmedTx.Tx.ID)
				t.maybeRetry(unconfirmedTx, false, false, txStore)
				continue
			default:
				// Unhandled result type - treat as unknown
				t.Logger.Errorw("transaction failed with unhandled result type", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "contractResult", contractResult, "txID", unconfirmedTx.Tx.ID)
				t.maybeRetry(unconfirmedTx, false, false, txStore)
				continue
			}
		}
	}
}

func (t *TronTxm) maybeRetry(unconfirmedTx *InflightTx, bumpEnergy bool, isOutOfTimeError bool, txStore *TxStore) {
	tx := unconfirmedTx.Tx

	if tx.Attempt >= MAX_RETRY_ATTEMPTS {
		t.Logger.Debugw("not retrying, already reached max retries", "txHash", unconfirmedTx.Hash, "lastAttempt", tx.Attempt, "bumpEnergy", bumpEnergy, "isOutOfTimeError", isOutOfTimeError, "txID", tx.ID)
		if err := txStore.OnErrored(tx.ID); err != nil {
			t.Logger.Errorw("failed to mark transaction as errored", "txID", tx.ID, "error", err)
		}
		return
	}
	if tx.OutOfTimeErrors >= 2 {
		t.Logger.Debugw("not retrying, multiple OUT_OF_TIME errors", "txHash", unconfirmedTx.Hash, "lastAttempt", tx.Attempt, "bumpEnergy", bumpEnergy, "isOutOfTimeError", isOutOfTimeError, "txID", tx.ID)
		if err := txStore.OnErrored(tx.ID); err != nil {
			t.Logger.Errorw("failed to mark transaction as errored", "txID", tx.ID, "error", err)
		}
		return
	}

	tx.Attempt += 1
	if bumpEnergy {
		tx.EnergyBumpTimes += 1
	}
	if isOutOfTimeError {
		tx.OutOfTimeErrors += 1
	}

	t.Logger.Infow("retrying transaction", "txID", tx.ID, "previousTxHash", unconfirmedTx.Hash, "attempt", tx.Attempt, "bumpEnergy", bumpEnergy, "isOutOfTimeError", isOutOfTimeError)
	tx.State = Pending
	txStore.OnPending(tx, true)
	select {
	// TODO: do we need to retry here or mark as fatal?
	case t.BroadcastChan <- tx:
	default:
		t.Logger.Errorw("failed to enqueue retry transaction", "previousTxHash", unconfirmedTx.Hash, "txID", tx.ID)
	}
}

func (t *TronTxm) checkFinalized() {
	allConfirmed := t.AccountStore.GetAllConfirmed()
	for acc, confirmedTxs := range allConfirmed {
		store := t.AccountStore.GetTxStore(acc)

		for _, pt := range confirmedTxs {
			txId := pt.Tx.ID
			txHash := pt.Hash

			_, err := t.GetClient().GetTransactionInfoById(txHash)
			if err != nil && t.checkReorged(txHash) {
				t.Logger.Warnw("tx missing after reorg, moving back to unconfirmed", "txID", txId)
				if derr := store.OnReorg(txId); derr != nil {
					t.Logger.Errorw("failed to OnReorg tx", "txID", txId, "error", derr)
				} else {
					select {
					case t.BroadcastChan <- pt.Tx:
						// Successfully enqueued the transaction
					default:
						t.Logger.Warnw("Broadcast channel is full, dropping transaction", "txID", txId)
					}
				}
				continue
			}

			if err := store.OnFinalized(txId); err != nil {
				t.Logger.Errorw("failed to finalize tx", "txID", txId, "error", err)
			} else {
				t.Logger.Infow("finalized transaction", "txID", txId)
			}
		}
	}
}

func (t *TronTxm) reapLoop() {
	defer t.Done.Done()
	ticker := time.NewTicker(t.Config.ReapInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			cutoff := time.Now().Add(-t.Config.RetentionPeriod)
			allFinished := t.AccountStore.GetAllFinished()
			accountTxIds := make(map[string][]string)

			for acc, finishedTxs := range allFinished {
				var idsToDelete []string
				for _, ft := range finishedTxs {
					if ft.RetentionTs.Before(cutoff) {
						idsToDelete = append(idsToDelete, ft.Tx.ID)
					}
				}
				if len(idsToDelete) > 0 {
					accountTxIds[acc] = idsToDelete
				}
			}

			if len(accountTxIds) > 0 {
				reapCount := t.AccountStore.DeleteAllFinishedTxs(accountTxIds)
				if reapCount > 0 {
					t.Logger.Debugw("reapLoop: reaped finished transactions", "count", reapCount)
				}
			}
		case <-t.Stop:
			t.Logger.Debugw("reapLoop: stopped")
			return
		}
	}
}

// GetTransactionStatus translates internal TXM transaction statuses to chainlink common statuses
func (t *TronTxm) GetTransactionStatus(ctx context.Context, transactionID string) (commontypes.TransactionStatus, error) {
	state, exists := t.AccountStore.GetStatusAll(transactionID)
	if exists {
		switch state {
		case Pending, Broadcasted:
			return commontypes.Pending, nil
		case Confirmed:
			return commontypes.Unconfirmed, nil
		case Finalized:
			return commontypes.Finalized, nil
		case Errored:
			return commontypes.Failed, nil
		case FatallyErrored:
			return commontypes.Fatal, nil
		default:
			return commontypes.Unknown, fmt.Errorf("found unknown transaction state for id %s", transactionID)
		}
	}
	return commontypes.Unknown, fmt.Errorf("failed to find transaction with id %s", transactionID)
}

func (t *TronTxm) InflightCount() (int, int) {
	return len(t.BroadcastChan), t.AccountStore.GetTotalInflightCount()
}

func (t *TronTxm) estimateEnergy(tx *TronTx) (int64, error) {
	if t.Config.FixedEnergyValue != 0 {
		return t.Config.FixedEnergyValue, nil
	}

	if t.EstimateEnergyEnabled {
		estimateEnergyMessage, err := t.GetClient().EstimateEnergy(
			tx.FromAddress,
			tx.ContractAddress,
			tx.Method,
			tx.Params,
			/* tAmount= */ 0,
		)
		if err == nil {
			t.Logger.Debugw("Estimated energy using EnergyEstimation Method", "energyRequired", estimateEnergyMessage.EnergyRequired, "tx", tx, "txID", tx.ID)
			return estimateEnergyMessage.EnergyRequired, nil
		}

		if strings.Contains(err.Error(), "this node does not support estimate energy") {
			t.EstimateEnergyEnabled = false
			t.Logger.Infow("Node does not support EstimateEnergy", "err", err, "tx", tx, "txID", tx.ID)
		} else {
			t.Logger.Errorw("Failed to call EstimateEnergy", "err", err, "tx", tx, "txID", tx.ID)
		}
	}

	// Using TriggerConstantContract as EstimateEnergy is unsupported or failed.
	triggerResponse, err := t.GetClient().TriggerConstantContractFullNode(tx.FromAddress, tx.ContractAddress, tx.Method, tx.Params)
	if err != nil {
		return 0, fmt.Errorf("failed to call TriggerConstantContract: %w", err)
	}
	if !triggerResponse.Result.Result {
		return 0, fmt.Errorf("failed to call TriggerConstantContract due to %s %s", triggerResponse.Result.Code, triggerResponse.Result.Message)
	}

	t.Logger.Debugw("Estimated energy using TriggerConstantContract Method", "energyUsed", triggerResponse.EnergyUsed, "energyPenalty", triggerResponse.EnergyPenalty, "tx", tx, "txID", tx.ID)

	return triggerResponse.EnergyUsed, nil
}

// checkReorged attempts to fetch transaction info with retries to distinguish
// between temporary RPC failures and actual chain reorgs.
func (t *TronTxm) checkReorged(txHash string) bool {
	_, err := t.GetClient().GetTransactionInfoByIdFullNode(txHash)
	if err == nil {
		return false
	}

	ticker := time.NewTicker(REORG_RETRY_DELAY)
	defer ticker.Stop()

	for retry := uint(0); retry < REORG_RETRY_COUNT; retry++ {
		<-ticker.C
		_, err = t.GetClient().GetTransactionInfoByIdFullNode(txHash)
		if err == nil {
			return false
		}
	}

	return true
}
