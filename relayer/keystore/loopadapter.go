package keystore

import (
	"context"

	tronsdk "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-evm/pkg/keys"
)

// LoopKeystoreAdapter is an adapter that allows the EVM keystore to be used by the Tron TXM
// It handles the conversion between tron addresses and evm addresses whilst delegating the signing to the EVM keystore
type LoopKeystoreAdapter struct {
	ks keys.Store
}

// Creates a new LoopKeystoreAdapter which allows the EVM keystore to be used by the Tron TXM
func NewLoopKeystoreAdapter(ks keys.Store) *LoopKeystoreAdapter {
	return &LoopKeystoreAdapter{ks: ks}
}

// Accounts returns the list of enabled addresses from the keystore
func (l *LoopKeystoreAdapter) Accounts(ctx context.Context) (accounts []string, err error) {
	enabledAddresses, err := l.ks.EnabledAddresses(ctx)
	if err != nil {
		return nil, err
	}

	for _, address := range enabledAddresses {
		accounts = append(accounts, address.String())
	}
	return accounts, nil
}

func (l *LoopKeystoreAdapter) Sign(ctx context.Context, account string, data []byte) (signed []byte, err error) {
	// We'll need to convert the tron address to an evm address to sign the data
	tronAddr, err := tronsdk.Base58ToAddress(account)
	if err != nil {
		return nil, err
	}

	addr := tronAddr.EthAddress()
	return l.ks.SignRawUnhashedBytes(ctx, addr, data)
}
