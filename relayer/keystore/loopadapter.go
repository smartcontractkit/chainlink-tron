package keystore

import (
	"context"
	"fmt"

	tronsdk "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/types/core"
	"github.com/smartcontractkit/chainlink-evm/pkg/keys"
)

var _ core.Keystore = (*loopKeystoreAdapter)(nil)

// LoopKeystoreAdapter is an adapter that allows the EVM keystore to be used by the Tron TXM
// It handles the conversion between tron addresses and evm addresses whilst delegating the signing to the EVM keystore
type loopKeystoreAdapter struct {
	ks keys.Store
}

// Creates a new LoopKeystoreAdapter which allows the EVM keystore to be used by the Tron TXM
func NewLoopKeystoreAdapter(ks keys.Store) core.Keystore {
	return &loopKeystoreAdapter{ks: ks}
}

// Accounts returns the list of enabled addresses from the keystore
func (l *loopKeystoreAdapter) Accounts(ctx context.Context) (accounts []string, err error) {
	enabledAddresses, err := l.ks.EnabledAddresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get enabled addresses: %w", err)
	}

	for _, address := range enabledAddresses {
		accounts = append(accounts, address.String())
	}
	return accounts, nil
}

func (l *loopKeystoreAdapter) Sign(ctx context.Context, account string, data []byte) (signed []byte, err error) {
	// We'll need to convert the tron address to an evm address to sign the data
	tronAddr, err := tronsdk.Base58ToAddress(account)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to TRON address: %w", err)
	}

	addr := tronAddr.EthAddress()
	return l.ks.SignRawUnhashedBytes(ctx, addr, data)
}
