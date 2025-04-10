package keystore

import (
	"context"

	tronsdk "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-evm/pkg/keys"
)

type loopKeystoreAdapter struct {
	ks keys.Store
}

// Accounts returns the list of enabled addresses from the keystore
func (l *loopKeystoreAdapter) Accounts(ctx context.Context) (accounts []string, err error) {
	enabledAddresses, err := l.ks.EnabledAddresses(ctx)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	addr := tronAddr.EthAddress()
	return l.ks.SignRawUnhashedBytes(ctx, addr, data)
}
