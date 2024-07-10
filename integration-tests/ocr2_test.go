package ocr2_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"math/big"
	"net/url"
	"os"
	"strings"
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

func TestOCRBasic(t *testing.T) {
	// Set up test environment
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

	grpcClient, chainlinkClient, commonConfig := setupLocalStack(t, logger, genesisAddress)
	defer teardownLocalStack(t, logger, commonConfig)

	clientLogger, err := relaylogger.New()
	require.NoError(t, err, "Could not create relay logger")

	testKeystore := testutils.NewTestKeystore(genesisAddress, genesisPrivateKey)
	txmgr := txm.New(clientLogger, testKeystore, grpcClient, txm.TronTxmConfig{
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
	})
	err = txmgr.Start(context.Background())
	require.NoError(t, err)

	logger.Info().Str("from", genesisAddress).Msg("Funding nodes")

	var transferAmount int64 = 1000000 * 1000
	for _, nodeAddr := range chainlinkClient.GetNodeAddresses() {
		transferTx, err := grpcClient.Transfer(genesisAddress, nodeAddr, transferAmount)
		require.NoError(t, err, "Creation of Transfer Txn from genesis account to node failed")
		_, err = txmgr.SignAndBroadcast(context.Background(), genesisAddress, transferTx)
		require.NoError(t, err, "Broadcast of Transfer Txn from genesis account to node failed")
	}

	startTime := time.Now()

	// Check that the nodes have been funded
	for _, nodeAddr := range chainlinkClient.GetNodeAddresses() {
		for {
			accountInfo, err := grpcClient.GetAccount(nodeAddr)
			if err != nil {
				// do not error on 'account not found' - this occurs when there is no account info (transfer hasnt executed yet)
				if err.Error() == "account not found" {
					time.Sleep(time.Second)
					continue
				}
				logger.Error().Str("address", nodeAddr).Err(err)
				t.Fatal("failed to get account info")
			}

			if accountInfo.Balance != transferAmount {
				time.Sleep(time.Second)
				continue
			}

			// timeout
			if time.Since(startTime).Seconds() > 30 {
				t.Fatal("failed to fund nodes in time")
			}
			break
		}
		logger.Info().Str("address", nodeAddr).Msg("successfully funded")
	}

	httpUrl := os.Getenv("HTTP_URL")
	if httpUrl == "" {
		httpUrl = fmt.Sprintf("http://%s:%s", testutils.GetTronNodeIpAddress(), utils.HttpPort)
	}
	logger.Info().Str("http url", httpUrl).Msg("TRON json client config")
	deployContract := func(contractName string, artifact *contract.Artifact, params []interface{}) string {
		txHash := testutils.DeployContractByJson(t, httpUrl, testKeystore, genesisAddress, contractName, artifact.AbiJson, artifact.Bytecode, params)
		txInfo := testutils.WaitForTransactionInfo(t, grpcClient, txHash, 30)
		contractAddress := address.Address(txInfo.ContractAddress).String()
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
	// TODO: check constructor args
	eacAggregatorProxyAddress := deployContract("EACAggregatorProxy", eacAggregatorProxyArtifact, []interface{}{
		utils.MustConvertToEthAddress(t, ocr2AggregatorAddress),
		utils.MustConvertToEthAddress(t, requesterAccessControllerAddress),
	})
	logger.Info().Str("address", eacAggregatorProxyAddress).Msg("Aggregator proxy deployed")

	mintAmount := new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil)
	mintAmount = mintAmount.Mul(mintAmount, big.NewInt(50000))

	err = txmgr.Enqueue(genesisAddress, linkTokenAddress, "grantMintAndBurnRoles(address)", "address", genesisAddress)
	require.NoError(t, err)
	err = txmgr.Enqueue(genesisAddress, linkTokenAddress, "mint(address,uint256)", "address", ocr2AggregatorAddress, "uint256", mintAmount.String())
	require.NoError(t, err)
	testutils.WaitForInflightTxs(clientLogger, txmgr, time.Second*30)

	balanceResponse, err := grpcClient.TriggerConstantContract("", linkTokenAddress, "balanceOf(address)", []any{"address", ocr2AggregatorAddress})
	require.NoError(t, err)
	balanceValue := new(big.Int).SetBytes(balanceResponse.ConstantResult[0])
	require.Equal(t, balanceValue, mintAmount)
	logger.Info().Str("amount", mintAmount.String()).Msg("Minted LINK token")

	signers, transmitters, f, onchainConfig, offchainConfigVersion, offchainConfig := chainlinkClient.GetSetConfigArgs(t)

	//// Define the values
	onchainConfigBytes := []byte{}
	// version (uint8)
	onchainConfigBytes = append(onchainConfigBytes, byte(1))
	// minAnswer (int192)
	onchainConfigBytes = append(onchainConfigBytes, ethcommon.LeftPadBytes(minAnswer.Bytes(), 24)...)
	// maxAnswer (int192)
	onchainConfigBytes = append(onchainConfigBytes, ethcommon.LeftPadBytes(maxAnswer.Bytes(), 24)...)

	//// version 2 (OCR2OffchainConfigVersion)
	//offchainConfigVersion := "2"

	fmt.Printf("TEST SIGNERS: %+v - TRANSMITTERS: %+v\n", signers, transmitters)
	fmt.Printf("ONCHAIN CONFIG: %T %+v\n", onchainConfig, onchainConfig)
	fmt.Printf("OFFCHAIN CONFIG: %T %+v\n", offchainConfig, offchainConfig)

	signerAddresses := []string{}
	for _, signer := range signers {
		// TODO: gotron-sdk only supports base58 addresses as input for address or address[], update it so that
		// we can pass common.Address directly
		signerAddresses = append(signerAddresses, utils.EthereumToTronAddressBase58(ethcommon.BytesToAddress(signer)))
	}

	// TODO: should we set onchainConfig as offchainConfig?
	err = txmgr.Enqueue(genesisAddress, ocr2AggregatorAddress, "setConfig(address[],address[],uint8,bytes,uint64,bytes)",
		/* signers= */ "address[]", signerAddresses,
		/* trasmitters= */ "address[]", chainlinkClient.GetNodeAddresses(),
		/* f= */ "uint8", fmt.Sprintf("%d", f),
		/* onchainConfig= */ "bytes", onchainConfigBytes,
		/* offchainConfigVersion= */ "uint64", fmt.Sprintf("%d", offchainConfigVersion),
		/* offchainConfig= */ "bytes", offchainConfig)
	require.NoError(t, err)

	// TODO: we need to fix the txmgr from returning 0 inflight count when it's processing a single transaction with nothing queued.
	time.Sleep(time.Second)

	testutils.WaitForInflightTxs(clientLogger, txmgr, time.Second*30)

	configDetailsResponse, err := grpcClient.TriggerConstantContract("", ocr2AggregatorAddress, "latestConfigDetails()", nil)
	require.NoError(t, err)

	configCount := new(big.Int).SetBytes(configDetailsResponse.ConstantResult[0][0:32])
	require.NoError(t, err)
	require.Equal(t, configCount, big.NewInt(1))
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

	err = validateRounds(t, grpcClient, mustConvertAddress(t, ocr2AggregatorAddress), mustConvertAddress(t, eacAggregatorProxyAddress), commonConfig.IsSoak, commonConfig.TestDuration)
	require.NoError(t, err, "Validating round should not fail")

	// TODO: does this need to be reenabled?
	//
	// t.Cleanup(func() {
	// 	err = actions.TeardownSuite(t, commonConfig.Env, "./", nil, nil, zapcore.DPanicLevel, nil)
	// 	//err = actions.TeardownSuite(t, t.Common.Env, utils.ProjectRoot, t.Cc.ChainlinkNodes, nil, zapcore.ErrorLevel)
	// 	require.NoError(t, err, "Error tearing down environment")
	// })
}

func setupLocalStack(t *testing.T, logger zerolog.Logger, genesisAddress string) (sdk.GrpcClient, *common.ChainlinkClient, *common.Common) {
	grpcUrl := os.Getenv("GRPC_URL")
	if grpcUrl == "" {
		grpcUrl = fmt.Sprintf("grpc://%s:%s/?insecure=true", testutils.GetTronNodeIpAddress(), utils.GrpcPort)
	}
	solidityGrpcUrl := os.Getenv("SOLIDITY_GRPC_URL")
	if solidityGrpcUrl == "" {
		solidityGrpcUrl = fmt.Sprintf("grpc://%s:%s/?insecure=true", testutils.GetTronNodeIpAddress(), utils.GrpcSolidityPort)
	}
	internalGrpcUrl := os.Getenv("INTERNAL_GRPC_URL")
	if internalGrpcUrl == "" {
		internalGrpcUrl = utils.DefaultInternalGrpcUrl
	}
	internalSolidityUrl := os.Getenv("INTERNAL_SOLIDITY_URL")
	if internalSolidityUrl == "" {
		internalSolidityUrl = utils.DefaultInternalSolidityUrl
	}
	internalJsonRpcUrl := os.Getenv("INTERNAL_JSON_RPC_URL")
	if internalJsonRpcUrl == "" {
		internalJsonRpcUrl = utils.DefaultInternalJsonRpcUrl
	}

	logger.Info().Msg("Starting java-tron container...")
	err := testutils.StartTronNode(genesisAddress)
	require.NoError(t, err, "Could not start java-tron container")

	logger.Info().Str("grpc url", grpcUrl).Msg("TRON node config")

	grpcUrlObj, err := url.Parse(grpcUrl)
	require.NoError(t, err)
	solidityGrpcUrlObj, err := url.Parse(solidityGrpcUrl)
	require.NoError(t, err)

	grpcClient, err := sdk.CreateGrpcClient(grpcUrlObj, solidityGrpcUrlObj)
	require.NoError(t, err)

	blockInfo, err := grpcClient.GetBlockByNum(0)
	require.NoError(t, err)

	blockId := blockInfo.Blockid

	// previously, we took the whole genesis block id as the chain id, which is the case depending on java-tron node config:
	// https://github.com/tronprotocol/java-tron/blob/b1fc2f0f2bd79527099bc3027b9aba165c2e20c2/actuator/src/main/java/org/tron/core/vm/program/Program.java#L1271
	//
	// however on both mainnet, testnets committee.allowOptimizedReturnValueOfChainId is enabled, so we've done the same for devnet
	// and the last 4 bytes is the chain id both when retrieved by eth_chainId and via the `block.chainid` call in the TVM, which
	// is important for the config digest calculation:
	// https://github.com/smartcontractkit/libocr/blob/063ceef8c42eeadbe94221e55b8892690d36099a/contract2/OCR2Aggregator.sol#L27
	chainId := "0x" + hex.EncodeToString(blockId[len(blockId)-4:])

	logger.Info().Str("chain id", chainId).Msg("Read first block")

	commonConfig := common.NewCommon(t, chainId, internalGrpcUrl, internalSolidityUrl, internalJsonRpcUrl)
	commonConfig.SetLocalEnvironment(t, genesisAddress)

	chainlinkClient, err := common.NewChainlinkClient(commonConfig.Env, commonConfig.ChainId, utils.CLNodeName)
	require.NoError(t, err, "Could not create chainlink client")
	logger.Info().Str("node addresses", strings.Join(chainlinkClient.GetNodeAddresses(), " ")).Msg("Created chainlink client")
	return grpcClient, chainlinkClient, commonConfig
}

func teardownLocalStack(t *testing.T, logger zerolog.Logger, commonConfig *common.Common) {
	commonConfig.TearDownLocalEnvironment(t)
	logger.Info().Msg("Tearing down java-tron container...")
	err := testutils.StopTronNode()
	require.NoError(t, err, "Could not tear down java-tron container")
}

func mustConvertAddress(t *testing.T, tronAddress string) address.Address {
	a, err := address.Base58ToAddress(tronAddress)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func validateRounds(t *testing.T, grpcClient sdk.GrpcClient, ocrAddress address.Address, ocrProxyAddress address.Address, isSoak bool, testDuration time.Duration) error {
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

	readerClient := reader.NewReader(grpcClient, ocrLogger)
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
		time.Sleep(5 * time.Second)
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

	//// Test proxy reading
	// TODO(BCI-1746): dynamic mock-adapter values
	//mockAdapterValue := 5
	//// TODO: would be good to test proxy switching underlying feeds
	//resp, err = cosmosClient.ContractState(ocrProxyAddress, []byte(`{"latest_round_data":{}}`))
	//if !isSoak {
	//require.NoError(t, err, "Reading round data from proxy should not fail")
	//// assert.Equal(t, len(roundDataRaw), 5, "Round data from proxy should match expected size")
	//}
	//roundData := struct {
	//Answer string `json:"answer"`
	//}{}
	//err = json.Unmarshal(resp, &roundData)
	//require.NoError(t, err, "Failed to unmarshal round data")

	//valueBig, success := new(big.Int).SetString(roundData.Answer, 10)
	//require.True(t, success, "Failed to parse round data")
	//value := valueBig.Int64()
	//require.Equal(t, value, int64(mockAdapterValue), "Reading from proxy should return correct value")

	return nil
}
