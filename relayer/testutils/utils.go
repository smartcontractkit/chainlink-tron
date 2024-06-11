package testutils

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/pborman/uuid"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
)

func Int64Ptr(i int64) *int64 {
	return &i
}

type TestKeystore struct {
	Keys map[string]*ecdsa.PrivateKey
}

var _ loop.Keystore = &TestKeystore{}

func NewTestKeystore(address string, privateKey *ecdsa.PrivateKey) *TestKeystore {
	// TODO: we don't actually need a map if we only have a single key pair.
	keys := map[string]*ecdsa.PrivateKey{}
	keys[address] = privateKey
	return &TestKeystore{Keys: keys}
}

func (tk *TestKeystore) Sign(ctx context.Context, id string, hash []byte) ([]byte, error) {
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

func (tk *TestKeystore) Accounts(ctx context.Context) ([]string, error) {
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
func CreateKey(rand io.Reader) *keystore.Key {
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
	key := NewKeyFromECDSA(privateKeyECDSA)
	return key
}

func NewKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *keystore.Key {
	id := uuid.NewRandom()
	key := &keystore.Key{
		ID:         id,
		Address:    address.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
	}
	return key
}

// Finds the closest git repo root, assuming that a directory with a .git directory is a git repo.
func FindGitRoot() (string, error) {
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

func StartTronNode(genesisAddress string) error {
	gitRoot, err := FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find Git root: %v", err)
	}

	scriptPath := filepath.Join(gitRoot, "tron/scripts/java-tron.sh")
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
