package monitor

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-tron/relayer/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-tron/relayer/gotron-sdk/pkg/http/soliditynode"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/types/core"
	"github.com/smartcontractkit/chainlink-common/pkg/utils/tests"
	"github.com/smartcontractkit/chainlink-tron/relayer"
	"github.com/smartcontractkit/chainlink-tron/relayer/testutils"
)

func TestBalanceMonitor(t *testing.T) {
	const chainID = "Chainlinktest-42"
	ks := keystore{}
	accounts := []address.Address{}
	for i := 0; i < 3; i++ {
		pubKeyHex := generatePublicKeyHex()
		addr, err := relayer.PublicKeyToTronAddress(pubKeyHex)
		assert.NoError(t, err)
		ks.keys = append(ks.keys, pubKeyHex)
		accounts = append(accounts, addr)
	}

	bals := []int64{0, 1, 1_000_000}
	expBals := []string{
		"0.000000",
		"0.000001",
		"1.000000",
	}

	mockClient := &MockSolidityGRPCClient{}
	type update struct{ acc, bal string }
	var exp []update
	for i := range bals {
		acc := accounts[i]
		exp = append(exp, update{acc.String(), expBals[i]})
	}
	mockClient.GetAccountBalanceFunc = func(accountAddress address.Address) (int64, error) {
		for i, acc := range accounts {
			if acc.String() == accountAddress.String() {
				return bals[i], nil
			}
		}
		return 0, fmt.Errorf("address not found")
	}
	cfg := &config{balancePollPeriod: time.Second}
	b := newBalanceMonitor(chainID, cfg, logger.Test(t), &ks, func() (BalanceClient, error) {
		return mockClient, nil
	})
	var got []update
	done := make(chan struct{})
	b.updateFn = func(acc address.Address, sun int64) {
		select {
		case <-done:
			return
		default:
		}
		v := sunToTrx(sun)
		got = append(got, update{acc.String(), fmt.Sprintf("%.6f", v)})
		if len(got) == len(exp) {
			close(done)
		}
	}
	b.reader = mockClient

	require.NoError(t, b.Start(tests.Context(t)))
	t.Cleanup(func() {
		assert.NoError(t, b.Close())
	})
	select {
	case <-time.After(tests.WaitTimeout(t)):
		t.Fatal("timed out waiting for balance monitor")
	case <-done:
	}

	assert.EqualValues(t, exp, got)
}

func generateTronAddress() address.Address {
	key := testutils.CreateKey(rand.Reader)
	return key.Address
}

func generatePublicKeyHex() string {
	randBytes := make([]byte, 64)
	_, err := rand.Reader.Read(randBytes)
	if err != nil {
		panic("key generation: could not read from random source: " + err.Error())
	}
	reader := bytes.NewReader(randBytes)
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}
	pubKeyBytes := crypto.FromECDSAPub(&privateKeyECDSA.PublicKey)
	return fmt.Sprintf("%x", pubKeyBytes)
}

type config struct {
	balancePollPeriod time.Duration
}

func (c *config) BalancePollPeriod() time.Duration {
	return c.balancePollPeriod
}

type keystore struct {
	core.UnimplementedKeystore
	keys []string
}

func (k keystore) Accounts(ctx context.Context) ([]string, error) {
	ks := []string{}
	for _, acc := range k.keys {
		ks = append(ks, acc)
	}
	return ks, nil
}

func (k keystore) Sign(ctx context.Context, id string, hash []byte) ([]byte, error) {
	// No Op
	return nil, nil
}

type MockSolidityGRPCClient struct {
	GetAccountBalanceFunc func(accountAddress address.Address) (int64, error)
}

func (m *MockSolidityGRPCClient) GetAccount(accountAddress address.Address) (*soliditynode.GetAccountResponse, error) {
	if m.GetAccountBalanceFunc != nil {
		balance, err := m.GetAccountBalanceFunc(accountAddress)
		if err != nil {
			return nil, err
		}
		return &soliditynode.GetAccountResponse{Balance: balance}, nil
	}
	return nil, fmt.Errorf("GetAccount not implemented")
}
