package testutils

import (
	"context"
	"crypto/ecdsa"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
)

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
