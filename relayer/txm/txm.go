package txm

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
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

	tx := &TronTx{FromAddress: fromAddress, ContractAddress: contractAddress, Method: method, Params: encodedParams}

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
			txStore.AddUnconfirmed(txHash, coreTx.RawData.Timestamp, tx)

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

	// TODO: estimateEnergy is closed by default but more accurate, consider using that if possible.
	estimateTxExtention, err := t.client.TriggerConstantContract(tx.FromAddress, tx.ContractAddress, tx.Method, paramsJsonStr)
	if err != nil {
		return nil, fmt.Errorf("failed to call TriggerConstantContract: %+w", err)
	}

	// TODO: GetEnergyPrices returns history energy pricing data, but is not available in gotron-sdk.
	// It was recently added to the gRPC interface, see TIP-586.
	// ref: https://developers.tron.network/reference/getenergyprices
	// ref: https://github.com/tronprotocol/tips/blob/master/tip-586.md
	energyUnitPrice := int64(1000)

	feeLimit := energyUnitPrice * estimateTxExtention.EnergyUsed

	t.logger.Debugw("Estimated energy", "energyUsed", estimateTxExtention.EnergyUsed, "energyUnitPrice", energyUnitPrice, "feeLimit", feeLimit)

	txExtention, err := t.client.TriggerContract(
		tx.FromAddress,
		tx.ContractAddress,
		tx.Method,
		paramsJsonStr,
		feeLimit,
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
				continue
			}
			err = t.accountStore.GetTxStore(fromAddress).Confirm(unconfirmedTx.Hash)
			if err != nil {
				t.logger.Errorw("could not confirm transaction locally", "error", err)
				continue
			}
			t.logger.Debugw("confirmed transaction", "txHash", unconfirmedTx.Hash, "blockNumber", txInfo.BlockNumber)
		}
	}
}

func (t *TronTxm) InflightCount() (int, int) {
	return len(t.broadcastChan), t.accountStore.GetTotalInflightCount()
}
