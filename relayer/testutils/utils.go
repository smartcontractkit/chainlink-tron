package testutils

import (
	"bytes"
	"crypto/ecdsa"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/pborman/uuid"
)

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
