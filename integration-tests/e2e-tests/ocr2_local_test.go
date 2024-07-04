//go:build integration

// These tests are for OCR2 Data Feed contracts also known as Datafeeds1.0
// The contracts for OCR2 are located here https://github.com/smartcontractkit/offchain-reporting/blob/master/contract2/src/OCR2Aggregator.sol
// The contracts for Datafeeds are located here https://github.com/smartcontractkit/chainlink-contracts-deprecated/blob/main/contracts/src/v0.6/AggregatorProxy.sol

package e2e_tests

import (
	"context"
	"encoding/hex"
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/contract"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/gauntlet"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployAndInvokeOCR2Local(t *testing.T) {
	ctx := context.Background()
	logger := logger.Test(t)

	// Setup Genesis Account
	genesisAddress, genesisPrivateKey, genesisPrivateKeyHex := utils.SetupTestGenesisAccount(t)
	genesisKeyStore := utils.NewTestKeystore(genesisAddress, genesisPrivateKey)
	logger.Debugw("Using genesis account", "address", genesisAddress)

	// Create 3 accounts for the test
	accounts := utils.CreateRandomAccounts(t, 3)

	err := utils.StartTronNodeWithGenesisAccount(genesisAddress)
	require.NoError(t, err, "Failed to start TRON node")
	logger.Debugw("Started TRON node")

	// TODO: can be refactored to test utils in the future
	jsonRpcPort := "16671"
	var jsonRpcAddress string

	if runtime.GOOS == "darwin" {
		jsonRpcAddress = "127.0.0.1" // Mac OS needs local host port forwarding for docker
	} else {
		jsonRpcAddress = "172.255.0.101" // Linux does not need port forwarding
	}

	jsonRPCClient, err := ethclient.Dial(fmt.Sprintf("http://%s:%s/jsonrpc", jsonRpcAddress, jsonRpcPort))
	require.NoError(t, err, "Failed to connect to TRON json rpc node")

	// Deploy LinkToken
	linkTokenConfig := gauntlet.NewDeploymentLocalTestConfig(genesisPrivateKeyHex)
	provider := gauntlet.NewProvider(ctx, &linkTokenConfig)
	linkTokenTest := gauntlet.NewContractTest("TestDeployAndInvokeLinkToken", &linkTokenConfig, provider, address.Address{})
	err = linkTokenTest.Setup(ctx, contract.LinkToken)
	require.NoError(t, err, "Failed to deploy LinkToken")
	logger.Debugw("Deployed LinkToken")
	grantMintAndBurnRoles(t, ctx, linkTokenTest, genesisAddress, logger)
	logger.Debugw("Granted Mint and Burn Roles")

	// Deploy FeedRegistry
	feedRegistryConfig := gauntlet.NewDeploymentLocalTestConfig(genesisPrivateKeyHex)
	feedRegistryTest := gauntlet.NewContractTest("TestDeployAndInvokeFeedRegistry", &feedRegistryConfig, provider, address.Address{})
	err = feedRegistryTest.Setup(ctx, contract.DataFeeds1Registry)
	require.NoError(t, err, "Failed to deploy FeedRegistry")
	logger.Debugw("Deployed FeedRegistry")

	// Deploy And Set Billing Access Controller
	simpleReadAccessControllerConfig := gauntlet.NewDeploymentLocalTestConfig(genesisPrivateKeyHex)
	simpleReadAccessControllerTest := gauntlet.NewContractTest("TestDeployAndInvokeSimpleReadAccessController", &simpleReadAccessControllerConfig, provider, address.Address{})
	err = simpleReadAccessControllerTest.Setup(ctx, contract.SimpleReadAC)
	require.NoError(t, err)
	logger.Debugw("Deployed SimpleReadAccessController")

	// OCR2 Aggregator Constructor values
	minSubmissionValue := 0
	maxSubmissionValue := new(big.Int)
	maxSubmissionValue.SetString("3138550867693340381917894711603833208051177722232017256447", 10) // taken from https://github.com/smartcontractkit/offchain-reporting/blob/9edbcf74ab7e959ca674c4021bc7021abdb49371/lib/offchainreporting2plus/testsetup/deployment.go#L208
	decimals := 9
	description := "Test OCR2 Aggregator"

	// Deploy OCR2Aggregator
	ocr2AggregatorConfig := gauntlet.NewDeploymentLocalTestConfig(genesisPrivateKeyHex)
	ocr2AggregatorTest := gauntlet.NewContractTest("TestDeployAndInvokeOCR2Aggregator", &ocr2AggregatorConfig, provider, address.Address{})
	err = ocr2AggregatorTest.Setup(ctx, contract.OCR2Aggregator,
		linkTokenTest.ContractAddress().String(),
		minSubmissionValue,
		maxSubmissionValue.String(),
		simpleReadAccessControllerTest.ContractAddress().String(),
		simpleReadAccessControllerTest.ContractAddress().String(),
		decimals,
		description,
	)
	require.NoError(t, err, "Failed to deploy OCR2Aggregator")
	logger.Debugw("Deployed OCR2Aggregator at address", "address", ocr2AggregatorTest.ContractAddress().String())

	// Deploy Aggregator Proxy
	aggregatorProxyConfig := gauntlet.NewDeploymentLocalTestConfig(genesisPrivateKeyHex)
	aggregatorProxyTest := gauntlet.NewContractTest("TestDeployAndInvokeAggregatorProxy", &aggregatorProxyConfig, provider, address.Address{})
	err = aggregatorProxyTest.Setup(ctx, contract.EACAggregatorProxy,
		ocr2AggregatorTest.ContractAddress().String(),
		simpleReadAccessControllerTest.ContractAddress().String(),
	)
	require.NoError(t, err, "Failed to deploy EACAggregatorProxy")
	logger.Debugw("Deployed EACAggregatorProxy")

	// Mint Link tokens to OCR2Aggregator
	amount := big.NewInt(10000000)
	mintTokens(t, ctx, linkTokenTest, amount, ocr2AggregatorTest.ContractAddress().String(), logger)
	contractBalance, err := getBalance(t, ctx, linkTokenTest, ocr2AggregatorTest.ContractAddress().String(), logger)
	require.NoError(t, err, "Failed to get contract balance")
	assert.Equal(t, amount, contractBalance, "Contract balance is not as expected")
	logger.Debugw("Minted tokens")

	// Set OCR2 Config
	signers := []string{genesisAddress, accounts[0].Address, accounts[1].Address, accounts[2].Address}
	transmitters := []string{genesisAddress, accounts[0].Address, accounts[1].Address, accounts[2].Address}
	faultyOracle := 1
	onchainConfig := "0x010000000000000000000000000000000000000000000000007fffffffffffffffffffffffffffffffffffffffffffffff" // https://gist.github.com/yongkangchia/377cdd3204d937b8000c4f31efa891aa
	offchainVersion := 1
	setConfig(t, ctx, ocr2AggregatorTest, logger, signers, transmitters, faultyOracle, onchainConfig, offchainVersion, onchainConfig)
	validateSetConfigLogs(t, ctx, ocr2AggregatorTest, logger, jsonRPCClient, signers, transmitters, faultyOracle, onchainConfig, offchainVersion) // Check Logs

	// Transmit Report

	// Report generation
	reportHex, medianReportValue, err := utils.GenerateOCR2Report()
	require.NoError(t, err, "Failed to generate OCR2 Report")
	reportBytes, err := hex.DecodeString(reportHex[2:])
	require.NoError(t, err, "Failed to decode report into Hex String")

	configDigest := getConfigDigest(t, ctx, ocr2AggregatorTest, logger)
	epochAndRound := "0x0000000000000000000000000000000000000000000000000000000000000001" // 27-byte padding + 4-byte epoch + 1-byte round
	extraHash := "0x0000000000000000000000000000000000000000000000000000000000000000"
	reportContext := []string{
		configDigest,
		epochAndRound,
		extraHash,
	}

	configDigestBytes, err := hex.DecodeString(configDigest[2:])
	require.NoError(t, err)
	var configDigestBytes32 [32]byte
	copy(configDigestBytes32[:], configDigestBytes)

	require.NoError(t, err)
	extraHashBytes, err := hex.DecodeString(extraHash[2:])
	require.NoError(t, err, "Failed to decode extraHash")
	var extraHashBytes32 [32]byte
	copy(extraHashBytes32[:], extraHashBytes)

	reportContextData := utils.ReportContext{
		ReportTimestamp: utils.ReportTimestamp{
			ConfigDigest: configDigestBytes32,
			Epoch:        0,
			Round:        1,
		},
		ExtraHash: extraHashBytes32,
	}

	reportContextBytes := utils.RawReportContext(reportContextData)
	reportContextHexstring := utils.RawReportContextToHexString(reportContextBytes)

	logger.Debugw("Report Generated", "report", reportHex, "reportContext", reportContextHexstring)

	// Signing Report
	var rs [][32]byte
	var ss [][32]byte
	var vs [32]byte

	// Use Genesis account to sign the report
	genesisSignature, err := genesisKeyStore.SignReport(ctx, genesisAddress, reportBytes, reportContextData)
	require.NoError(t, err, "Failed to sign report")
	r, s, v, err := utils.SplitSignature(genesisSignature)
	require.NoError(t, err)
	rs = append(rs, r)
	ss = append(ss, s)
	vs[0] = v

	numSigners := faultyOracle

	// Use numSigners to sign the report
	for i, account := range accounts[:numSigners] {
		signature, err := account.Keystore.SignReport(ctx, account.Address, reportBytes, reportContextData)
		require.NoError(t, err, "Failed to sign report")
		r, s, v, err := utils.SplitSignature(signature)
		require.NoError(t, err, "Failed to split signature")
		rs = append(rs, r)
		ss = append(ss, s)
		vs[i+1] = v
	}

	// Convert the signatures to hex strings for G++
	rawVsHexStr := utils.Convert32BytesToHexString(vs)
	rsHexStr := utils.ConvertSliceOf32BytesToHexStrings(rs)
	ssHexStr := utils.ConvertSliceOf32BytesToHexStrings(ss)
	logger.Debugw("Signing Report", "rs", rs, "ss", ss, "rawVs", vs)
	latestTransmissionDetails(t, ctx, ocr2AggregatorTest, logger)

	// Transmit the report
	transmitReport(t, ctx, ocr2AggregatorTest, logger, reportContext, reportHex, rsHexStr, ssHexStr, rawVsHexStr)
	validateNewTransmissionLogs(t, ctx, ocr2AggregatorTest, logger, jsonRPCClient, medianReportValue, genesisAddress)

	// Get Latest Round
	latestRoundValue := latestTransmissionDetails(t, ctx, ocr2AggregatorTest, logger)

	// Validate Median Answer
	assert.Equal(t, medianReportValue, latestRoundValue, "Median Report Value is not as expected")

	// Get Round
	startNewRound(t, ctx, ocr2AggregatorTest, logger)
	getRoundData(t, ctx, ocr2AggregatorTest, logger, 1)

	// Set OCR2 Billing
	setBilling(t, ctx, ocr2AggregatorTest, logger)
	getBilling(t, ctx, ocr2AggregatorTest, logger)
}

// ===================== Helper Functions ===================== \\

func getConfigDigest(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, logger logger.Logger) string {
	configDigestJson, err := ocr2AggregatorTest.QueryContract(ctx, ocr2AggregatorTest.ContractAddress().String(), contract.AcccessControlledAggregator, "latestConfigDetails()", "")
	require.NoError(t, err, "Failed to get config digest")
	configDigest := configDigestJson.GetStringBytes("2")
	configDigestStr := string(configDigest)
	logger.Debugw("Got Config Digest", "configDigest", configDigestStr)
	return configDigestStr
}

func transmitReport(
	t *testing.T, ctx context.Context,
	ocr2AggregatorTest *gauntlet.ContractTest,
	logger logger.Logger, reportContext []string,
	report string, rs, ss []string, rawVs string,
) {
	logger.Debugw("Transmitting Report", "reportContext", reportContext, "report", report, "rs", rs, "ss", ss, "rawVs", rawVs)
	err := ocr2AggregatorTest.InvokeOperation(ctx, ocr2AggregatorTest.ContractAddress().String(), contract.AcccessControlledAggregator, TransmitFunction, reportContext, report, rs, ss, rawVs)
	require.NoError(t, err, "Failed to transmit report")
	logger.Debugw("Transmitted Report")
}

func setConfig(
	t *testing.T, ctx context.Context,
	ocr2AggregatorTest *gauntlet.ContractTest,
	logger logger.Logger,
	signers, transmitters []string,
	f int, onchainConfig string,
	offchainConfigVersion int,
	offchainConfig string,
) {
	logger.Debugw("Setting Config", "signers", signers, "transmitters", transmitters, "f", f, "onchainConfig", onchainConfig, "offchainConfigVersion", offchainConfigVersion, "offchainConfig", offchainConfig)
	err := ocr2AggregatorTest.InvokeOperation(ctx, ocr2AggregatorTest.ContractAddress().String(), contract.AcccessControlledAggregator, SetConfigFunction, signers, transmitters, f, onchainConfig, offchainConfigVersion, offchainConfig)
	require.NoError(t, err, "Failed to set config for OCR2 Aggregator")
	logger.Debugw("Set Config")
}

func startNewRound(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, logger logger.Logger) {
	err := ocr2AggregatorTest.InvokeOperation(ctx, ocr2AggregatorTest.ContractAddress().String(), contract.AcccessControlledAggregator, StartNewRoundFunction)
	require.NoError(t, err, "Failed to start new round")
	logger.Debugw("Started New Round")
}

func getLatestRound(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, logger logger.Logger) {
	roundJson, err := ocr2AggregatorTest.QueryContract(ctx, ocr2AggregatorTest.ContractAddress().String(), contract.AcccessControlledAggregator, GetLatestRoundFunction, "")
	require.NoError(t, err, "Failed to get latest round")
	logger.Debugw("Got Round", "round", roundJson)
}

func latestTransmissionDetails(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, logger logger.Logger) string {
	roundJson, err := ocr2AggregatorTest.QueryContract(ctx, ocr2AggregatorTest.ContractAddress().String(), contract.AcccessControlledAggregator, LatestTransmissionDetailsFunction, "")
	require.NoError(t, err)
	latestAnswerArray := roundJson.GetObject("3")
	latestAnswer := strings.Trim(latestAnswerArray.Get("hex").String(), "\"")
	return latestAnswer
}

func getRoundData(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, logger logger.Logger, roundId int) {
	roundJson, err := ocr2AggregatorTest.QueryContract(ctx, ocr2AggregatorTest.ContractAddress().String(), contract.AcccessControlledAggregator, GetRoundFunction, "", roundId)
	require.NoError(t, err, "Failed to get round")
	logger.Debugw("Got Round", "round", roundJson)
}

func setBilling(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, logger logger.Logger) {
	maximumGasPriceGwei := uint32(200)
	reasonableGasPriceGwei := uint32(100)
	observationPaymentGjuels := uint32(1000000000)
	transmissionPaymentGjuels := uint32(1000000000)
	accountingGas := uint32(1000000)

	err := ocr2AggregatorTest.InvokeOperation(ctx, ocr2AggregatorTest.ContractAddress().String(), contract.AcccessControlledAggregator, SetBillingFunction, maximumGasPriceGwei, reasonableGasPriceGwei, observationPaymentGjuels, transmissionPaymentGjuels, accountingGas)
	require.NoError(t, err, "Failed to set billing")
	logger.Debugw("Set Billing")
}

func getBilling(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, logger logger.Logger) {
	billingJson, err := ocr2AggregatorTest.QueryContract(ctx, ocr2AggregatorTest.ContractAddress().String(), contract.AcccessControlledAggregator, GetBillingFunction, "")
	require.NoError(t, err, "Failed to get billing")
	logger.Debugw("Got Billing", "billing", billingJson)
}

/** Functions to validate the event logs **/

func validateSetConfigLogs(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, logger logger.Logger, client *ethclient.Client, signers, transmitters []string, faultyOracle int, onchainConfig string, offchainVersion int) {
	eventSignature := "ConfigSet(uint32,bytes32,uint64,address[],address[],uint8,bytes,uint64,bytes)"
	const eventABI = `[{"anonymous":false,"inputs":[{"indexed":false,"internalType":"uint32","name":"previousConfigBlockNumber","type":"uint32"},{"indexed":false,"internalType":"bytes32","name":"configDigest","type":"bytes32"},{"indexed":false,"internalType":"uint64","name":"configCount","type":"uint64"},{"indexed":false,"internalType":"address[]","name":"signers","type":"address[]"},{"indexed":false,"internalType":"address[]","name":"transmitters","type":"address[]"},{"indexed":false,"internalType":"uint8","name":"f","type":"uint8"},{"indexed":false,"internalType":"bytes","name":"onchainConfig","type":"bytes"},{"indexed":false,"internalType":"uint64","name":"offchainConfigVersion","type":"uint64"},{"indexed":false,"internalType":"bytes","name":"offchainConfig","type":"bytes"}],"name":"ConfigSet","type":"event"}]`
	parsedABI, err := abi.JSON(strings.NewReader(eventABI))
	if err != nil {
		panic(err)
	}

	eventHash := utils.FunctionSignatureHash(eventSignature)
	topics := [][]common.Hash{{common.HexToHash(eventHash)}}
	logs := retrieveLogs(t, ctx, ocr2AggregatorTest, topics, client)

	assert.Equal(t, 1, len(logs), "Expected 1 log entry for SetConfig")

	configSetLog := logs[0]
	logData := configSetLog.Data

	event := make(map[string]interface{})
	err = parsedABI.UnpackIntoMap(event, "ConfigSet", logData)
	if err != nil {
		panic(err)
	}

	// Validate the Log data
	assert.Equal(t, uint32(0), event["previousConfigBlockNumber"].(uint32), "Expected previousConfigBlockNumber to be 0")
	assert.Equal(t, uint64(1), event["configCount"], "Expected configCount to be 1")
	assert.Equal(t, uint8(faultyOracle), event["f"].(uint8), "Expected f to be 1")
	assert.Equal(t, onchainConfig, hexutil.Encode(event["onchainConfig"].([]byte)), "Expected onchainConfig to be the same")
	assert.Equal(t, uint64(offchainVersion), event["offchainConfigVersion"].(uint64), "Expected offchainConfigVersion to be 1")

	signersArray := event["signers"].([]common.Address)
	transmittersArray := event["transmitters"].([]common.Address)
	tronSigners := make([]string, len(signersArray))
	tronTransmitters := make([]string, len(transmittersArray))

	for i := range signersArray {
		tronSigners[i] = utils.EthereumToTronAddressBase58(signersArray[i])
		tronTransmitters[i] = utils.EthereumToTronAddressBase58(transmittersArray[i])
	}

	assert.ElementsMatch(t, signers, tronSigners)
	assert.ElementsMatch(t, transmitters, tronTransmitters)

	logger.Debugw("Validated ConfigSet Log", "previousConfigBlockNumber", event["previousConfigBlockNumber"], "configCount", event["configCount"], "f", event["f"], "onchainConfig", event["onchainConfig"], "offchainConfigVersion", event["offchainConfigVersion"])
}

func validateNewTransmissionLogs(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, logger logger.Logger, client *ethclient.Client, expectedMedianAnswer string, expectedTransmitter string) {
	eventSignature := "NewTransmission(uint32,int192,address,uint32,int192[],bytes,int192,bytes32,uint40)"
	const eventABI = `[{"anonymous":false,"inputs":[{"indexed":true,"internalType":"uint32","name":"aggregatorRoundId","type":"uint32"},{"indexed":false,"internalType":"int192","name":"answer","type":"int192"},{"indexed":false,"internalType":"address","name":"transmitter","type":"address"},{"indexed":false,"internalType":"uint32","name":"observationsTimestamp","type":"uint32"},{"indexed":false,"internalType":"int192[]","name":"observations","type":"int192[]"},{"indexed":false,"internalType":"bytes","name":"observers","type":"bytes"},{"indexed":false,"internalType":"int192","name":"juelsPerFeeCoin","type":"int192"},{"indexed":false,"internalType":"bytes32","name":"configDigest","type":"bytes32"},{"indexed":false,"internalType":"uint40","name":"epochAndRound","type":"uint40"}],"name":"NewTransmission","type":"event"}]`

	parsedABI, err := abi.JSON(strings.NewReader(eventABI))
	if err != nil {
		panic(err)
	}
	eventHash := utils.FunctionSignatureHash(eventSignature)
	topics := [][]common.Hash{{common.HexToHash(eventHash)}}
	logs := retrieveLogs(t, ctx, ocr2AggregatorTest, topics, client)

	assert.Equal(t, 1, len(logs), "Expected 1 log entry for Transmit")
	newTransmissionLog := logs[0]
	logData := newTransmissionLog.Data

	event := make(map[string]interface{})
	err = parsedABI.UnpackIntoMap(event, "NewTransmission", logData)
	if err != nil {
		panic(err)
	}

	// Validate the Log data
	expectedAnswerBigInt := new(big.Int)
	expectedAnswerBigInt.SetString(expectedMedianAnswer, 10)
	assert.Equal(t, expectedMedianAnswer[2:], fmt.Sprintf("%x", event["answer"].(*big.Int)), "Expected answer to match")
	assert.Equal(t, expectedTransmitter, utils.EthereumToTronAddressBase58(event["transmitter"].(common.Address)), "Expected transmitter to match")

	logger.Debugw("Validated NewTransmission Log", "answer", event["answer"], "transmitter", event["transmitter"])
}

func retrieveLogs(t *testing.T, ctx context.Context, ocr2AggregatorTest *gauntlet.ContractTest, topics [][]common.Hash, client *ethclient.Client) []types.Log {
	contractAddress := ocr2AggregatorTest.ContractAddress().String()
	ethContractAddress, err := utils.TronAddressBase58ToEthereum(contractAddress)
	require.NoError(t, err, "Failed to convert Tron address to Ethereum address")

	query := ethereum.FilterQuery{
		Addresses: []common.Address{ethContractAddress},
		Topics:    topics,
	}

	logs, err := client.FilterLogs(ctx, query)
	require.NoError(t, err, "Failed to get logs")

	return logs
}

const (
	GetBenchMarkRegistryFunction      = "getBenchmarks(bytes32[])"
	GetReportsRegistryFunction        = "getReports(bytes32[])"
	SetBillingFunction                = "setBilling(uint32,uint32,uint32,uint32,uint24)"
	GetBillingFunction                = "getBilling()"
	StartNewRoundFunction             = "requestNewRound()"
	GetRoundFunction                  = "getRoundData(uint80)"
	GetLatestRoundFunction            = "getLatestRound()"
	SetConfigFunction                 = "setConfig(address[],address[],uint8,bytes,uint64,bytes)"
	TransmitFunction                  = "transmit(bytes32[3],bytes,bytes32[],bytes32[],bytes32)"
	LatestTransmissionDetailsFunction = "latestTransmissionDetails()"
)
