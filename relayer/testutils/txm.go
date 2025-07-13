package testutils

import (
	"context"
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"

	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-tron/relayer/txm"
)

func WaitForInflightTxs(logger logger.Logger, txmgr *txm.TronTxm, timeout time.Duration) {
	time.Sleep(5 * time.Second) // reduce flakiness
	start := time.Now()
	for {
		queueLen, unconfirmedLen := txmgr.InflightCount()
		logger.Debugw("Inflight count", "queued", queueLen, "unconfirmed", unconfirmedLen)
		if queueLen == 0 && unconfirmedLen == 0 {
			break
		}
		if time.Since(start) > timeout {
			panic("Timeout waiting for inflight txs")
		}
		time.Sleep(500 * time.Millisecond)
	}
}

func SignAndDeployContract(t *testing.T, fullnodeClient sdk.FullNodeClient, keystore loop.Keystore, fromAddress address.Address, contractName string, abiJson string, codeHex string, feeLimit int, params []interface{}) string {
	ctx := context.Background()
	deployResponse, err := fullnodeClient.DeployContract(ctx,
		fromAddress, contractName, abiJson, codeHex, 0, 100, feeLimit, params)
	require.NoError(t, err)

	tx := &deployResponse.Transaction
	txIdBytes, err := hex.DecodeString(tx.TxID)
	require.NoError(t, err)

	signature, err := keystore.Sign(context.Background(), fromAddress.String(), txIdBytes)
	require.NoError(t, err)
	tx.AddSignatureBytes(signature)

	broadcastResponse, err := fullnodeClient.BroadcastTransaction(ctx, tx)
	require.NoError(t, err)

	return broadcastResponse.TxID
}

func CheckContractDeployed(t *testing.T, fullnodeClient sdk.FullNodeClient, address address.Address) (contractDeployed bool) {
	ctx := context.Background()
	_, err := fullnodeClient.GetContract(ctx, address)
	require.NoError(t, err)

	return true // require call above stops execution if false
}

func WaitForTransactionInfo(t *testing.T, client sdk.FullNodeClient, txHash string, waitSecs int) *soliditynode.TransactionInfo {
	ctx := context.Background()
	for i := 1; i <= waitSecs; i++ {
		txInfo, err := client.GetTransactionInfoById(ctx, txHash)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		return txInfo
	}

	require.FailNow(t, fmt.Sprintf("failed to wait for transaction: %s", txHash))

	return nil
}
