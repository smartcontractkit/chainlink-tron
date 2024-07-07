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
	"github.com/fbsobreira/gotron-sdk/pkg/client"
	"github.com/stretchr/testify/require"

	relaylogger "github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/common"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/contract"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/utils"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/ocr2"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/reader"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
	// "github.com/smartcontractkit/chainlink/integration-tests/actions"
	// "go.uber.org/zap/zapcore"
)

const (
	defaultInternalGrpcUrl = "grpc://host.docker.internal:16669/?insecure=true"
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

	grpcUrl := os.Getenv("GRPC_URL")
	if grpcUrl == "" {
		grpcUrl = fmt.Sprintf("grpc://%s:16669/?insecure=true", testutils.GetTronNodeIpAddress())
	}
	httpUrl := os.Getenv("HTTP_URL")
	if httpUrl == "" {
		httpUrl = fmt.Sprintf("http://%s:16667", testutils.GetTronNodeIpAddress())
	}
	internalGrpcUrl := os.Getenv("INTERNAL_GRPC_URL")
	if internalGrpcUrl == "" {
		internalGrpcUrl = defaultInternalGrpcUrl
	}

	logger.Info().Msg("Starting java-tron container...")
	err := testutils.StartTronNode(genesisAddress)
	require.NoError(t, err, "Could not start java-tron container")

	logger.Info().Str("grpc url", grpcUrl).Str("http url", httpUrl).Msg("TRON node config")

	grpcUrlObj, err := url.Parse(grpcUrl)
	require.NoError(t, err)

	grpcClient, err := sdk.CreateGrpcClient(grpcUrlObj)
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

	commonConfig := common.NewCommon(t, chainId, internalGrpcUrl)
	commonConfig.SetLocalEnvironment(t, genesisAddress)

	clientLogger, err := relaylogger.New()
	require.NoError(t, err, "Could not create relay logger")

	testKeystore := testutils.NewTestKeystore(genesisAddress, genesisPrivateKey)
	txmgr := txm.New(clientLogger, testKeystore, grpcClient, txm.TronTxmConfig{
		BroadcastChanSize: 100,
		ConfirmPollSecs:   2,
	})
	err = txmgr.Start(context.Background())
	require.NoError(t, err)

	nodeName := "primary"
	chainlinkClient, err := common.NewChainlinkClient(commonConfig.Env, commonConfig.ChainId, nodeName)
	require.NoError(t, err, "Could not create chainlink client")

	logger.Info().Str("node addresses", strings.Join(chainlinkClient.GetNodeAddresses(), " ")).Msg("Created chainlink client")
	require.NoError(t, err, "Could not create private key from mnemonic")
	logger.Info().Str("from", genesisAddress).Msg("Funding nodes")

	var transferAmount int64 = 1000000 * 1000
	for _, nodeAddr := range chainlinkClient.GetNodeAddresses() {
		transferTx, err := grpcClient.Transfer(genesisAddress, nodeAddr, transferAmount)
		require.NoError(t, err)
		_, err = txmgr.SignAndBroadcast(context.Background(), genesisAddress, transferTx)
		require.NoError(t, err)
	}

	startTime := time.Now()
	for _, nodeAddr := range chainlinkClient.GetNodeAddresses() {
		for {
			accountInfo, err := grpcClient.GetAccount(nodeAddr)
			require.NoError(t, err)

			if accountInfo.Balance != transferAmount {
				if time.Since(startTime).Seconds() > 30 {
					t.Fatal("failed to fund nodes in time")
				}
				time.Sleep(time.Second)
			}
			break
		}
		logger.Info().Str("address", nodeAddr).Msg("successfully funded")
	}

	deployContract := func(contractName string, artifact *contract.Artifact, params []interface{}) string {
		txHash := testutils.DeployContractByJson(t, httpUrl, testKeystore, genesisAddress, contractName, artifact.AbiJson, artifact.Bytecode, params)
		txInfo := testutils.WaitForTransactionInfo(t, grpcClient, txHash, 30)
		contractAddress := address.Address(txInfo.ContractAddress).String()
		return contractAddress
	}

	linkTokenArtifact := contract.MustLoadArtifact(t, "link-v0.8/LinkToken.json")
	linkTokenAddress := deployContract("LinkToken", linkTokenArtifact, nil)
	logger.Info().Str("address", linkTokenAddress).Msg("Link token contract deployed")

	readAccessControllerArtifact := contract.MustLoadArtifact(t, "datafeeds-v0.6/SimpleReadAccessController.json")
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
	ocr2AggregatorArtifact := contract.MustLoadArtifact(t, "datafeeds-v0.6/OCR2Aggregator.json")
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

	eacAggregatorProxyArtifact := contract.MustLoadArtifact(t, "datafeeds-v0.6/EACAggregatorProxy.json")
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
		nodeName,
		p2pPort,
		commonConfig.MockUrl,
		commonConfig.JuelsPerFeeCoinSource,
		ocr2AggregatorAddress)
	require.NoError(t, err, "Could not create jobs for contract")

	//gauntletWorkingDir := fmt.Sprintf("%s/", utils.ProjectRoot)
	//logger.Info().Str("working dir", gauntletWorkingDir).Msg("Initializing gauntlet")

	//cg, err := gauntlet.NewCosmosGauntlet(gauntletWorkingDir)
	//require.NoError(t, err, "Could not create cosmos gauntlet")

	//err = cg.InstallDependencies()
	//require.NoError(t, err, "Failed to install gauntlet dependencies")

	//err = cg.SetupNetwork(commonConfig.GrpcUrl, commonConfig.Mnemonic)
	//require.NoError(t, err, "Setting up gauntlet network should not fail")

	//// Upload contracts
	//_, err = cg.UploadContracts(nil)
	//require.NoError(t, err, "Could not upload contracts")

	//// Deploy contracts
	//linkTokenAddress, err := cg.DeployLinkTokenContract()
	//require.NoError(t, err, "Could not deploy link token contract")
	//logger.Info().Str("address", linkTokenAddress).Msg("Deployed LINK token")
	//os.Setenv("LINK", linkTokenAddress)

	//billingAccessControllerAddress, err := cg.DeployAccessControllerContract()
	//require.NoError(t, err, "Could not deploy billing access controller")
	//logger.Info().Str("address", billingAccessControllerAddress).Msg("Deployed billing access controller")
	//os.Setenv("BILLING_ACCESS_CONTROLLER", billingAccessControllerAddress)

	//requesterAccessControllerAddress, err := cg.DeployAccessControllerContract()
	//require.NoError(t, err, "Could not deploy requester access controller")
	//logger.Info().Str("address", requesterAccessControllerAddress).Msg("Deployed requester access controller")
	//os.Setenv("REQUESTER_ACCESS_CONTROLLER", requesterAccessControllerAddress)

	//minSubmissionValue := int64(0)
	//maxSubmissionValue := int64(100000000000)
	//decimals := 9
	//name := "auto"
	//ocrAddress, err := cg.DeployOCR2ControllerContract(minSubmissionValue, maxSubmissionValue, decimals, name, linkTokenAddress)
	//require.NoError(t, err, "Could not deploy OCR2 controller contract")
	//logger.Info().Str("address", ocrAddress).Msg("Deployed OCR2 Controller contract")

	//ocrProxyAddress, err := cg.DeployOCR2ProxyContract(ocrAddress)
	//require.NoError(t, err, "Could not deploy OCR2 proxy contract")
	//logger.Info().Str("address", ocrProxyAddress).Msg("Deployed OCR2 proxy contract")

	//// Mint LINK tokens to aggregator
	//_, err = cg.MintLinkToken(linkTokenAddress, ocrAddress, "100000000000000000000")
	//require.NoError(t, err, "Could not mint LINK token")

	//// Set OCR2 Billing
	//observationPaymentGjuels := int64(1)
	//transmissionPaymentGjuels := int64(1)
	//recommendedGasPriceMicro := "1"
	//_, err = cg.SetOCRBilling(observationPaymentGjuels, transmissionPaymentGjuels, recommendedGasPriceMicro, ocrAddress)
	//require.NoError(t, err, "Could not set OCR billing")

	//// OCR2 Config Proposal
	//proposalId, err := cg.BeginProposal(ocrAddress)
	//require.NoError(t, err, "Could not begin proposal")

	//cfg, err := chainlinkClient.LoadOCR2Config(proposalId)
	//require.NoError(t, err, "Could not load OCR2 config")

	//var parsedConfig []byte
	//parsedConfig, err = json.Marshal(cfg)
	//require.NoError(t, err, "Could not parse JSON config")

	//_, err = cg.ProposeConfig(string(parsedConfig), ocrAddress)
	//require.NoError(t, err, "Could not propose config")

	//_, err = cg.ProposeOffchainConfig(string(parsedConfig), ocrAddress)
	//require.NoError(t, err, "Could not propose offchain config")

	//digest, err := cg.FinalizeProposal(proposalId, ocrAddress)
	//require.NoError(t, err, "Could not finalize proposal")

	//var acceptProposalInput = struct {
	//ProposalId     string            `json:"proposalId"`
	//Digest         string            `json:"digest"`
	//OffchainConfig common.OCR2Config `json:"offchainConfig"`
	//RandomSecret   string            `json:"randomSecret"`
	//}{
	//ProposalId:     proposalId,
	//Digest:         digest,
	//OffchainConfig: *cfg,
	//RandomSecret:   cfg.Secret,
	//}
	//var parsedInput []byte
	//parsedInput, err = json.Marshal(acceptProposalInput)
	//require.NoError(t, err, "Could not parse JSON input")
	//_, err = cg.AcceptProposal(string(parsedInput), ocrAddress)
	//require.NoError(t, err, "Could not accept proposed config")

	//p2pPort := "50200"
	//err = chainlinkClient.CreateJobsForContract(
	//commonConfig.ChainId,
	//nodeName,
	//p2pPort,
	//commonConfig.MockUrl,
	//commonConfig.JuelsPerFeeCoinSource,
	//ocrAddress)
	//require.NoError(t, err, "Could not create jobs for contract")

	err = validateRounds(t, grpcClient, mustConvertAddress(t, ocr2AggregatorAddress), mustConvertAddress(t, eacAggregatorProxyAddress), commonConfig.IsSoak, commonConfig.TestDuration)
	require.NoError(t, err, "Validating round should not fail")

	// Tear down local stack
	commonConfig.TearDownLocalEnvironment(t)

	logger.Info().Msg("Tearing down java-tron container...")
	err = testutils.StopTronNode()
	require.NoError(t, err, "Could not tear down java-tron container")

	// t.Cleanup(func() {
	// 	err = actions.TeardownSuite(t, commonConfig.Env, "./", nil, nil, zapcore.DPanicLevel, nil)
	// 	//err = actions.TeardownSuite(t, t.Common.Env, utils.ProjectRoot, t.Cc.ChainlinkNodes, nil, zapcore.ErrorLevel)
	// 	require.NoError(t, err, "Error tearing down environment")
	// })
}

func mustConvertAddress(t *testing.T, tronAddress string) address.Address {
	a, err := address.Base58ToAddress(tronAddress)
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func validateRounds(t *testing.T, grpcClient *client.GrpcClient, ocrAddress address.Address, ocrProxyAddress address.Address, isSoak bool, testDuration time.Duration) error {
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
	var positive bool

	ocrLogger, err := relaylogger.New()
	require.NoError(t, err, "Failed to create OCR relay logger")

	readerClient := reader.NewReader(grpcClient, ocrLogger)
	ocrReader := ocr2.NewOCR2Reader(readerClient, ocrLogger)

	previous := ocr2.TransmissionDetails{}

	for start := time.Now(); time.Since(start) < testDuration; {
		logger.Info().Msg(fmt.Sprintf("Elapsed time: %s, Round wait: %s ", time.Since(start), testDuration))
		current, err2 := ocrReader.LatestTransmissionDetails(ctx, ocrAddress)
		require.NoError(t, err2, "Failed to get latest transmission details")
		// end condition: enough rounds have occurred
		if !isSoak && increasing >= rounds && positive {
			break
		}

		// end condition: rounds have been stuck
		if stuck && stuckCount > 50 {
			logger.Debug().Msg("failing to fetch transmissions means blockchain may have stopped")
			break
		}

		// try to fetch rounds
		time.Sleep(5 * time.Second)

		if err2 != nil {
			logger.Error().Msg(fmt.Sprintf("Transmission Error: %+v", err2))
			continue
		}
		logger.Info().Msg(fmt.Sprintf("Transmission Details: %+v", current))

		// continue if no changes
		if current.Epoch == 0 && current.Round == 0 {
			continue
		}

		ansCmp := current.LatestAnswer.Cmp(big.NewInt(0))
		positive = ansCmp == 1 || positive

		// if changes from zero values set (should only initially)
		if current.Epoch > 0 && previous.Epoch == 0 {
			if !isSoak {
				require.Greater(t, current.Epoch, previous.Epoch)
				require.GreaterOrEqual(t, current.Round, previous.Round)
				require.NotEqual(t, ansCmp, 0) // require changed from 0
				require.NotEqual(t, current.Digest, previous.Digest)
				require.Equal(t, previous.LatestTimestamp.Before(current.LatestTimestamp), true)
			}
			previous = current
			continue
		}
		// check increasing rounds
		if !isSoak {
			require.Equal(t, current.Digest, previous.Digest, "Config digest should not change")
		} else {
			if current.Digest != previous.Digest {
				logger.Error().Msg(fmt.Sprintf("Config digest should not change, expected %s got %s", previous.Digest, current.Digest))
			}
		}
		if (current.Epoch > previous.Epoch || (current.Epoch == previous.Epoch && current.Round > previous.Round)) && previous.LatestTimestamp.Before(current.LatestTimestamp) {
			increasing++
			stuck = false
			stuckCount = 0 // reset counter
			continue
		}

		// reach this point, answer has not changed
		stuckCount++
		if stuckCount > 30 {
			stuck = true
			increasing = 0
		}
	}
	if !isSoak {
		require.GreaterOrEqual(t, increasing, rounds, "Round + epochs should be increasing")
		require.Equal(t, positive, true, "Positive value should have been submitted")
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
