package monitor

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/utils/tests"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
)

func TestBalanceMonitor(t *testing.T) {
	const chainID = "Chainlinktest-42"
	ks := keystore{}
	for i := 0; i < 3; i++ {
		addr := generateTronAddress()
		ks = append(ks, addr)
	}

	bals := []int64{0, 1, 1_000_000}
	expBals := []string{
		"0.000000",
		"0.000001",
		"1.000000",
	}

	mockClient := &MockSolidityGRPCClient{}
	var exp []update
	for i := range bals {
		acc := ks[i]
		exp = append(exp, update{acc.String(), expBals[i]})
	}
	mockClient.GetAccountBalanceFunc = func(address tronaddress.Address) (int64, error) {
		for i, addr := range ks {
			if addr.String() == address.String() {
				return bals[i], nil
			}
		}
		return 0, fmt.Errorf("address not found")
	}
	cfg := &config{balancePollPeriod: time.Second}

	var got []update
	baseMonitor := newBalanceMonitor(chainID, cfg, logger.Test(t), ks, mockClient)
	mockMonitor := &mockBalanceMonitor{
		balanceMonitor: baseMonitor,
		got:            got,
		exp:            exp,
		done:           make(chan struct{}),
	}

	mockMonitor.reader = mockClient

	require.NoError(t, mockMonitor.Start(tests.Context(t)))
	t.Cleanup(func() {
		assert.NoError(t, mockMonitor.Close())
	})
	select {
	case <-time.After(tests.WaitTimeout(t)):
		t.Fatal("timed out waiting for balance monitor")
	case <-mockMonitor.done:
	}

	assert.EqualValues(t, exp, mockMonitor.got)
}

func generateTronAddress() tronaddress.Address {
	key := testutils.CreateKey(rand.Reader)
	return key.Address
}

type config struct {
	balancePollPeriod time.Duration
}

func (c *config) BalancePollPeriod() time.Duration {
	return c.balancePollPeriod
}

type keystore []tronaddress.Address

func (k keystore) Accounts(ctx context.Context) (ks []string, err error) {
	for _, acc := range k {
		ks = append(ks, acc.String())
	}
	return
}

func (k keystore) Sign(ctx context.Context, id string, hash []byte) ([]byte, error) {
	// No Op
	return nil, nil
}

type MockSolidityGRPCClient struct {
	GetAccountBalanceFunc func(address tronaddress.Address) (int64, error)
}

func (m *MockSolidityGRPCClient) GetAccountBalance(address tronaddress.Address) (int64, error) {
	if m.GetAccountBalanceFunc != nil {
		return m.GetAccountBalanceFunc(address)
	}
	return 0, fmt.Errorf("GetAccountBalance not implemented")
}

type update struct{ acc, bal string }

type mockBalanceMonitor struct {
	*balanceMonitor
	got  []update
	exp  []update
	done chan struct{}
}

func (m *mockBalanceMonitor) updateProm(acc tronaddress.Address, sun int64) {
	select {
	case <-m.done:
		return
	default:
	}
	v := sunToTrx(sun)
	m.got = append(m.got, update{acc.String(), fmt.Sprintf("%.6f", v)})
	if len(m.got) == len(m.exp) {
		close(m.done)
	}
}
