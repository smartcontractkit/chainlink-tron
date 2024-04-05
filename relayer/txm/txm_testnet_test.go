//go:build integration && testnet

package txm

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/stretchr/testify/require"
)

func TestTxmShasta(t *testing.T) {
	logger := logger.Test(t)

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		t.Fatal("PRIVATE_KEY environment variable is not set")
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	require.NoError(t, err)

	pubAddress := address.PubkeyToAddress(privateKey.PublicKey).String()

	logger.Debugw("Loaded private key", "address", pubAddress)

	keystore := newTestKeystore(pubAddress, privateKey)

	config := TronTxmConfig{
		RPCAddress:        "grpc.shasta.trongrid.io:50051",
		RPCInsecure:       true,
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
	}

	runTxmTest(t, logger, config, keystore, pubAddress)
}
