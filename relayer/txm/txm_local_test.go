//go:build integration

package txm_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"

	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-tron/relayer/testutils"
	"github.com/smartcontractkit/chainlink-tron/relayer/txm"
)

func TestTxmLocal(t *testing.T) {
	logger := logger.Test(t)

	var genesisAddress address.Address
	var genesisPrivateKey *ecdsa.PrivateKey

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		genesisAccountKey := testutils.CreateKey(rand.Reader)
		genesisAddress = genesisAccountKey.Address
		genesisPrivateKey = genesisAccountKey.PrivateKey
	} else {
		privateKey, err := crypto.HexToECDSA(privateKeyHex)
		require.NoError(t, err)

		genesisAddress = address.PubkeyToAddress(privateKey.PublicKey)
		genesisPrivateKey = privateKey
	}
	logger.Debugw("Using genesis account", "address", genesisAddress.String())

	err := testutils.StartTronNode(genesisAddress.String())
	require.NoError(t, err)
	logger.Debugw("Started TRON node")

	keystore := testutils.NewTestKeystore(genesisAddress.String(), genesisPrivateKey)

	ipAddress := testutils.GetTronNodeIpAddress()
	rpcAddress := "http://" + ipAddress + ":" + testutils.FullNodePort + "/wallet"

	fullnodeUrl, err := url.ParseRequestURI(rpcAddress)
	require.NoError(t, err)
	solidityUrl, err := url.ParseRequestURI(rpcAddress + "solidity")
	require.NoError(t, err)

	combinedClient, err := sdk.CreateCombinedClient(fullnodeUrl, solidityUrl)
	require.NoError(t, err)

	config := txm.TronTxmConfig{
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
		RetentionPeriod:   10 * time.Second,
		ReapInterval:      1 * time.Second,
		// EnergyMultiplier is set to 1.5 by default
	}

	runTxmTest(t, logger, combinedClient, config, keystore, genesisAddress, 10)
}

func runTxmTest(t *testing.T, logger logger.Logger, combinedClient sdk.CombinedClient, config txm.TronTxmConfig, keystore loop.Keystore, fromAddress address.Address, iterations int) {
	txmgr := txm.New(logger, keystore, combinedClient, config)
	err := txmgr.Start(context.Background())
	require.NoError(t, err)

	contractAddress := deployTestContract(t, txmgr, fromAddress, keystore)
	logger.Debugw("Deployed test contract", "contractAddress", contractAddress)

	expectedValue := 0

	for i := 0; i < iterations; i++ {
		err = txmgr.Enqueue(txm.TronTxmRequest{
			FromAddress:     fromAddress,
			ContractAddress: contractAddress,
			Method:          "increment()",
			Params:          []any{},
		})
		require.NoError(t, err)
		expectedValue += 1

		err = txmgr.Enqueue(
			txm.TronTxmRequest{
				FromAddress:     fromAddress,
				ContractAddress: contractAddress,
				Method:          "increment_mult(uint256,uint256)",
				Params: []any{
					"uint256", "5",
					"uint256", "7",
				},
			},
		)
		require.NoError(t, err)
		expectedValue += 5 * 7
	}

	testutils.WaitForInflightTxs(logger, txmgr, 30*time.Second)

	// not strictly necessary, but docs note: "For constant call you can use the all-zero address."
	txExtention, err := txmgr.GetClient().TriggerConstantContractFullNode(address.ZeroAddress, contractAddress, "count()", nil)
	require.NoError(t, err)

	constantResult := txExtention.ConstantResult
	require.Equal(t, len(constantResult), 1)

	actualValue, err := strconv.ParseInt(constantResult[0], 16, 32)
	require.NoError(t, err)
	logger.Debugw("Read count value", "countStr", constantResult[0], "count", actualValue, "expected", expectedValue)

	require.Equal(t, int64(expectedValue), actualValue)
}

func getTestCounterContract() (string, string) {
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

	return abiJson, codeHex
}

func deployTestContract(t *testing.T, txmgr *txm.TronTxm, fromAddress address.Address, keystore loop.Keystore) address.Address {
	abiJson, codeHex := getTestCounterContract()
	client := txmgr.GetClient()
	txHash := testutils.SignAndDeployContract(t, client, keystore, fromAddress, "Counter", abiJson, codeHex, testutils.DevnetFeeLimit, nil)
	txInfo := testutils.WaitForTransactionInfo(t, client, txHash, 30)
	contractAddress, err := address.HexToAddress(txInfo.ContractAddress)
	require.NoError(t, err)
	return contractAddress
}
