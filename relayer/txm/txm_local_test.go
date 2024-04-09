//go:build integration

package txm

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/common"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/pborman/uuid"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	"github.com/stretchr/testify/require"
)

func TestTxmLocal(t *testing.T) {
	logger := logger.Test(t)

	var genesisAddress string
	var genesisPrivateKey *ecdsa.PrivateKey

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		genesisAccountKey := createKey(rand.Reader)
		genesisAddress = genesisAccountKey.Address.String()
		genesisPrivateKey = genesisAccountKey.PrivateKey
	} else {
		privateKey, err := crypto.HexToECDSA(privateKeyHex)
		require.NoError(t, err)

		genesisAddress = address.PubkeyToAddress(privateKey.PublicKey).String()
		genesisPrivateKey = privateKey
	}
	logger.Debugw("Using genesis account", "address", genesisAddress)

	err := startTronNode(genesisAddress)
	require.NoError(t, err)
	logger.Debugw("Started TRON node")

	keystore := newTestKeystore(genesisAddress, genesisPrivateKey)

	config := TronTxmConfig{
		RPCAddress:        "127.0.0.1:16669",
		RPCInsecure:       true,
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
	}

	runTxmTest(t, logger, config, keystore, genesisAddress, 10)
}

func int64Ptr(i int64) *int64 {
	return &i
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

	for {
		queueLen, unconfirmedLen := txm.InflightCount()
		logger.Debugw("Inflight count", "queued", queueLen, "unconfirmed", unconfirmedLen)
		if queueLen == 0 && unconfirmedLen == 0 {
			break
		}
		time.Sleep(500 * time.Millisecond)
	}

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

	txExtention, err := txm.client.DeployContract(
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
		txInfo, err := txm.client.GetTransactionInfoByID(txHash)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		return txInfo
	}

	require.FailNow(t, fmt.Sprintf("failed to wait for transaction: %s", txHash))

	return nil
}

type testKeystore struct {
	Keys map[string]*ecdsa.PrivateKey
}

var _ loop.Keystore = &testKeystore{}

func newTestKeystore(address string, privateKey *ecdsa.PrivateKey) *testKeystore {
	// TODO: we don't actually need a map if we only have a single key pair.
	keys := map[string]*ecdsa.PrivateKey{}
	keys[address] = privateKey
	return &testKeystore{Keys: keys}
}

func (tk *testKeystore) Sign(ctx context.Context, id string, hash []byte) ([]byte, error) {
	privateKey, ok := tk.Keys[id]
	if !ok {
		return nil, fmt.Errorf("no such key")
	}

	// used to check if the account exists.
	if hash == nil {
		return nil, nil
	}

	return crypto.Sign(hash, privateKey)
}

func (tk *testKeystore) Accounts(ctx context.Context) ([]string, error) {
	accounts := make([]string, 0, len(tk.Keys))
	for id := range tk.Keys {
		accounts = append(accounts, id)
	}
	return accounts, nil
}

// this is copied from keystore.NewKeyFromDirectICAP, which keeps trying to
// recreate the key if it doesn't start with a 0 prefix and can take significantly longer.
// the function we need is keystore.newKey which is unfortunately private.
// ref: https://github.com/fbsobreira/gotron-sdk/blob/1e824406fe8ce02f2fec4c96629d122560a3598f/pkg/keystore/key.go#L146
func createKey(rand io.Reader) *keystore.Key {
	randBytes := make([]byte, 64)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic("key generation: could not read from random source: " + err.Error())
	}
	reader := bytes.NewReader(randBytes)
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}
	key := newKeyFromECDSA(privateKeyECDSA)
	return key
}

func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *keystore.Key {
	id := uuid.NewRandom()
	key := &keystore.Key{
		ID:         id,
		Address:    address.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
	}
	return key
}

// Finds the closest git repo root, assuming that a directory with a .git directory is a git repo.
func findGitRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(currentDir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("no Git repository found")
		}

		currentDir = parentDir
	}
}

func startTronNode(genesisAddress string) error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find Git root: %v", err)
	}

	scriptPath := filepath.Join(gitRoot, "scripts/java-tron.sh")
	cmd := exec.Command(scriptPath, genesisAddress)

	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Printf("Failed to start java-tron, dumping output:\n%s\n", string(output))
			return fmt.Errorf("Failed to start java-tron, bad exit code: %v", exitError.ExitCode())
		}
		return fmt.Errorf("Failed to start java-tron: %+v", err)
	}

	return nil
}
