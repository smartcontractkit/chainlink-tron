package testutils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
)

func WaitForInflightTxs(logger logger.Logger, txmgr *txm.TronTxm, timeout time.Duration) {
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

func DeployContract(t *testing.T, txmgr *txm.TronTxm, fromAddress string, contractName string, abiJson string, codeHex string) string {
	abi, err := contract.JSONtoABI(abiJson)
	require.NoError(t, err)

	txExtention, err := txmgr.GetClient().DeployContract(
		fromAddress,
		contractName,
		abi,
		codeHex,
		/* feeLimit= */ 1000000000,
		/* curPercent= */ 100,
		/* oeLimit= */ 10000000)
	require.NoError(t, err)

	_, err = txmgr.SignAndBroadcast(context.Background(), fromAddress, txExtention)
	require.NoError(t, err)

	txHash := common.BytesToHexString(txExtention.Txid)
	return txHash
}

func WaitForTransactionInfo(t *testing.T, grpcClient sdk.GrpcClient, txHash string, waitSecs int) *core.TransactionInfo {
	for i := 1; i <= waitSecs; i++ {
		txInfo, err := grpcClient.GetTransactionInfoByID(txHash)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		return txInfo
	}

	require.FailNow(t, fmt.Sprintf("failed to wait for transaction: %s", txHash))

	return nil
}
