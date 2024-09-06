package ocr2_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"fmt"
	"math/big"
	"os"
	"testing"
	"time"

	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/require"

	relaylogger "github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/common"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/contract"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/utils"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/ocr2"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/reader"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
)

func TestOCRLocal(t *testing.T) {
	logger := common.GetTestLogger(t)

	var genesisAddress string
	var genesisPrivateKey *ecdsa.PrivateKey
	privateKeyHex := os.Getenv("PRIVATE_KEY")
	if privateKeyHex == "" {
		genesisAccountKey := testutils.CreateKey(rand.Reader)
		genesisAddress = genesisAccountKey.Address.String()
		genesisPrivateKey = genesisAccountKey.PrivateKey
	} else {
		privateKey, err := crypto.HexToECDSA(privateKeyHex)
		require.NoError(t, err)

		genesisAddress = address.PubkeyToAddress(privateKey.PublicKey).String()
		genesisPrivateKey = privateKey
	}
	logger.Info().Str("genesis address", genesisAddress).Msg("Using genesis account")

	runOCR2Test(t, logger, genesisPrivateKey, genesisAddress, testutils.Devnet)
}

func runOCR2Test(
	t *testing.T, logger zerolog.Logger,
	privateKey *ecdsa.PrivateKey,
	pubAddress string, network string,
) {
	var httpUrl string
	var combinedClient *sdk.CombinedGrpcClient
	var chainlinkClient *common.ChainlinkClient
	var commonConfig *common.Common
	var feeLimit int
	var txnWaitTime int
	var pollFrequency int
	var ocrTransmissionFrequency time.Duration

	switch network {
	case testutils.Devnet:
		combinedClient, chainlinkClient, commonConfig = utils.SetupLocalStack(t, logger, pubAddress)
		defer utils.TeardownLocalStack(t, logger, commonConfig)
		httpUrl = os.Getenv("HTTP_URL")
		if httpUrl == "" {
			httpUrl = fmt.Sprintf("http://%s:%s", testutils.GetTronNodeIpAddress(), testutils.HttpPort)
		}
		feeLimit = testutils.DevnetFeeLimit
		txnWaitTime = testutils.DevnetMaxWaitTime
		pollFrequency = testutils.DevnetPollFrequency
		ocrTransmissionFrequency = testutils.DevnetOcrTransmissionFrequency

	case testutils.Shasta:
		httpUrl = testutils.ShastaHttpUrl
		combinedClient, chainlinkClient, commonConfig = utils.SetupTestnetStack(t, logger, pubAddress, network)
		defer utils.TeardownTestnetStack(t, logger, commonConfig)
		feeLimit = testutils.TestnetFeeLimit
		txnWaitTime = testutils.TestnetMaxWaitTime
		pollFrequency = testutils.TestnetPollFrequency
		ocrTransmissionFrequency = testutils.TestnetOcrTransmissionFrequency

	case testutils.Nile:
		httpUrl = testutils.NileHttpUrl
		combinedClient, chainlinkClient, commonConfig = utils.SetupTestnetStack(t, logger, pubAddress, network)
		defer utils.TeardownTestnetStack(t, logger, commonConfig)
		feeLimit = testutils.TestnetFeeLimit
		txnWaitTime = testutils.TestnetMaxWaitTime
		pollFrequency = testutils.TestnetPollFrequency
		ocrTransmissionFrequency = testutils.TestnetOcrTransmissionFrequency

	default:
		t.Fatal("Unsupported network")
	}

	clientLogger, err := relaylogger.New()
	require.NoError(t, err, "Could not create relay logger")

	testKeystore := testutils.NewTestKeystore(pubAddress, privateKey)
	txmgr := txm.New(clientLogger, testKeystore, combinedClient, txm.TronTxmConfig{
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
	})
	err = txmgr.Start(context.Background())
	require.NoError(t, err)

	logger.Info().Str("from", pubAddress).Msg("Funding nodes")

	var transferAmount int64 = utils.SunPerTrx * 500 // 500 TRX
	for _, nodeAddr := range chainlinkClient.GetNodeAddresses() {
		transferTx, err := combinedClient.Transfer(pubAddress, nodeAddr, transferAmount)
		require.NoError(t, err, "Creation of Transfer Txn from genesis account to node failed")
		_, err = txmgr.SignAndBroadcast(context.Background(), pubAddress, transferTx)
		require.NoError(t, err, "Broadcast of Transfer Txn from genesis account to node failed")
	}

	startTime := time.Now()

	// Check that the nodes have been funded
	for _, nodeAddr := range chainlinkClient.GetNodeAddresses() {
		for {
			// use the full node grpc client to check account for quicker feedback.
			accountInfo, err := combinedClient.GrpcClient.GetAccount(nodeAddr)
			if err != nil {
				// do not error on 'account not found' - this occurs when there is no account info (transfer hasnt executed yet)
				if err.Error() == "account not found" {
					time.Sleep(time.Second * time.Duration(pollFrequency))
					continue
				}
				logger.Error().Str("address", nodeAddr).Err(err)
				t.Fatal("failed to get account info")
			}

			if accountInfo.Balance != transferAmount {
				time.Sleep(time.Second * time.Duration(pollFrequency))
				continue
			}

			// timeout
			if time.Since(startTime).Seconds() > float64(txnWaitTime) {
				t.Fatal("failed to fund nodes in time")
			}
			break
		}
		logger.Info().Str("address", nodeAddr).Msg("successfully funded")
	}

	deployContract := func(contractName string, artifact *contract.Artifact, params []interface{}) string {
		txHash := testutils.DeployContractByJson(t, httpUrl, testKeystore, pubAddress, contractName, artifact.AbiJson, artifact.Bytecode, feeLimit, params)
		// use full node client for quicker feedback
		txInfo := testutils.WaitForTransactionInfo(t, combinedClient.GrpcClient, txHash, txnWaitTime)
		contractAddress := address.Address(txInfo.ContractAddress).String()
		contractDeployed := testutils.CheckContractDeployed(t, httpUrl, contractAddress)
		require.True(t, contractDeployed, "Contract not deployed")
		return contractAddress
	}

	linkTokenArtifact := contract.MustLoadArtifact(t, "link-v0.8/LinkToken.json")
	linkTokenAddress := deployContract("LinkToken", linkTokenArtifact, nil)
	logger.Info().Str("address", linkTokenAddress).Msg("Link token contract deployed")

	readAccessControllerArtifact := contract.MustLoadArtifact(t, "ocr2-v0.8/SimpleReadAccessController.json")
	billingAccessControllerAddress := deployContract("SimpleReadAccessController", readAccessControllerArtifact, nil)
	logger.Info().Str("address", billingAccessControllerAddress).Msg("Billing access controller deployed")
	requesterAccessControllerAddress := deployContract("SimpleReadAccessController", readAccessControllerArtifact, nil)
	logger.Info().Str("address", requesterAccessControllerAddress).Msg("Requester access controller deployed")

	minAnswer := big.NewInt(0)
	maxAnswer := new(big.Int)
	// taken from https://github.com/smartcontractkit/offchain-reporting/blob/9edbcf74ab7e959ca674c4021bc7021abdb49371/lib/offchainreporting2plus/testsetup/deployment.go#L208
	maxAnswer.SetString("3138550867693340381917894711603833208051177722232017256447", 10)
	// TODO: check default for decimals
	decimals := uint8(9)
	description := "Test OCR2 Aggregator"
	ocr2AggregatorArtifact := contract.MustLoadArtifact(t, "ocr2-v0.8/OCR2Aggregator.json")
	ocr2AggregatorAddress := deployContract("OCR2Aggregator", ocr2AggregatorArtifact, []interface{}{
		utils.MustConvertToEthAddress(t, linkTokenAddress),
		minAnswer,
		maxAnswer,
		utils.MustConvertToEthAddress(t, billingAccessControllerAddress),
		utils.MustConvertToEthAddress(t, requesterAccessControllerAddress),
		decimals,
		description,
	})
	logger.Info().Str("address", ocr2AggregatorAddress).Msg("OCR2 aggregator deployed")

	eacAggregatorProxyArtifact := contract.MustLoadArtifact(t, "ocr2-v0.6/EACAggregatorProxy.json")
	eacAggregatorProxyAddress := deployContract("EACAggregatorProxy", eacAggregatorProxyArtifact, []interface{}{
		utils.MustConvertToEthAddress(t, ocr2AggregatorAddress),
		utils.MustConvertToEthAddress(t, requesterAccessControllerAddress),
	})
	logger.Info().Str("address", eacAggregatorProxyAddress).Msg("Aggregator proxy deployed")

	mintAmount := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	mintAmount = mintAmount.Mul(mintAmount, big.NewInt(50000))

	err = txmgr.Enqueue(pubAddress, linkTokenAddress, "grantMintAndBurnRoles(address)", "address", pubAddress)
	require.NoError(t, err)
	err = txmgr.Enqueue(pubAddress, linkTokenAddress, "mint(address,uint256)", "address", ocr2AggregatorAddress, "uint256", mintAmount.String())
	require.NoError(t, err)
	testutils.WaitForInflightTxs(clientLogger, txmgr, time.Second*time.Duration(txnWaitTime))

	// Use the full node grpc client to check balance for quicker feedback.
	balanceResponse, err := combinedClient.GrpcClient.TriggerConstantContract("", linkTokenAddress, "balanceOf(address)", []any{"address", ocr2AggregatorAddress})
	require.NoError(t, err)
	balanceValue := new(big.Int).SetBytes(balanceResponse.ConstantResult[0])

	require.Equal(t, mintAmount, balanceValue)
	logger.Info().Str("amount", mintAmount.String()).Msg("Minted LINK token")

	signers, transmitters, f, _, offchainConfigVersion, offchainConfig := chainlinkClient.GetSetConfigArgs(t)

	transmittersStr := make([]string, len(transmitters))
	for i, t := range transmitters {
		evmAddress := ethcommon.HexToAddress(string(t))
		transmittersStr[i] = string(utils.EthereumToTronAddressBase58(evmAddress))
	}
	require.Equal(t, chainlinkClient.GetNodeAddresses(), transmittersStr, "Transmitters should match node addresses")

	// Define the values
	onchainConfigBytes := []byte{}
	// version (uint8)
	onchainConfigBytes = append(onchainConfigBytes, byte(1))
	// minAnswer (int192)
	onchainConfigBytes = append(onchainConfigBytes, ethcommon.LeftPadBytes(minAnswer.Bytes(), 24)...)
	// maxAnswer (int192)
	onchainConfigBytes = append(onchainConfigBytes, ethcommon.LeftPadBytes(maxAnswer.Bytes(), 24)...)

	signerAddresses := []string{}
	for _, signer := range signers {
		// TODO: gotron-sdk only supports base58 addresses as input for address or address[], update it so that
		// we can pass common.Address directly
		signerAddresses = append(signerAddresses, utils.EthereumToTronAddressBase58(ethcommon.BytesToAddress(signer)))
	}

	// TODO: should we set onchainConfig as offchainConfig?
	err = txmgr.Enqueue(pubAddress, ocr2AggregatorAddress, "setConfig(address[],address[],uint8,bytes,uint64,bytes)",
		/* signers= */ "address[]", signerAddresses,
		/* trasmitters= */ "address[]", transmittersStr,
		/* f= */ "uint8", fmt.Sprintf("%d", f),
		/* onchainConfig= */ "bytes", onchainConfigBytes,
		/* offchainConfigVersion= */ "uint64", fmt.Sprintf("%d", offchainConfigVersion),
		/* offchainConfig= */ "bytes", offchainConfig)
	require.NoError(t, err)

	// TODO: we need to fix the txmgr from returning 0 inflight count when it's processing a single transaction with nothing queued.
	time.Sleep(time.Second * time.Duration(pollFrequency))

	testutils.WaitForInflightTxs(clientLogger, txmgr, time.Second*time.Duration(txnWaitTime))

	configDetailsResponse, err := combinedClient.GrpcClient.TriggerConstantContract("", ocr2AggregatorAddress, "latestConfigDetails()", nil)
	require.NoError(t, err)

	configCount := new(big.Int).SetBytes(configDetailsResponse.ConstantResult[0][0:32])
	require.NoError(t, err)
	require.Equal(t, big.NewInt(1), configCount)
	logger.Info().Msg("successfully set config")

	p2pPort := "50200"
	err = chainlinkClient.CreateJobsForContract(
		commonConfig.ChainId,
		utils.CLNodeName,
		p2pPort,
		commonConfig.MockUrl,
		commonConfig.JuelsPerFeeCoinSource,
		ocr2AggregatorAddress)
	require.NoError(t, err, "Could not create jobs for contract")

	err = validateRounds(t, combinedClient, utils.MustConvertAddress(t, ocr2AggregatorAddress), utils.MustConvertAddress(t, eacAggregatorProxyAddress), commonConfig.IsSoak, ocrTransmissionFrequency, commonConfig.TestDuration)
	require.NoError(t, err, "Validating round should not fail")

}

func validateRounds(t *testing.T, combinedClient sdk.GrpcClient, ocrAddress address.Address, ocrProxyAddress address.Address, isSoak bool, ocrTransmissionFrequency, testDuration time.Duration) error {
	var rounds int
	if isSoak {
		rounds = 99999999
	} else {
		rounds = 10
	}

	logger := common.GetTestLogger(t)
	ctx := context.Background() // context background used because timeout handled by requestTimeout param
	// assert new rounds are occurring
	increasing := 0 // track number of increasing rounds
	var stuck bool
	stuckCount := 0

	ocrLogger, err := relaylogger.New()
	require.NoError(t, err, "Failed to create OCR relay logger")

	readerClient := reader.NewReader(combinedClient, ocrLogger)
	ocrReader := ocr2.NewOCR2Reader(readerClient, ocrLogger)

	previous := ocr2.TransmissionDetails{}

	for start := time.Now(); time.Since(start) < testDuration; {
		logger.Info().Msg(fmt.Sprintf("Elapsed time: %s, Round wait: %s ", time.Since(start), testDuration))

		// end condition: enough rounds have occurred
		if !isSoak && increasing >= rounds {
			break
		}

		// end condition: rounds have been stuck
		if stuck && stuckCount > 50 {
			logger.Debug().Msg("failing to fetch transmissions means blockchain may have stopped")
			break
		}

		// try to fetch rounds
		time.Sleep(ocrTransmissionFrequency)
		current, err := ocrReader.LatestTransmissionDetails(ctx, ocrAddress)
		if err != nil {
			logger.Error().Msg(fmt.Sprintf("Transmission Error: %+v", err))
			t.Fatal("Failed to get latest transmission details", err)
			continue
		}

		// if no changes, increment stuck counter and continue
		if current.Epoch == previous.Epoch && current.Round == previous.Round {
			stuckCount++
			if stuckCount > 30 {
				stuck = true
				increasing = 0
			}
			continue
		}

		// epoch or round has changed, log transmission details
		logger.Info().Msg(fmt.Sprintf("Transmission Details: %+v", current))

		// validate epoch/round/timestamp increasing
		if current.Epoch < previous.Epoch || (current.Epoch == previous.Epoch && current.Round < previous.Round) {
			logger.Error().Msg(fmt.Sprintf("Epoch/Round should be increasing - previous epoch %d round %d, current epoch %d round %d", previous.Epoch, previous.Round, current.Epoch, current.Round))
		}
		if current.LatestTimestamp.Before(previous.LatestTimestamp) {
			logger.Error().Msg(fmt.Sprintf("LatestTimestamp should be increasing - previous: %s, current: %s", previous.LatestTimestamp, current.LatestTimestamp))
		}
		if !isSoak {
			require.True(t, current.Epoch > previous.Epoch || (current.Epoch == previous.Epoch && current.Round > previous.Round), "Epoch/Round should be increasing")
			require.GreaterOrEqual(t, current.LatestTimestamp, previous.LatestTimestamp, "Latest timestamp should be increasing")
		}

		// check latest answer is positive
		ansCmp := current.LatestAnswer.Cmp(big.NewInt(0))
		if ansCmp != 1 {
			logger.Error().Msg(fmt.Sprintf("LatestAnswer should be greater than zero, got %s", current.LatestAnswer.String()))
		}
		if !isSoak {
			require.Equal(t, ansCmp == 1, true, "LatestAnswer should be greater than zero")
		}

		// check no changes to config digest
		emptyDigest := types.ConfigDigest{}
		if previous.Digest != emptyDigest {
			if current.Digest != previous.Digest {
				logger.Error().Msg(fmt.Sprintf("Config digest should not change, expected %s got %s", previous.Digest, current.Digest))
			}
			if !isSoak {
				require.Equal(t, current.Digest, previous.Digest, "Config digest should not change")
			}
		}

		// transmission updated, reset stuck trackers and increment increasing rounds
		increasing++
		stuck = false
		stuckCount = 0
		previous = current
	}

	if !isSoak {
		require.GreaterOrEqual(t, increasing, rounds, "Round + epochs should be increasing")
		require.Equal(t, stuck, false, "Round + epochs should not be stuck")
	}

	/// Test proxy reading
	// TODO: would be good to test proxy switching underlying feeds
	mockAdapterValue := 5
	latestRoundData, err := ocrReader.LatestRoundData(ctx, ocrProxyAddress)
	if !isSoak {
		require.NoError(t, err, "Reading round data from proxy should not fail")
	}
	value := latestRoundData.Answer.Int64()
	require.Equal(t, value, int64(mockAdapterValue), "Reading from proxy should return correct value")

	return nil
}
