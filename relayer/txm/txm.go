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

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/utils"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
)

var _ services.Service = &TronTxm{}

const (
	MAX_RETRY_ATTEMPTS           = 5
	MAX_BROADCAST_RETRY_DURATION = 30 * time.Second
	BROADCAST_DELAY_DURATION     = 2 * time.Second
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

func New(lgr logger.Logger, keystore loop.Keystore, client sdk.FullNodeClient, config TronTxmConfig) *TronTxm {
	return &TronTxm{
		Logger:                logger.Named(lgr, "TronTxm"),
		Keystore:              keystore,
		Config:                config,
		EstimateEnergyEnabled: true,

		Client:        client,
		BroadcastChan: make(chan *TronTx, config.BroadcastChanSize),
		AccountStore:  NewAccountStore(),
		Stop:          make(chan struct{}),
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
		t.Done.Add(2) // waitgroup: broadcast loop and confirm loop
		go t.broadcastLoop()
		go t.confirmLoop()
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
func (t *TronTxm) Enqueue(fromAddress, contractAddress address.Address, method string, params ...any) error {
	if _, err := t.Keystore.Sign(context.Background(), fromAddress.String(), nil); err != nil {
		return fmt.Errorf("failed to sign: %+w", err)
	}

	if len(params)%2 == 1 {
		return fmt.Errorf("odd number of params")
	}
	for i := 0; i < len(params); i += 2 {
		paramType := params[i]
		_, ok := paramType.(string)
		if !ok {
			return fmt.Errorf("non-string param type")
		}
	}

	tx := &TronTx{FromAddress: fromAddress, ContractAddress: contractAddress, Method: method, Params: params, Attempt: 1}

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
				t.Logger.Errorw("failed to trigger smart contract", "error", err, "tx", tx)
				continue
			}

			coreTx := triggerResponse.Transaction
			txHash := coreTx.TxID

			// RefBlockNum is optional and does not seem in use anymore.
			t.Logger.Debugw("created transaction", "method", tx.Method, "txHash", txHash, "timestampMs", coreTx.RawData.Timestamp, "expirationMs", coreTx.RawData.Expiration, "refBlockHash", coreTx.RawData.RefBlockHash, "feeLimit", coreTx.RawData.FeeLimit)

			_, err = t.SignAndBroadcast(ctx, tx.FromAddress, coreTx)
			if err != nil {
				t.Logger.Errorw("transaction failed to broadcast", "txHash", txHash, "error", err, "tx", tx, "triggerResponse", triggerResponse)
				continue
			}

			t.Logger.Infow("transaction broadcasted", "method", tx.Method, "txHash", txHash, "timestampMs", coreTx.RawData.Timestamp, "expirationMs", coreTx.RawData.Expiration, "refBlockHash", coreTx.RawData.RefBlockHash, "feeLimit", coreTx.RawData.FeeLimit)

			txStore := t.AccountStore.GetTxStore(tx.FromAddress.String())
			txStore.AddUnconfirmed(txHash, coreTx.RawData.Expiration, tx)

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
	paddedFeeLimit := CalculatePaddedFeeLimit(feeLimit, tx.EnergyBumpTimes)

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
			if err != nil {
				// if the transaction has expired and we still can't find the hash, rebroadcast.
				if unconfirmedTx.ExpirationMs < timestampMs {
					err = t.AccountStore.GetTxStore(fromAddress).Confirm(unconfirmedTx.Hash)
					if err != nil {
						t.Logger.Errorw("could not confirm expired transaction locally", "error", err)
						continue
					}
					t.Logger.Debugw("transaction missing after expiry", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "timestampMs", timestampMs, "expirationMs", unconfirmedTx.ExpirationMs)
					t.maybeRetry(unconfirmedTx, false, false)
				}
				continue
			}
			err = t.AccountStore.GetTxStore(fromAddress).Confirm(unconfirmedTx.Hash)
			if err != nil {
				t.Logger.Errorw("could not confirm transaction locally", "error", err)
				continue
			}
			receipt := txInfo.Receipt
			contractResult := receipt.Result
			switch contractResult {
			case soliditynode.TransactionResultOutOfEnergy:
				t.Logger.Errorw("transaction failed due to out of energy", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, true, false)
				continue
			case soliditynode.TransactionResultOutOfTime:
				t.Logger.Errorw("transaction failed due to out of time", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, false, true)
				continue
			case soliditynode.TransactionResultUnknown:
				t.Logger.Errorw("transaction failed due to unknown error", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, false, false)
				continue
			}
			t.Logger.Infow("confirmed transaction", "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "contractResult", contractResult)
		}
	}
}

func (t *TronTxm) maybeRetry(unconfirmedTx *UnconfirmedTx, bumpEnergy bool, isOutOfTimeError bool) {
	tx := unconfirmedTx.Tx
	if tx.Attempt >= MAX_RETRY_ATTEMPTS {
		t.Logger.Debugw("not retrying, already reached max retries", "txHash", unconfirmedTx.Hash, "lastAttempt", tx.Attempt, "bumpEnergy", bumpEnergy, "isOutOfTimeError", isOutOfTimeError)
		return
	}
	if tx.OutOfTimeErrors >= 2 {
		t.Logger.Debugw("not retrying, multiple OUT_OF_TIME errors", "txHash", unconfirmedTx.Hash, "lastAttempt", tx.Attempt, "bumpEnergy", bumpEnergy, "isOutOfTimeError", isOutOfTimeError)
		return
	}

	tx.Attempt += 1
	if bumpEnergy {
		tx.EnergyBumpTimes += 1
	}
	if isOutOfTimeError {
		tx.OutOfTimeErrors += 1
	}

	t.Logger.Infow("retrying transaction", "previousTxHash", unconfirmedTx.Hash, "attempt", tx.Attempt, "bumpEnergy", bumpEnergy, "isOutOfTimeError", isOutOfTimeError)

	select {
	case t.BroadcastChan <- tx:
	default:
		t.Logger.Errorw("failed to enqueue retry transaction", "previousTxHash", unconfirmedTx.Hash)
	}
}

func (t *TronTxm) InflightCount() (int, int) {
	return len(t.BroadcastChan), t.AccountStore.GetTotalInflightCount()
}

func (t *TronTxm) estimateEnergy(tx *TronTx) (int64, error) {

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
