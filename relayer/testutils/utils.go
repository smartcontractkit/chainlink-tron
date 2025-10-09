package testutils

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/google/uuid"
	"github.com/smartcontractkit/chainlink-tron/relayer/gotron-sdk/pkg/address"
)

// this is copied from keystore.NewKeyFromDirectICAP, which keeps trying to
// recreate the key if it doesn't start with a 0 prefix and can take significantly longer.
// the function we need is keystore.newKey which is unfortunately private.
// ref: https://github.com/smartcontractkit/chainlink-tron/relayer/gotron-sdk/blob/1e824406fe8ce02f2fec4c96629d122560a3598f/pkg/keystore/key.go#L146

type TestKey struct {
	ID uuid.UUID // Version 4 "random" for unique id not derived from key data
	// to simplify lookups we also store the address
	Address address.Address
	// we only store privkey as pubkey/address can be derived from it
	// privkey in this struct is always in plaintext
	PrivateKey *ecdsa.PrivateKey
}

func CreateKey(rand io.Reader) *TestKey {
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

func NewKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *TestKey {
	id, err := uuid.NewRandom()
	if err != nil {
		panic(err)
	}
	key := &TestKey{
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
