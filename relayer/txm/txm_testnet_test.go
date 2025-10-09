//go:build integration && testnet

package txm_test

import (
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-tron/relayer/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-tron/relayer/testutils"
	"github.com/smartcontractkit/chainlink-tron/relayer/txm"
)

func TestTxmShasta(t *testing.T) {
	runTestnetTest(t, "grpc.shasta.trongrid.io:50051")
}

func TestTxmNile(t *testing.T) {
	runTestnetTest(t, "https://nile.trongrid.io/wallet")
}

func runTestnetTest(t *testing.T, uri string) {
	logger := logger.Test(t)

	fullnodeUrl, err := url.ParseRequestURI(uri)
	require.NoError(t, err)
	solidityUrl, err := url.ParseRequestURI(uri + "solidity")
	require.NoError(t, err)
	combinedClient, err := sdk.CreateCombinedClient(fullnodeUrl, solidityUrl)
	require.NoError(t, err)

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		t.Fatal("PRIVATE_KEY environment variable is not set")
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	require.NoError(t, err)

	pubAddress := address.PubkeyToAddress(privateKey.PublicKey)

	logger.Debugw("Loaded private key", "address", pubAddress)

	keystore := testutils.NewTestKeystore(pubAddress.String(), privateKey)

	config := txm.TronTxmConfig{
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
		RetentionPeriod:   10 * time.Second,
		ReapInterval:      1 * time.Second,
	}

	runTxmTest(t, logger, combinedClient, config, keystore, pubAddress, 5)
}
