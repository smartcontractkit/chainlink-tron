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
)

type TronTxm struct {
	Logger                logger.Logger
	Keystore              loop.Keystore
	Config                TronTxmConfig
	EstimateEnergyEnabled bool // TODO: Move this to a NodeState/Config struct when we move to MultiNode

	Client        sdk.FullNodeClient
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

func New(lgr logger.Logger, keystore loop.Keystore, client sdk.FullNodeClient, config TronTxmConfig) *TronTxm {
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

func (t *TronTxm) GetClient() sdk.FullNodeClient {
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
			return fmt.Errorf("transaction with ID %s already exists", request.ID)
		}
	}

	// Construct the transaction
	tx := &TronTx{FromAddress: request.FromAddress, ContractAddress: request.ContractAddress, Method: request.Method, Params: request.Params, Attempt: 1, ID: request.ID, CreateTs: time.Now()}

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
				t.Logger.Errorw("failed to trigger smart contract", "error", err, "tx", tx)
				continue
			}

			coreTx := triggerResponse.Transaction
			txHash := coreTx.TxID

			// RefBlockNum is optional and does not seem in use anymore.
			t.Logger.Debugw("created transaction", "method", tx.Method, "txHash", txHash, "timestampMs", coreTx.RawData.Timestamp, "expirationMs", coreTx.RawData.Expiration, "refBlockHash", coreTx.RawData.RefBlockHash, "feeLimit", coreTx.RawData.FeeLimit)
			txStore := t.AccountStore.GetTxStore(tx.FromAddress.String())
			txStore.OnPending(txHash, coreTx.RawData.Expiration, tx)

			_, err = t.SignAndBroadcast(ctx, tx.FromAddress, coreTx)
			if err != nil {
				t.Logger.Errorw("transaction failed to broadcast", "txHash", txHash, "error", err, "tx", tx, "triggerResponse", triggerResponse)
				txStore.OnFatalError(tx.ID)
				continue
			}

			t.Logger.Infow("transaction broadcasted", "method", tx.Method, "txHash", txHash, "timestampMs", coreTx.RawData.Timestamp, "expirationMs", coreTx.RawData.Expiration, "refBlockHash", coreTx.RawData.RefBlockHash, "feeLimit", coreTx.RawData.FeeLimit)

			txStore.OnBroadcasted(tx.ID)
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
			t.Logger.Errorw("error parsing energy unit price", "error", err)
		}
	} else {
		t.Logger.Errorw("failed to get energy unit price", "error", err)
	}

	feeLimit := energyUnitPrice * int32(energyUsed)
	paddedFeeLimit := CalculatePaddedFeeLimit(feeLimit, tx.EnergyBumpTimes, t.Config.EnergyMultiplier)

	t.Logger.Debugw("Trigger smart contract", "energyBumpTimes", tx.EnergyBumpTimes, "energyUnitPrice", energyUnitPrice, "feeLimit", feeLimit, "paddedFeeLimit", paddedFeeLimit)

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
		nowBlock, err := t.GetClient().GetNowBlock()
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
			txInfo, err := t.GetClient().GetTransactionInfoById(unconfirmedTx.Hash)
			txStore := t.AccountStore.GetTxStore(fromAddress)

			if err != nil {
				// if the transaction has expired and we still can't find the hash, rebroadcast.
				if unconfirmedTx.ExpirationMs < timestampMs {
					t.Logger.Debugw("transaction missing after expiry", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "timestampMs", timestampMs, "expirationMs", unconfirmedTx.ExpirationMs)
					t.maybeRetry(unconfirmedTx, false, false, txStore)
				}
				continue
			}
			err = txStore.OnConfirmed(unconfirmedTx.Tx.ID)
			if err != nil {
				t.Logger.Errorw("could not confirm transaction locally", "error", err)
				continue
			}
			receipt := txInfo.Receipt
			contractResult := receipt.Result
			switch contractResult {
			case soliditynode.TransactionResultOutOfEnergy:
				t.Logger.Errorw("transaction failed due to out of energy", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, true, false, txStore)
				continue
			case soliditynode.TransactionResultOutOfTime:
				t.Logger.Errorw("transaction failed due to out of time", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, false, true, txStore)
				continue
			case soliditynode.TransactionResultUnknown:
				t.Logger.Errorw("transaction failed due to unknown error", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, false, false, txStore)
				continue
			}
			t.Logger.Infow("confirmed transaction", "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "contractResult", contractResult)
		}
	}
}

func (t *TronTxm) maybeRetry(unconfirmedTx *PendingTx, bumpEnergy bool, isOutOfTimeError bool, txStore *TxStore) {
	tx := unconfirmedTx.Tx
	txStore.OnErrored(tx.ID)
	if tx.Attempt >= MAX_RETRY_ATTEMPTS {
		t.Logger.Debugw("not retrying, already reached max retries", "txHash", unconfirmedTx.Hash, "lastAttempt", tx.Attempt, "bumpEnergy", bumpEnergy, "isOutOfTimeError", isOutOfTimeError)
		txStore.OnFatalError(tx.ID)
		return
	}
	if tx.OutOfTimeErrors >= 2 {
		t.Logger.Debugw("not retrying, multiple OUT_OF_TIME errors", "txHash", unconfirmedTx.Hash, "lastAttempt", tx.Attempt, "bumpEnergy", bumpEnergy, "isOutOfTimeError", isOutOfTimeError)
		txStore.OnFatalError(tx.ID)
		return
	}

	tx.Attempt += 1
	if bumpEnergy {
		tx.EnergyBumpTimes += 1
	}
	if isOutOfTimeError {
		tx.OutOfTimeErrors += 1
	}

	// TODO: add ID to logger everywhere
	t.Logger.Infow("retrying transaction", "txID", tx.ID, "previousTxHash", unconfirmedTx.Hash, "attempt", tx.Attempt, "bumpEnergy", bumpEnergy, "isOutOfTimeError", isOutOfTimeError)

	select {
	// TODO: do we need to retry here or mark as fatal?
	case t.BroadcastChan <- tx:
	default:
		t.Logger.Errorw("failed to enqueue retry transaction", "previousTxHash", unconfirmedTx.Hash)
	}
}

func (t *TronTxm) checkFinalized() {
	nowBlk, err := t.GetClient().GetNowBlock()
	if err != nil {
		t.Logger.Errorw("could not get latest block for finalization", "error", err)
		return
	}
	currentHeight := nowBlk.BlockHeader.RawData.Number

	for acc := range t.AccountStore.store {
		store := t.AccountStore.GetTxStore(acc)
		store.lock.RLock()
		pts := make([]*PendingTx, 0, len(store.confirmedTxs))
		for _, pt := range store.confirmedTxs {
			pts = append(pts, pt)
		}
		store.lock.RUnlock()

		for _, pt := range pts {
			txInfo, err := t.GetClient().GetTransactionInfoById(pt.Hash)
			if err != nil {
				t.Logger.Warnw("tx missing after reorg, moving back to unconfirmed", "txID", pt.Tx.ID)
				if derr := store.OnReorg(pt.Tx.ID); derr != nil {
					t.Logger.Errorw("failed to OnReorg tx", "txID", pt.Tx.ID, "error", derr)
				}
				t.BroadcastChan <- pt.Tx
				continue
			}

			depth := currentHeight - txInfo.BlockNumber
			if depth < 0 {
				t.Logger.Warnf("RPC Lagging! Negative depth for tx %s, currentHeight: %d, txBlockNumber: %d", pt.Tx.ID, currentHeight, txInfo.BlockNumber)
				continue
			}

			convertedDepth := uint64(depth)
			if convertedDepth >= t.Config.FinalityDepth {
				if err := store.OnFinalized(pt.Tx.ID); err != nil {
					t.Logger.Errorw("failed to finalize tx", "txID", pt.Tx.ID, "error", err)
				} else {
					t.Logger.Infow("finalized transaction", "txID", pt.Tx.ID, "depth", depth)
				}
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
			for acc := range t.AccountStore.store {
				store := t.AccountStore.GetTxStore(acc)
				store.lock.Lock()
				for id, ft := range store.finishedTxs {
					if ft.Tx.CreateTs.Before(cutoff) {
						delete(store.finishedTxs, id)
					}
				}
				store.lock.Unlock()
			}
		case <-t.Stop:
			return
		}
	}
}

// GetTransactionStatus translates internal TXM transaction statuses to chainlink common statuses
func (t *TronTxm) GetTransactionStatus(ctx context.Context, transactionID string) (commontypes.TransactionStatus, error) {
	for acc := range t.AccountStore.store {
		store := t.AccountStore.GetTxStore(acc)
		state, exists := store.GetStatus(transactionID)
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
			t.Logger.Debugw("Estimated energy using EnergyEstimation Method", "energyRequired", estimateEnergyMessage.EnergyRequired, "tx", tx)
			return estimateEnergyMessage.EnergyRequired, nil
		}

		if strings.Contains(err.Error(), "this node does not support estimate energy") {
			t.EstimateEnergyEnabled = false
			t.Logger.Infow("Node does not support EstimateEnergy", "err", err, "tx", tx)
		} else {
			t.Logger.Errorw("Failed to call EstimateEnergy", "err", err, "tx", tx)
		}
	}

	// Using TriggerConstantContract as EstimateEnergy is unsupported or failed.
	triggerResponse, err := t.GetClient().TriggerConstantContract(tx.FromAddress, tx.ContractAddress, tx.Method, tx.Params)
	if err != nil {
		return 0, fmt.Errorf("failed to call TriggerConstantContract: %w", err)
	}
	if !triggerResponse.Result.Result {
		return 0, fmt.Errorf("failed to call TriggerConstantContract due to %s %s", triggerResponse.Result.Code, triggerResponse.Result.Message)
	}

	t.Logger.Debugw("Estimated energy using TriggerConstantContract Method", "energyUsed", triggerResponse.EnergyUsed, "energyPenalty", triggerResponse.EnergyPenalty, "tx", tx)

	return triggerResponse.EnergyUsed, nil
}
