//go:build integration && testnet

package txm_test

import (
	"os"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
)

func TestTxmShasta(t *testing.T) {
	runTestnetTest(t, "grpc.shasta.trongrid.io:50051")
}

func TestTxmNile(t *testing.T) {
	runTestnetTest(t, "grpc.nile.trongrid.io:50051")
}

func runTestnetTest(t *testing.T, grpcAddress string) {
	logger := logger.Test(t)

	grpcClient := client.NewGrpcClientWithTimeout(grpcAddress, 15*time.Second)
	err := grpcClient.Start(grpc.WithTransportCredentials(credentials.NewTLS(nil)))
	require.NoError(t, err)

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		t.Fatal("PRIVATE_KEY environment variable is not set")
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	require.NoError(t, err)

	pubAddress := address.PubkeyToAddress(privateKey.PublicKey).String()

	logger.Debugw("Loaded private key", "address", pubAddress)

	keystore := testutils.NewTestKeystore(pubAddress, privateKey)

	config := TronTxmConfig{
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
	}

	runTxmTest(t, logger, grpcClient, config, keystore, pubAddress, 5)
}
