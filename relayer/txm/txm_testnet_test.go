//go:build integration && testnet

package txm_test

import (
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/fullnode"
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

func runTestnetTest(t *testing.T, fullnodeUrl string) {
	logger := logger.Test(t)

	fullnodeClient := fullnode.NewClient(fullnodeUrl, sdk.CreateHttpClientWithTimeout(15*time.Second))

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
	}

	runTxmTest(t, logger, fullnodeClient, config, keystore, pubAddress, 5)
}
