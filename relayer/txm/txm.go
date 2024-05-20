package txm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/proto"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-common/pkg/utils"
)

var _ services.Service = &TronTxm{}

const DEFAULT_ENERGY_UNIT_PRICE int64 = 420

type TronTxm struct {
	logger   logger.Logger
	keystore loop.Keystore
	config   TronTxmConfig

	client        *client.GrpcClient
	broadcastChan chan *TronTx
	accountStore  *AccountStore
	starter       utils.StartStopOnce
	done          sync.WaitGroup
	stop          chan struct{}
}

func New(lgr logger.Logger, keystore loop.Keystore, config TronTxmConfig) *TronTxm {
	return &TronTxm{
		logger:   logger.Named(lgr, "TronTxm"),
		keystore: keystore,
		config:   config,

		client:        client.NewGrpcClientWithTimeout(config.RPCAddress, 15*time.Second),
		broadcastChan: make(chan *TronTx, config.BroadcastChanSize),
		accountStore:  newAccountStore(),
		stop:          make(chan struct{}),
	}
}

func (t *TronTxm) Name() string {
	return t.logger.Name()
}

func (t *TronTxm) Ready() error {
	return t.starter.Ready()
}

func (t *TronTxm) HealthReport() map[string]error {
	return map[string]error{t.Name(): t.starter.Healthy()}
}

func (t *TronTxm) GetClient() *client.GrpcClient {
	return t.client
}

func (t *TronTxm) Start(ctx context.Context) error {
	return t.starter.StartOnce("TronTxm", func() error {
		var transportCredentials credentials.TransportCredentials
		if t.config.RPCInsecure {
			transportCredentials = insecure.NewCredentials()
		} else {
			transportCredentials = credentials.NewTLS(nil)
		}
		err := t.client.Start(grpc.WithTransportCredentials(transportCredentials))
		if err != nil {
			return fmt.Errorf("failed to start GrpcClient: %+w", err)
		}
		t.done.Add(2) // waitgroup: broadcast loop and confirm loop
		go t.broadcastLoop()
		go t.confirmLoop()
		return nil
	})
}

func (t *TronTxm) Close() error {
	return t.starter.StopOnce("TronTxm", func() error {
		close(t.stop)
		t.done.Wait()
		t.client.Stop()
		return nil
	})
}

// Enqueues a transaction for broadcasting.
// Each item in the params array should be a map with a single key-value pair, where
// the key is the ABI type.
func (t *TronTxm) Enqueue(fromAddress, contractAddress, method string, params ...string) error {
	if _, err := t.keystore.Sign(context.Background(), fromAddress, nil); err != nil {
		return fmt.Errorf("failed to sign: %+w", err)
	}

	encodedParams := make([]map[string]string, 0)
	if len(params)%2 == 1 {
		return fmt.Errorf("odd number of params")
	}
	for i := 0; i < len(params); i += 2 {
		encodedParams = append(encodedParams, map[string]string{params[i]: params[i+1]})
	}

	tx := &TronTx{FromAddress: fromAddress, ContractAddress: contractAddress, Method: method, Params: encodedParams, Attempt: 1}

	select {
	case t.broadcastChan <- tx:
	default:
		return fmt.Errorf("failed to enqueue transaction: %+v", tx)
	}

	return nil
}

func (t *TronTxm) broadcastLoop() {
	defer t.done.Done()

	ctx, cancel := utils.ContextFromChan(t.stop)
	defer cancel()

	t.logger.Debugw("broadcastLoop: started")
	for {
		select {
		case tx := <-t.broadcastChan:
			txExtention, err := t.TriggerSmartContract(ctx, tx)
			if err != nil {
				t.logger.Errorw("failed to trigger smart contract", "error", err, "tx", tx)
				continue
			}

			txHash := common.BytesToHexString(txExtention.Txid)

			coreTx := txExtention.Transaction
			// RefBlockNum is optional and does not seem in use anymore.
			t.logger.Debugw("created transaction", "txHash", txHash, "timestamp", coreTx.RawData.Timestamp, "expiration", coreTx.RawData.Expiration, "refBlockHash", common.BytesToHexString(coreTx.RawData.RefBlockHash), "feeLimit", coreTx.RawData.FeeLimit)

			_, err = t.SignAndBroadcast(ctx, tx.FromAddress, txExtention)
			if err != nil {
				t.logger.Errorw("transaction failed to broadcast", "txHash", txHash, "error", err, "tx", tx, "txExtention", txExtention)
				continue
			}

			t.logger.Infow("transaction broadcasted", "txHash", txHash)

			txStore := t.accountStore.GetTxStore(tx.FromAddress)
			txStore.AddUnconfirmed(txHash, time.Now().Unix(), tx)

		case <-t.stop:
			t.logger.Debugw("broadcastLoop: stopped")
			return
		}
	}
}

func (t *TronTxm) TriggerSmartContract(ctx context.Context, tx *TronTx) (*api.TransactionExtention, error) {
	// TODO: consider calling GrpcClient.Client.TriggerContract directly to avoid
	// the extra marshal/unmarshal steps.
	paramsJsonBytes, err := json.Marshal(tx.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal params: %+w", err)
	}

	paramsJsonStr := string(paramsJsonBytes)

	var energyUsed int64

	if t.config.EnableEstimateEnergy {
		estimateEnergyMessage, err := t.client.EstimateEnergy(
			tx.FromAddress,
			tx.ContractAddress,
			tx.Method,
			paramsJsonStr,
			/* tAmount= */ 0,
			/* tTokenID= */ "",
			/* tTokenAmount= */ 0,
		)
		if err == nil {
			energyUsed = estimateEnergyMessage.EnergyRequired
			t.logger.Debugw("Estimated energy (EE)", "energyRequired", estimateEnergyMessage.EnergyRequired)
		} else {
			// Falls back to TriggerConstantContract estimation.
			t.logger.Errorw("failed to call EstimateEnergy", "err", err)
		}
	}

	if energyUsed == 0 {
		estimateTxExtention, err := t.client.TriggerConstantContract(tx.FromAddress, tx.ContractAddress, tx.Method, paramsJsonStr)
		if err != nil {
			return nil, fmt.Errorf("failed to call TriggerConstantContract: %+w", err)
		}
		energyUsed = estimateTxExtention.EnergyUsed

		t.logger.Debugw("Estimated energy (TCC)", "energyUsed", estimateTxExtention.EnergyUsed, "energyPenalty", estimateTxExtention.EnergyPenalty)
	}

	energyUnitPrice := DEFAULT_ENERGY_UNIT_PRICE

	if energyPrices, err := t.client.GetEnergyPrices(); err == nil {
		if parsedPrice, err := parseLatestEnergyPrice(energyPrices.Prices); err == nil {
			energyUnitPrice = parsedPrice
		} else {
			t.logger.Errorw("error parsing energy unit price", "error", err)
		}
	} else {
		t.logger.Errorw("failed to get energy unit price", "error", err)
	}

	feeLimit := energyUnitPrice * energyUsed
	paddedFeeLimit := int64(float64(feeLimit) * math.Pow(1.5, float64(tx.Attempt)))

	t.logger.Debugw("Trigger contract", "attempt", tx.Attempt, "energyUnitPrice", energyUnitPrice, "feeLimit", feeLimit, "paddedFeeLimit", paddedFeeLimit)

	txExtention, err := t.client.TriggerContract(
		tx.FromAddress,
		tx.ContractAddress,
		tx.Method,
		paramsJsonStr,
		paddedFeeLimit,
		/* tAmount= (TRX amount) */ 0,
		/* tTokenID= (TRC10 token id) */ "",
		/* tTokenAmount= (TRC10 token amount) */ 0)

	if err != nil {
		return nil, fmt.Errorf("failed to call TriggerContract: %+w", err)
	}

	return txExtention, nil
}

func (t *TronTxm) SignAndBroadcast(ctx context.Context, fromAddress string, txExtention *api.TransactionExtention) (*api.Return, error) {
	coreTx := txExtention.Transaction

	// ref: https://github.com/fbsobreira/gotron-sdk/blob/1e824406fe8ce02f2fec4c96629d122560a3598f/pkg/keystore/keystore.go#L273
	rawData, err := proto.Marshal(coreTx.GetRawData())
	if err != nil {
		return nil, fmt.Errorf("failed to marshall transaction data: %+w", err)
	}

	h256h := sha256.New()
	h256h.Write(rawData)
	hash := h256h.Sum(nil)

	signature, err := t.keystore.Sign(ctx, fromAddress, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %+w", err)
	}

	coreTx.Signature = append(coreTx.Signature, signature)

	// the *api.Return error message and code is embedded in err.
	apiReturn, err := t.client.Broadcast(coreTx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %+w", err)
	}

	return apiReturn, nil
}

func (t *TronTxm) confirmLoop() {
	defer t.done.Done()

	_, cancel := utils.ContextFromChan(t.stop)
	defer cancel()

	tick := time.After(time.Duration(t.config.ConfirmPollSecs) * time.Second)

	t.logger.Debugw("confirmLoop: started")

	for {
		select {
		case <-tick:
			start := time.Now()

			t.checkUnconfirmed()

			remaining := time.Duration(t.config.ConfirmPollSecs) - time.Since(start)
			tick = time.After(utils.WithJitter(remaining.Abs()))

		case <-t.stop:
			t.logger.Debugw("confirmLoop: stopped")
			return
		}
	}
}

func (t *TronTxm) checkUnconfirmed() {
	allUnconfirmedTxs := t.accountStore.GetAllUnconfirmed()
	for fromAddress, unconfirmedTxs := range allUnconfirmedTxs {
		for _, unconfirmedTx := range unconfirmedTxs {
			txInfo, err := t.client.GetTransactionInfoByID(unconfirmedTx.Hash)
			if err != nil {
				// the default transaction expiration time is 60 seconds - if we still can't find the hash,
				// rebroadcast since the transaction has expired.
				if (time.Now().Unix() - unconfirmedTx.Timestamp) > 150 {
					err = t.accountStore.GetTxStore(fromAddress).Confirm(unconfirmedTx.Hash)
					if err != nil {
						t.logger.Errorw("could not confirm expired transaction locally", "error", err)
						continue
					}
					t.logger.Debugw("transaction missing after expiry", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash)
					t.maybeRetry(unconfirmedTx, false, false)
				}
				continue
			}
			err = t.accountStore.GetTxStore(fromAddress).Confirm(unconfirmedTx.Hash)
			if err != nil {
				t.logger.Errorw("could not confirm transaction locally", "error", err)
				continue
			}
			receipt := txInfo.Receipt
			if receipt == nil {
				t.logger.Errorw("could not read transaction receipt", "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				continue
			}
			contractResult := receipt.Result
			switch contractResult {
			case core.Transaction_Result_OUT_OF_ENERGY:
				t.logger.Debugw("transaction failed due to out of energy", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, true, false)
				continue
			case core.Transaction_Result_OUT_OF_TIME:
				t.logger.Debugw("transaction failed due to out of time", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, false, true)
				continue
			case core.Transaction_Result_UNKNOWN:
				t.logger.Debugw("transaction failed due to unknown error", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, false, false)
				continue
			}
			t.logger.Debugw("confirmed transaction", "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "contractResult", contractResult)
		}
	}
}

func (t *TronTxm) maybeRetry(unconfirmedTx *UnconfirmedTx, bumpEnergy bool, isOutOfTimeError bool) {
	tx := unconfirmedTx.Tx
	if tx.Attempt >= 5 {
		t.logger.Debugw("not retrying, already reached max retries", "txHash", unconfirmedTx.Hash)
		return
	}
	if tx.OutOfTimeErrors >= 2 {
		t.logger.Debugw("not retrying, multiple OUT_OF_TIME errors", "txHash", unconfirmedTx.Hash)
		return
	}

	newTx := &*tx
	newTx.Attempt += 1
	if bumpEnergy {
		newTx.EnergyBumpTimes += 1
	}
	if isOutOfTimeError {
		newTx.OutOfTimeErrors += 1
	}

	t.logger.Infow("retrying transaction", "previousTxHash", unconfirmedTx.Hash, "attempt", newTx.Attempt)

	select {
	case t.broadcastChan <- newTx:
	default:
		t.logger.Errorw("failed to enqueue retry transaction", "previousTxHash", unconfirmedTx.Hash)
	}
}

func (t *TronTxm) InflightCount() (int, int) {
	return len(t.broadcastChan), t.accountStore.GetTotalInflightCount()
}
