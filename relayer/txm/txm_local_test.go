//go:build integration

package txm

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
	"github.com/stretchr/testify/require"
)

func TestTxmLocal(t *testing.T) {
	logger := logger.Test(t)

	var genesisAddress string
	var genesisPrivateKey *ecdsa.PrivateKey

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		genesisAccountKey := testutils.CreateKey(rand.Reader)
		genesisAddress = genesisAccountKey.Address.String()
		genesisPrivateKey = genesisAccountKey.PrivateKey
	} else {
		privateKey, err := crypto.HexToECDSA(privateKeyHex)
		require.NoError(t, err)

		genesisAddress = address.PubkeyToAddress(privateKey.PublicKey).String()
		genesisPrivateKey = privateKey
	}
	logger.Debugw("Using genesis account", "address", genesisAddress)

	err := testutils.StartTronNode(genesisAddress)
	require.NoError(t, err)
	logger.Debugw("Started TRON node")

	keystore := testutils.NewTestKeystore(genesisAddress, genesisPrivateKey)

	// TODO: can be refactored to test utils in the future as integration tests needs this as well
	var rpcAddress string

	if runtime.GOOS == "darwin" {
		rpcAddress = "127.0.0.1:16669" // Mac OS needs local host port forwarding for docker
	} else {
		rpcAddress = "172.255.0.101:16669" // Linux does not need port forwarding
	}

	config := TronTxmConfig{
		RPCAddress:        rpcAddress,
		RPCInsecure:       true,
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
	}

	runTxmTest(t, logger, config, keystore, genesisAddress, 10)
}

func runTxmTest(t *testing.T, logger logger.Logger, config TronTxmConfig, keystore loop.Keystore, fromAddress string, iterations int) {
	txm := New(logger, keystore, config)
	err := txm.Start(context.Background())
	require.NoError(t, err)

	contractAddress := deployTestContract(t, txm, fromAddress)
	logger.Debugw("Deployed test contract", "contractAddress", contractAddress)

	expectedValue := 0

	for i := 0; i < iterations; i++ {
		err = txm.Enqueue(fromAddress, contractAddress, "increment()")
		require.NoError(t, err)
		expectedValue += 1

		err = txm.Enqueue(fromAddress, contractAddress,
			"increment_mult(uint256,uint256)",
			"uint256", "5",
			"uint256", "7",
		)
		require.NoError(t, err)
		expectedValue += 5 * 7
	}

	WaitForInflightTxs(logger, txm, 30*time.Second)

	// not strictly necessary, but docs note: "For constant call you can use the all-zero address."
	// this address maps to 0x410000000000000000000000000000000000000000 where 0x41 is the TRON address
	// prefix.
	zeroAddress := "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"
	txExtention, err := txm.client.TriggerConstantContract(zeroAddress, contractAddress, "count()", "")
	require.NoError(t, err)

	constantResult := txExtention.ConstantResult
	require.Equal(t, len(constantResult), 1)

	actualValueStr := common.BytesToHexString(constantResult[0])
	actualValue, err := strconv.ParseInt(actualValueStr[2:], 16, 32)
	require.NoError(t, err)
	logger.Debugw("Read count value", "countStr", actualValueStr, "count", actualValue, "expected", expectedValue)

	require.Equal(t, int64(expectedValue), actualValue)
}

func deployTestContract(t *testing.T, txm *TronTxm, fromAddress string) string {
	// small test counter contract:
	//
	//  contract Counter {
	//    uint256 public count = 0;
	//
	//    function increment() public returns (uint256) {
	//        count += 1;
	//        return count;
	//    }
	//    function increment_mult(a uint256, b uint256) public returns (uint256) {
	//        count += a * b;
	//        return count;
	//    }
	//  }

	abiJson := "[{\"inputs\":[],\"name\":\"count\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"view\",\"type\":\"function\"},{\"inputs\":[],\"name\":\"increment\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"},{\"inputs\":[{\"internalType\":\"uint256\",\"name\":\"a\",\"type\":\"uint256\"},{\"internalType\":\"uint256\",\"name\":\"b\",\"type\":\"uint256\"}],\"name\":\"increment_mult\",\"outputs\":[{\"internalType\":\"uint256\",\"name\":\"\",\"type\":\"uint256\"}],\"stateMutability\":\"nonpayable\",\"type\":\"function\"}]"

	codeHex := "60806040526000805534801561001457600080fd5b5061016c80610024600039" +
		"6000f3fe608060405234801561001057600080fd5b5060043610610040576000" +
		"3560e01c8062bf70861461004557806306661abd1461006a578063d09de08a14" +
		"610073575b600080fd5b6100586100533660046100c7565b61007b565b604051" +
		"90815260200160405180910390f35b61005860005481565b6100586100a6565b" +
		"60006100878284610101565b60008082825461009791906100e9565b90915550" +
		"506000549392505050565b600060016000808282546100ba91906100e9565b90" +
		"91555050600054919050565b600080604083850312156100da57600080fd5b50" +
		"508035926020909101359150565b600082198211156100fc576100fc61012056" +
		"5b500190565b600081600019048311821515161561011b5761011b610120565b" +
		"500290565b634e487b7160e01b600052601160045260246000fdfea264697066" +
		"73582212209b5ec6726bb13377d7e7824aaf14b6e31224ee82dc6a3062bc4cf9" +
		"881233197264736f6c63430008070033"

	abi, err := contract.JSONtoABI(abiJson)
	require.NoError(t, err)

	txExtention, err := txm.GetClient().DeployContract(
		fromAddress,
		"Counter",
		abi,
		codeHex,
		/* feeLimit= */ 1000000000,
		/* curPercent= */ 100,
		/* oeLimit= */ 10000000)
	require.NoError(t, err)

	_, err = txm.SignAndBroadcast(context.Background(), fromAddress, txExtention)
	require.NoError(t, err)

	txHash := common.BytesToHexString(txExtention.Txid)

	txInfo := waitForTransactionInfo(t, txm, txHash, 30)
	contractAddress := address.Address(txInfo.ContractAddress).String()
	return contractAddress
}

func waitForTransactionInfo(t *testing.T, txm *TronTxm, txHash string, waitSecs int) *core.TransactionInfo {
	for i := 1; i <= waitSecs; i++ {
		txInfo, err := txm.GetClient().GetTransactionInfoByID(txHash)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		return txInfo
	}

	require.FailNow(t, fmt.Sprintf("failed to wait for transaction: %s", txHash))

	return nil
}
