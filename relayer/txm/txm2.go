package txm

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/fullnode"
	"github.com/google/uuid"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
)

type Txm struct {
	services.StateMachine
	lggr            logger.Logger
	energyEstimator EnergyEstimator

	chBroadcast chan pendingTx
	chStop      services.StopChan
	done        sync.WaitGroup
	cfg         TronTxmConfig
	txStore     *InMemoryTxStore
	ks          loop.Keystore
	client      sdk.FullNodeClient
}

func NewTxm(lggr logger.Logger, ks loop.Keystore, client sdk.FullNodeClient, cfg TronTxmConfig) *Txm {
	return &Txm{
		lggr:            lggr,
		energyEstimator: NewBasicEnergyEstimator(lggr, client, true),
		chBroadcast:     make(chan pendingTx, cfg.BroadcastChanSize),
		chStop:          make(services.StopChan),
		done:            sync.WaitGroup{},
		cfg:             cfg,
		txStore:         NewInMemoryTxStore(),
		ks:              ks,
		client:          client,
	}
}

func (t *Txm) Enqueue(request *TronTxmRequest) error {
	if _, err := t.ks.Sign(context.Background(), request.FromAddress.String(), nil); err != nil {
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

	// Construct the transaction
	tronTx := &TronTx{FromAddress: request.FromAddress, ContractAddress: request.ContractAddress, Method: request.Method, Params: request.Params, Attempt: 1, Meta: request.Meta}

	// Use transaction ID provided by caller if set
	id := uuid.NewString()
	if request.Id != "" {
		id = request.Id
	}

	// Create the pending transaction
	tx := pendingTx{
		tx:    tronTx,
		id:    id,
		state: AwaitingBroadcast,
	}

	// Send the transaction to the broadcast channel to be picked up by the broadcast loop
	select {
	case t.chBroadcast <- tx:
	default:
		return fmt.Errorf("failed to enqueue transaction: %+v", tx)
	}

	return nil
}

func (txm *Txm) Start(ctx context.Context) error {
	return txm.StartOnce("Txm", func() error {
		txm.lggr.Info("Starting TronTxm")
		if err := txm.energyEstimator.Start(ctx); err != nil {
			return fmt.Errorf("failed to start energy estimator: %w", err)
		}
		txm.done.Add(1)
		go txm.broadcastLoop()
		return nil
	})
}

func (txm *Txm) Close() error {
	return txm.StopOnce("Txm", func() error {
		close(txm.chStop)
		txm.done.Wait()
		return nil
	})
}

func (txm *Txm) Name() string {
	return txm.lggr.Name()
}

func (txm *Txm) HealthReport() map[string]error {
	return map[string]error{txm.Name(): txm.Healthy()}
}

func (t *Txm) GetClient() sdk.FullNodeClient {
	return t.client
}

func (txm *Txm) defaultConfig() TronTxmConfig {
	return TronTxmConfig{
		BroadcastChanSize: 1000,
		StatusChecker:     nil,
	}
}

func (txm *Txm) broadcastLoop() {
	defer txm.done.Done()
	ctx, cancel := txm.chStop.NewCtx()
	defer cancel()

	for {
		select {
		case tx := <-txm.chBroadcast:
			txm.lggr.Info("Received transaction", "tx", tx)
			constructedTx, err := txm.ConstructTx(&tx)
			if err != nil {
				txm.lggr.Errorw("Failed to construct transaction", "err", err, "tx", tx)
				txm.txStore.OnErrored(tx)
				continue
			}

			txm.lggr.Debugw("created transaction", "method", tx.tx.Method, "txHash", constructedTx.TxID, "timestampMs", constructedTx.RawData.Timestamp, "expirationMs", constructedTx.RawData.Expiration, "refBlockHash", constructedTx.RawData.RefBlockHash, "feeLimit", constructedTx.RawData.FeeLimit)
			_, err = txm.SignAndBroadcast(ctx, tx.tx.FromAddress.String(), constructedTx)
			if err != nil {
				txm.lggr.Errorw("Failed to sign and broadcast transaction", "err", err, "tx", tx)
				txm.txStore.OnErrored(tx)
				continue
			}

			txm.txStore.OnBroadcasted(tx, constructedTx)
		case <-txm.chStop:
			txm.lggr.Info("Stopping TronTxm")
			return
		}
	}
}

func (t *Txm) SignAndBroadcast(ctx context.Context, fromAddress string, tx *common.Transaction) (*fullnode.BroadcastResponse, error) {
	tx, err := t.signTx(ctx, tx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %+w", err)
	}

	return t.broadcastTx(tx)
}

func (t *Txm) ConstructTx(tx *pendingTx) (*common.Transaction, error) {
	// estimateEnergy will attempt multiple methods to estimate the energy used by the contract
	// if we can't estimate the energy, it's most likely a fatal error and we should stop the txn from executing
	energyUsed, err := t.energyEstimator.EstimateEnergy(tx.tx)
	if err != nil {
		return nil, err
	}

	paddedFeeLimit := CalculatePaddedFeeLimit(int32(energyUsed), tx.tx.EnergyBumpTimes, t.cfg.EnergyMultiplier)
	constructedTx, err := t.TriggerSmartContract(tx.tx, paddedFeeLimit)
	if err != nil {
		return nil, err
	}

	// If the transaction failed we'll set the tx as errored
	if !constructedTx.Result.Result {
		return nil, fmt.Errorf("transaction simulation failed!")
	}

	return constructedTx.Transaction, nil
}

func (t *Txm) TriggerSmartContract(tx *TronTx, feeLimit int32) (*fullnode.TriggerSmartContractResponse, error) {
	txExtention, err := t.GetClient().TriggerSmartContract(tx.FromAddress, tx.ContractAddress, tx.Method, tx.Params, feeLimit /* tAmount= (TRX amount) */, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to call TriggerSmartContract: %+w", err)
	}

	return txExtention, nil
}

func (t *Txm) signTx(ctx context.Context, tx *common.Transaction, fromAddress string) (*common.Transaction, error) {
	txIdBytes, err := hex.DecodeString(tx.TxID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode transaction id: %+w", err)
	}

	signature, err := t.ks.Sign(ctx, fromAddress, txIdBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %+w", err)
	}

	tx.AddSignatureBytes(signature)

	return tx, nil
}

func (t *Txm) broadcastTx(tx *common.Transaction) (*fullnode.BroadcastResponse, error) {
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
				t.lggr.Debugw("SERVER_BUSY or BLOCK_UNSOLIDIFIED: retry broadcast after timeout", "attempt", attempt)
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
