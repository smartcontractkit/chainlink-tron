//go:build integration && testnet

// For Testnet, we need to run with timeout flag of 30 mins or more.
// This is because checking for solidified transactions can take a long time.
package ocr2_test

import (
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/smartcontractkit/chainlink-tron/relayer/gotron-sdk/pkg/address"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-tron/integration-tests/common"
	"github.com/smartcontractkit/chainlink-tron/relayer/testutils"
)

// TestOCR2Shasta runs e2e tests for OCR2 Datafeedds contracts on Shasta testnet
func TestOCR2Shasta(t *testing.T) {
	runTestnetTest(t, testutils.Shasta)
}

// TestOCR2Nile runs e2e tests for OCR2 Datafeedds contracts on Nile testnet
func TestOCR2Nile(t *testing.T) {
	runTestnetTest(t, testutils.Nile)
}

func runTestnetTest(t *testing.T, network string) {
	logger := common.GetTestLogger(t)

	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		t.Fatal("PRIVATE_KEY environment variable is not set")
	}
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	require.NoError(t, err)

	pubAddress := address.PubkeyToAddress(privateKey.PublicKey)

	runOCR2Test(t, logger, privateKey, pubAddress, network)
}
