package utils

import (
	"context"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"sort"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median/evmreportcodec"

	testcommon "github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/common"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
)

// Constants for e2e tests
const (
	CLNodeName = "primary"
	SunPerTrx  = 1_000_000
)

// GetRSVFromSignature extracts r, s, and v values from the given signature.
// r = first 32 bytes of signature
// s = second 32 bytes of signature
// v = final 1 byte of signature
func GetRSVFromSignature(signature []byte) (r, s string, v byte, err error) {
	if len(signature) != 65 {
		return "", "", byte(0), errors.New("invalid signature length, expected 65 bytes")
	}

	r = ToHexString(signature[:32])
	s = ToHexString(signature[32:64])
	v = signature[64]

	return r, s, v, nil
}

// ToHexString converts a byte slice to a hex string prefixed with "0x".
func ToHexString(b []byte) string {
	return fmt.Sprintf("0x%s", hex.EncodeToString(b))
}

func ConvertSliceOf32BytesToHexStrings(data [][32]byte) []string {
	var hexStrings []string
	for _, item := range data {
		hexStrings = append(hexStrings, "0x"+hex.EncodeToString(item[:]))
	}
	return hexStrings
}

// Convert32ByteToHexString converts a [32]byte array to a hex string with 0x prefix.
func Convert32BytesToHexString(data [32]byte) string {
	return "0x" + hex.EncodeToString(data[:])
}

// Pad or trim rawVs to ensure it is exactly 32 bytes long
func PadToBytes32(rawVs []byte) []byte {
	if len(rawVs) > 32 {
		return rawVs[:32]
	}
	padded := make([]byte, 32)
	copy(padded, rawVs)
	return padded
}

// Digest of the configuration for a OCR2 protocol instance. The first two
// bytes indicate which config digester (typically specific to a targeted
// blockchain) was used to compute a ConfigDigest. This value is used as a
// domain separator between different protocol instances and is thus security
// critical. It should be the output of a cryptographic hash function over all
// relevant configuration fields as well as e.g. the address of the target
// contract/state accounts/...
type ConfigDigest [32]byte

// ReportTimestamp is the logical timestamp of a report.
type ReportTimestamp struct {
	ConfigDigest ConfigDigest
	Epoch        uint32
	Round        uint8
}

type ReportContext struct {
	ReportTimestamp
	// A hash over some data that is exchanged during execution of the offchain
	// protocol. The data itself is not needed onchain, but we still want to
	// include it in the signature that goes onchain.
	ExtraHash [32]byte
}

func RawReportContext(repctx ReportContext) [3][32]byte {
	rawRepctx := [3][32]byte{}
	copy(rawRepctx[0][:], repctx.ConfigDigest[:])
	binary.BigEndian.PutUint32(rawRepctx[1][32-5:32-1], repctx.Epoch)
	rawRepctx[1][31] = repctx.Round
	rawRepctx[2] = repctx.ExtraHash
	return rawRepctx
}

// Convert RawReportContext to hex string
func RawReportContextToHexString(rawRepctx [3][32]byte) [3]string {
	rawRepctxStr := [3]string{}
	for i, raw := range rawRepctx {
		rawRepctxStr[i] = "0x" + hex.EncodeToString(raw[:])
	}
	return rawRepctxStr
}

func SplitSignature(sig []byte) (r, s [32]byte, v byte, err error) {
	if len(sig) != 65 {
		return r, s, v, fmt.Errorf("SplitSignature: wrong size")
	}
	r = common.BytesToHash(sig[:32])
	s = common.BytesToHash(sig[32:64])
	v = sig[64]
	return r, s, v, nil
}

type ParsedAttributedObservation struct {
	Timestamp        uint32
	Value            *big.Int
	JuelsPerFeeCoin  *big.Int
	GasPriceSubunits *big.Int
	Observer         uint8
}

func GenerateOCR2Report() (hexReport string, medianObservation string, err error) {
	juelsPerFeeCoin := big.NewInt(1e18)

	// Oracle and their observations are generated here https://github.com/smartcontractkit/offchain-reporting/blob/master/lib/offchainreporting2plus/internal/ocr2/contract_tests/helpers_test.go#L97
	// And turned into struct here https://github.com/smartcontractkit/offchain-reporting/blob/7e6c241e684bd322516251d6b70f026ffecb6953/lib/offchainreporting2plus/internal/ocr2/contract_tests/helpers_test.go#L40
	paos := []median.ParsedAttributedObservation{
		{1750755864, big.NewInt(1271127070705242445), juelsPerFeeCoin, big.NewInt(0), 0},
		{901430628, big.NewInt(7142860741125246750), juelsPerFeeCoin, big.NewInt(0), 1},
		{3441600637, big.NewInt(6632808366083118734), juelsPerFeeCoin, big.NewInt(0), 2},
		{2535110149, big.NewInt(8911919877521849818), juelsPerFeeCoin, big.NewInt(0), 3},
		{3892996001, big.NewInt(786026922747303397), juelsPerFeeCoin, big.NewInt(0), 4},
		{632275502, big.NewInt(2731549183433775004), juelsPerFeeCoin, big.NewInt(0), 5},
		{3117570168, big.NewInt(7911589968318461182), juelsPerFeeCoin, big.NewInt(0), 6},
		{2894514909, big.NewInt(5776549002916223716), juelsPerFeeCoin, big.NewInt(0), 7},
		{810936509, big.NewInt(6200928152459444193), juelsPerFeeCoin, big.NewInt(0), 8},
		{1745543156, big.NewInt(6701302325764932302), juelsPerFeeCoin, big.NewInt(0), 9},
		{1480268637, big.NewInt(8154764892406459019), juelsPerFeeCoin, big.NewInt(0), 10},
		{4251749876, big.NewInt(2846500446905292057), juelsPerFeeCoin, big.NewInt(0), 11},
		{2841869438, big.NewInt(6297123943045257721), juelsPerFeeCoin, big.NewInt(0), 12},
		{2778158242, big.NewInt(4898235019263516856), juelsPerFeeCoin, big.NewInt(0), 13},
		{358910537, big.NewInt(538793199214391335), juelsPerFeeCoin, big.NewInt(0), 14},
		{1929768584, big.NewInt(1848167929414573296), juelsPerFeeCoin, big.NewInt(0), 15},
		{2097386607, big.NewInt(6045326344142570107), juelsPerFeeCoin, big.NewInt(0), 16},
		{1796893012, big.NewInt(6191080222126842088), juelsPerFeeCoin, big.NewInt(0), 17},
		{2325194541, big.NewInt(5164923635739237355), juelsPerFeeCoin, big.NewInt(0), 18},
		{806699139, big.NewInt(8639664443292681347), juelsPerFeeCoin, big.NewInt(0), 19},
		{1561217097, big.NewInt(1933969485392146004), juelsPerFeeCoin, big.NewInt(0), 20},
		{2577347004, big.NewInt(6988926203054418701), juelsPerFeeCoin, big.NewInt(0), 21},
		{637874287, big.NewInt(5165075787665716655), juelsPerFeeCoin, big.NewInt(0), 22},
		{2560416105, big.NewInt(6193326628629273182), juelsPerFeeCoin, big.NewInt(0), 23},
		{1636608432, big.NewInt(8221375515390474978), juelsPerFeeCoin, big.NewInt(0), 24},
		{1962450390, big.NewInt(6513272651572558066), juelsPerFeeCoin, big.NewInt(0), 25},
		{2760531991, big.NewInt(7814566626301179977), juelsPerFeeCoin, big.NewInt(0), 26},
		{3432267792, big.NewInt(4173452982976083880), juelsPerFeeCoin, big.NewInt(0), 27},
		{2808783005, big.NewInt(6740236431543500352), juelsPerFeeCoin, big.NewInt(0), 28},
		{2258741909, big.NewInt(4575545089208729518), juelsPerFeeCoin, big.NewInt(0), 29},
		{2051531540, big.NewInt(4567101290862106515), juelsPerFeeCoin, big.NewInt(0), 30},
	}

	// Extract the observations from paos
	observations := make([]*big.Int, len(paos))
	for i, pao := range paos {
		observations[i] = pao.Value
	}

	// Calculate the median observation
	// Sort the observations by value
	sort.Slice(paos, func(i, j int) bool {
		return paos[i].Value.Cmp(paos[j].Value) < 0
	})

	// Calculate the median observation
	medianIndex := len(paos) / 2
	median := paos[medianIndex].Value
	// Convert the median to hex format
	medianHex := fmt.Sprintf("0x%x", median)

	reportCodec := evmreportcodec.ReportCodec{}
	report, err := reportCodec.BuildReport(context.Background(), paos)
	if err != nil {
		return "", "", err
	}
	hexReport = "0x" + fmt.Sprintf("%x", report)

	return hexReport, medianHex, nil
}

// Create keccak256 hex string hash of function signature
func FunctionSignatureHash(functionSignature string) string {
	hash := crypto.Keccak256([]byte(functionSignature))
	return "0x" + hex.EncodeToString(hash)
}

// Converts a Tron address to an Ethereum address.
func MustConvertToEthAddress(t *testing.T, tronAddress string) common.Address {
	tronHexAddress, err := address.StringToAddress(tronAddress)
	if err != nil {
		t.Fatal(err)
	}
	return tronHexAddress.EthAddress()
}

func MustMarshalParams(t *testing.T, params ...any) string {
	encodedParams := make([]map[string]any, 0)
	if len(params)%2 == 1 {
		t.Fatal("odd number of params")
	}
	for i := 0; i < len(params); i += 2 {
		paramType := params[i]
		paramTypeStr, ok := paramType.(string)
		if !ok {
			t.Fatal("non-string param type")
		}
		encodedParams = append(encodedParams, map[string]any{paramTypeStr: params[i+1]})
	}

	paramsJsonBytes, err := json.Marshal(encodedParams)
	if err != nil {
		t.Fatal(err)
	}

	return string(paramsJsonBytes)
}

func MustConvertAddress(t *testing.T, tronAddress string) address.Address {
	a, err := address.StringToAddress(tronAddress)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

// SetupLocalStack sets up chainlink node, client, gRPC client and config for the local tests
func SetupLocalStack(t *testing.T, logger zerolog.Logger, genesisAddress string) (sdk.CombinedClient, *testcommon.ChainlinkClient, *testcommon.Common) {
	fullNodeUrl := os.Getenv("FULL_NODE_URL")
	if fullNodeUrl == "" {
		fullNodeUrl = fmt.Sprintf("http://%s:%s/wallet", testutils.GetTronNodeIpAddress(), testutils.FullNodePort)
	}
	solidityNodeUrl := os.Getenv("SOLIDITY_URL")
	if solidityNodeUrl == "" {
		solidityNodeUrl = fmt.Sprintf("http://%s:%s/walletsolidity", testutils.GetTronNodeIpAddress(), testutils.SolidityNodePort)
	}
	jsonRpcUrl := os.Getenv("JSON_RPC_URL")
	if jsonRpcUrl == "" {
		jsonRpcUrl = fmt.Sprintf("http://%s:%s/jsonrpc", testutils.GetTronNodeIpAddress(), testutils.JsonRpcPort)
	}
	internalFullNodeUrl := os.Getenv("INTERNAL_FULL_NODE_URL")
	if internalFullNodeUrl == "" {
		internalFullNodeUrl = testutils.DefaultInternalFullNodeUrl
	}
	internalSolidityNodeUrl := os.Getenv("INTERNAL_SOLIDITY_NODE_URL")
	if internalSolidityNodeUrl == "" {
		internalSolidityNodeUrl = testutils.DefaultInternalSolidityNodeUrl
	}
	internalJsonRpcUrl := os.Getenv("INTERNAL_JSON_RPC_URL")
	if internalJsonRpcUrl == "" {
		internalJsonRpcUrl = testutils.DefaultInternalJsonRpcUrl
	}

	logger.Info().Msg("Starting java-tron container...")
	err := testutils.StartTronNode(genesisAddress)
	require.NoError(t, err, "Could not start java-tron container")
	logger.Info().Str("fullNodeUrl", fullNodeUrl).Str("solidityNodeUrl", solidityNodeUrl).Msg("TRON node config")

	return setUpTronEnvironment(t, logger, fullNodeUrl, solidityNodeUrl, jsonRpcUrl, internalFullNodeUrl, internalSolidityNodeUrl, internalJsonRpcUrl, genesisAddress)
}

func TeardownLocalStack(t *testing.T, logger zerolog.Logger, commonConfig *testcommon.Common) {
	logger.Info().Msg("Stopping chainlink-nodes container...")
	commonConfig.TearDownLocalEnvironment(t)
	logger.Info().Msg("Tearing down java-tron container...")
	err := testutils.StopTronNode()
	require.NoError(t, err, "Could not tear down java-tron container")
}

// SetupTestnetStack sets up chainlink node, client, gRPC client and config for the testnet tests
func SetupTestnetStack(t *testing.T, logger zerolog.Logger, pubAddress, network string) (grpcClient sdk.CombinedClient, chainlinkClient *testcommon.ChainlinkClient, commonConfig *testcommon.Common) {
	var fullNodeUrl, solidityNodeUrl, jsonRpcUrl string
	switch network {
	case testutils.Shasta:
		fullNodeUrl = testutils.ShastaFullNodeUrl
		solidityNodeUrl = testutils.ShastaSolidityNodeUrl
		jsonRpcUrl = testutils.ShastaJsonRpcUrl
	case testutils.Nile:
		fullNodeUrl = testutils.NileFullNodeUrl
		solidityNodeUrl = testutils.NileSolidityNodeUrl
		jsonRpcUrl = testutils.NileJsonRpcUrl
	default:
		t.Fatalf("Unknown network: %s", network)
	}
	return setUpTronEnvironment(t, logger, fullNodeUrl, solidityNodeUrl, jsonRpcUrl, fullNodeUrl, solidityNodeUrl, jsonRpcUrl, pubAddress)
}

func TeardownTestnetStack(t *testing.T, logger zerolog.Logger, commonConfig *testcommon.Common) {
	logger.Info().Msg("Stopping chainlink-nodes container...")
	commonConfig.TearDownLocalEnvironment(t)
}

// setupTronEnvironment creates chainlink client, gRPC client and chainId for the tests
func setUpTronEnvironment(
	t *testing.T, logger zerolog.Logger,
	fullNodeUrl, solidityNodeUrl, jsonRpcUrl,
	internalFullNodeUrl, internalSolidityNodeUrl, internalJsonRpcUrl,
	genesisAddress string,
) (sdk.CombinedClient, *testcommon.ChainlinkClient, *testcommon.Common) {
	fullNodeUrlObj, err := url.Parse(fullNodeUrl)
	require.NoError(t, err)
	solidityNodeUrlObj, err := url.Parse(solidityNodeUrl)
	require.NoError(t, err)
	jsonRpcUrlObj, err := url.Parse(jsonRpcUrl)
	require.NoError(t, err)

	combinedClient, err := sdk.CreateCombinedClient(fullNodeUrlObj, solidityNodeUrlObj, jsonRpcUrlObj)
	require.NoError(t, err)

	blockInfo, err := combinedClient.GetBlockByNum(0)
	require.NoError(t, err)

	blockId := blockInfo.BlockID
	// previously, we took the whole genesis block id as the chain id, which is the case depending on java-tron node config:
	// https://github.com/tronprotocol/java-tron/blob/b1fc2f0f2bd79527099bc3027b9aba165c2e20c2/actuator/src/main/java/org/tron/core/vm/program/Program.java#L1271
	//
	// however on both mainnet, testnets committee.allowOptimizedReturnValueOfChainId is enabled, so we've done the same for devnet
	// and the last 4 bytes is the chain id both when retrieved by eth_chainId and via the `block.chainid` call in the TVM, which
	// is important for the config digest calculation:
	// https://github.com/smartcontractkit/libocr/blob/063ceef8c42eeadbe94221e55b8892690d36099a/contract2/OCR2Aggregator.sol#L27
	chainIdHex := blockId[len(blockId)-8:]
	chainIdInt := new(big.Int)
	chainIdInt.SetString(chainIdHex, 16)
	chainId := chainIdInt.String()
	logger.Info().Str("chain id", chainId).Msg("Read first block")

	commonConfig := testcommon.NewCommon(t, chainId, internalFullNodeUrl, internalSolidityNodeUrl, internalJsonRpcUrl)
	commonConfig.SetLocalEnvironment(t, genesisAddress)

	chainlinkClient, err := testcommon.NewChainlinkClient(commonConfig.Env, commonConfig.ChainId, CLNodeName)
	require.NoError(t, err, "Could not create chainlink client")

	logMessage := logger.Info()
	for i, nodeAddress := range chainlinkClient.GetNodeAddresses() {
		logMessage.Str(fmt.Sprintf("node %d", i+1), nodeAddress.String())
	}
	logMessage.Msg("Created chainlink client")

	return combinedClient, chainlinkClient, commonConfig
}
