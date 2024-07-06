package txm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"

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

	Client        sdk.GrpcClient
	BroadcastChan chan *TronTx
	AccountStore  *AccountStore
	Starter       utils.StartStopOnce
	Done          sync.WaitGroup
	Stop          chan struct{}
}

func New(lgr logger.Logger, keystore loop.Keystore, client sdk.GrpcClient, config TronTxmConfig) *TronTxm {
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

func (t *TronTxm) GetClient() sdk.GrpcClient {
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
		t.GetClient().Stop()
		return nil
	})
}

// Enqueues a transaction for broadcasting.
// Each item in the params array should be a map with a single key-value pair, where
// the key is the ABI type.
func (t *TronTxm) Enqueue(fromAddress, contractAddress, method string, params ...any) error {
	if _, err := t.Keystore.Sign(context.Background(), fromAddress, nil); err != nil {
		return fmt.Errorf("failed to sign: %+w", err)
	}

	encodedParams := make([]map[string]any, 0)
	if len(params)%2 == 1 {
		return fmt.Errorf("odd number of params")
	}
	for i := 0; i < len(params); i += 2 {
		paramType := params[i]
		paramTypeStr, ok := paramType.(string)
		if !ok {
			return fmt.Errorf("non-string param type")
		}
		encodedParams = append(encodedParams, map[string]any{paramTypeStr: params[i+1]})
	}

	tx := &TronTx{FromAddress: fromAddress, ContractAddress: contractAddress, Method: method, Params: encodedParams, Attempt: 1}

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
			txExtention, err := t.TriggerSmartContract(ctx, tx)
			if err != nil {
				t.Logger.Errorw("failed to trigger smart contract", "error", err, "tx", tx)
				continue
			}

			txHash := common.BytesToHexString(txExtention.Txid)

			coreTx := txExtention.Transaction
			// RefBlockNum is optional and does not seem in use anymore.
			t.Logger.Debugw("created transaction", "txHash", txHash, "timestamp", coreTx.RawData.Timestamp, "expiration", coreTx.RawData.Expiration, "refBlockHash", common.BytesToHexString(coreTx.RawData.RefBlockHash), "feeLimit", coreTx.RawData.FeeLimit)

			_, err = t.SignAndBroadcast(ctx, tx.FromAddress, txExtention)
			if err != nil {
				t.Logger.Errorw("transaction failed to broadcast", "txHash", txHash, "error", err, "tx", tx, "txExtention", txExtention)
				continue
			}

			t.Logger.Infow("transaction broadcasted", "txHash", txHash)

			txStore := t.AccountStore.GetTxStore(tx.FromAddress)
			txStore.AddUnconfirmed(txHash, time.Now().Unix(), tx)

		case <-t.Stop:
			t.Logger.Debugw("broadcastLoop: stopped")
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

	energyUsed, err := t.estimateEnergy(tx, paramsJsonStr)
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

	feeLimit := energyUnitPrice * energyUsed
	paddedFeeLimit := CalculatePaddedFeeLimit(feeLimit, tx.EnergyBumpTimes)

	t.Logger.Debugw("Trigger contract", "Energy Bump Times", tx.EnergyBumpTimes, "energyUnitPrice", energyUnitPrice, "feeLimit", feeLimit, "paddedFeeLimit", paddedFeeLimit)

	txExtention, err := t.GetClient().TriggerContract(
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

	signature, err := t.Keystore.Sign(ctx, fromAddress, hash)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %+w", err)
	}

	coreTx.Signature = append(coreTx.Signature, signature)

	// the *api.Return error message and code is embedded in err.
	apiReturn, err := t.broadcastTx(coreTx)
	if err != nil {
		return nil, fmt.Errorf("failed to broadcast transaction: %+w", err)
	}

	return apiReturn, nil
}

func (t *TronTxm) broadcastTx(tx *core.Transaction) (*api.Return, error) {
	var apiReturn *api.Return
	var err error
	startTime := time.Now()
	attempt := 1
	for time.Since(startTime) < MAX_BROADCAST_RETRY_DURATION {
		apiReturn, err = t.GetClient().Broadcast(tx)
		if err == nil {
			break
		}

		// err != nil, check response code
		resCode := apiReturn.GetCode()
		if resCode == api.Return_SERVER_BUSY || resCode == api.Return_BLOCK_UNSOLIDIFIED {
			// wait and retry tx broadcast upon SERVER_BUSY and BLOCK_UNSOLIDIFIED error responses
			t.Logger.Debugw("SERVER_BUSY or BLOCK_UNSOLIDIFIED: retry broadcast after timeout", "attempt", attempt)
			time.Sleep(BROADCAST_DELAY_DURATION)
			attempt = attempt + 1
			continue
		} else {
			// do not retry on other broadcast errors
			return nil, err
		}
	}
	if err != nil {
		return nil, fmt.Errorf("SERVER_BUSY or BLOCK_UNSOLIDIFIED: max retry duration reached, error: %w", err)
	}
	return apiReturn, nil
}

func (t *TronTxm) confirmLoop() {
	defer t.Done.Done()

	_, cancel := utils.ContextFromChan(t.Stop)
	defer cancel()

	tick := time.After(time.Duration(t.Config.ConfirmPollSecs) * time.Second)

	t.Logger.Debugw("confirmLoop: started")

	for {
		select {
		case <-tick:
			start := time.Now()

			t.checkUnconfirmed()

			remaining := time.Duration(t.Config.ConfirmPollSecs) - time.Since(start)
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
		for _, unconfirmedTx := range unconfirmedTxs {
			txInfo, err := t.GetClient().GetTransactionInfoByID(unconfirmedTx.Hash)
			if err != nil {
				// the default transaction expiration time is 60 seconds - if we still can't find the hash,
				// rebroadcast since the transaction has expired.
				if (time.Now().Unix() - unconfirmedTx.Timestamp) > 150 {
					err = t.AccountStore.GetTxStore(fromAddress).Confirm(unconfirmedTx.Hash)
					if err != nil {
						t.Logger.Errorw("could not confirm expired transaction locally", "error", err)
						continue
					}
					t.Logger.Debugw("transaction missing after expiry", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash)
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
			if receipt == nil {
				t.Logger.Errorw("could not read transaction receipt", "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				continue
			}
			contractResult := receipt.Result
			switch contractResult {
			case core.Transaction_Result_OUT_OF_ENERGY:
				t.Logger.Debugw("transaction failed due to out of energy", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, true, false)
				continue
			case core.Transaction_Result_OUT_OF_TIME:
				t.Logger.Debugw("transaction failed due to out of time", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, false, true)
				continue
			case core.Transaction_Result_UNKNOWN:
				t.Logger.Debugw("transaction failed due to unknown error", "attempt", unconfirmedTx.Tx.Attempt, "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
				t.maybeRetry(unconfirmedTx, false, false)
				continue
			}
			t.Logger.Debugw("confirmed transaction", "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber, "contractResult", contractResult)
		}
	}
}

func (t *TronTxm) maybeRetry(unconfirmedTx *UnconfirmedTx, bumpEnergy bool, isOutOfTimeError bool) {
	tx := unconfirmedTx.Tx
	if tx.Attempt >= MAX_RETRY_ATTEMPTS {
		t.Logger.Debugw("not retrying, already reached max retries", "txHash", unconfirmedTx.Hash)
		return
	}
	if tx.OutOfTimeErrors >= 2 {
		t.Logger.Debugw("not retrying, multiple OUT_OF_TIME errors", "txHash", unconfirmedTx.Hash)
		return
	}

	tx.Attempt += 1
	if bumpEnergy {
		tx.EnergyBumpTimes += 1
	}
	if isOutOfTimeError {
		tx.OutOfTimeErrors += 1
	}

	t.Logger.Infow("retrying transaction", "previousTxHash", unconfirmedTx.Hash, "attempt", tx.Attempt)

	select {
	case t.BroadcastChan <- tx:
	default:
		t.Logger.Errorw("failed to enqueue retry transaction", "previousTxHash", unconfirmedTx.Hash)
	}
}

func (t *TronTxm) InflightCount() (int, int) {
	return len(t.BroadcastChan), t.AccountStore.GetTotalInflightCount()
}

func (t *TronTxm) estimateEnergy(tx *TronTx, paramsJsonStr string) (int64, error) {

	if t.EstimateEnergyEnabled {
		estimateEnergyMessage, err := t.GetClient().EstimateEnergy(
			tx.FromAddress,
			tx.ContractAddress,
			tx.Method,
			paramsJsonStr,
			/* tAmount= */ 0,
			/* tTokenID= */ "",
			/* tTokenAmount= */ 0,
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
	estimateTxExtention, err := t.GetClient().TriggerConstantContract(tx.FromAddress, tx.ContractAddress, tx.Method, paramsJsonStr)

	if err != nil {
		return 0, fmt.Errorf("failed to call TriggerConstantContract: %w", err)
	}
	if estimateTxExtention.Result.Code > 0 {
		return 0, fmt.Errorf("failed to call TriggerConstantContract due to %s", string(estimateTxExtention.Result.Message))
	}

	t.Logger.Debugw("Estimated energy using TriggerConstantContract Method", "energyUsed", estimateTxExtention.EnergyUsed, "energyPenalty", estimateTxExtention.EnergyPenalty, "tx", tx)

	return estimateTxExtention.EnergyUsed, nil
}
