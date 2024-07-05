package ocr2_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/rand"
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
	"github.com/stretchr/testify/require"

	relaylogger "github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/common"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/contract"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/utils"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
	// "github.com/smartcontractkit/chainlink/integration-tests/actions"
	// "go.uber.org/zap/zapcore"
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

	logger.Debug().Str("genesis address", genesisAddress).Msg("Using genesis account")
	commonConfig := common.NewCommon(t)
	commonConfig.SetLocalEnvironment(t, genesisAddress)

	clientLogger, err := relaylogger.New()
	require.NoError(t, err, "Could not create relay logger")

	logger.Debug().Str("grpc url", commonConfig.GrpcUrl).Str("http url", commonConfig.HttpUrl).Msg("Node config")
	grpcUrl, err := url.Parse(commonConfig.GrpcUrl)
	require.NoError(t, err)

	grpcClient, err := sdk.CreateGrpcClient(grpcUrl)
	require.NoError(t, err)

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
		txHash := testutils.DeployContractByJson(t, commonConfig.HttpUrl, testKeystore, genesisAddress, contractName, artifact.AbiJson, artifact.Bytecode, params)
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

	balanceResponse, err := grpcClient.TriggerConstantContract("", linkTokenAddress, "balanceOf(address)", utils.MustMarshalParams(t, "address", ocr2AggregatorAddress))
	require.NoError(t, err)
	balanceValue := new(big.Int).SetBytes(balanceResponse.ConstantResult[0])
	require.Equal(t, balanceValue, mintAmount)
	logger.Info().Str("amount", mintAmount.String()).Msg("Minted LINK token")

	// Define the values
	onchainConfigBytes := []byte{}
	// version (uint8)
	onchainConfigBytes = append(onchainConfigBytes, byte(1))
	// minAnswer (int192)
	onchainConfigBytes = append(onchainConfigBytes, ethcommon.LeftPadBytes(minAnswer.Bytes(), 24)...)
	// maxAnswer (int192)
	onchainConfigBytes = append(onchainConfigBytes, ethcommon.LeftPadBytes(maxAnswer.Bytes(), 24)...)

	// TODO: should we set onchainConfig as offchainConfig?
	err = txmgr.Enqueue(genesisAddress, ocr2AggregatorAddress, "setConfig(address[],address[],uint8,bytes,uint64,bytes)",
		/* signers= */ "address[]", chainlinkClient.GetNodeAddresses(),
		/* trasmitters= */ "address[]", chainlinkClient.GetNodeAddresses(),
		/* f= */ "uint8", "1",
		/* onchainConfig= */ "bytes", onchainConfigBytes,
		/* offchainConfigVersion= */ "uint64", "1",
		/* offchainConfig= */ "bytes", onchainConfigBytes)
	require.NoError(t, err)

	// TODO: we need to fix the txmgr from returning 0 inflight count when it's processing a single transaction with nothing queued.
	time.Sleep(time.Second)

	testutils.WaitForInflightTxs(clientLogger, txmgr, time.Second*30)

	configDetailsResponse, err := grpcClient.TriggerConstantContract("", ocr2AggregatorAddress, "latestConfigDetails()", "")
	require.NoError(t, err)

	configCount := new(big.Int).SetBytes(configDetailsResponse.ConstantResult[0][0:32])
	require.NoError(t, err)
	require.Equal(t, configCount, big.NewInt(1))
	logger.Info().Msg("successfully set config")

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

	//err = validateRounds(t, cosmosClient, types.MustAccAddressFromBech32(ocrAddress), types.MustAccAddressFromBech32(ocrProxyAddress), commonConfig.IsSoak, commonConfig.TestDuration)
	//require.NoError(t, err, "Validating round should not fail")

	// Tear down local stack
	commonConfig.TearDownLocalEnvironment(t)

	// t.Cleanup(func() {
	// 	err = actions.TeardownSuite(t, commonConfig.Env, "./", nil, nil, zapcore.DPanicLevel, nil)
	// 	//err = actions.TeardownSuite(t, t.Common.Env, utils.ProjectRoot, t.Cc.ChainlinkNodes, nil, zapcore.ErrorLevel)
	// 	require.NoError(t, err, "Error tearing down environment")
	// })
}

//func validateRounds(t *testing.T, cosmosClient *client.Client, ocrAddress types.AccAddress, ocrProxyAddress types.AccAddress, isSoak bool, testDuration time.Duration) error {
//var rounds int
//if isSoak {
//rounds = 99999999
//} else {
//rounds = 10
//}

//// TODO(BCI-1746): dynamic mock-adapter values
//mockAdapterValue := 5

//logger := common.GetTestLogger(t)
//ctx := context.Background() // context background used because timeout handled by requestTimeout param
//// assert new rounds are occurring
//increasing := 0 // track number of increasing rounds
//var stuck bool
//stuckCount := 0
//var positive bool
//resp, err := cosmosClient.ContractState(
//ocrAddress,
//[]byte(`{"link_available_for_payment":{}}`),
//)
//if err != nil {
//return err
//}

//linkResponse := struct {
//Amount string `json:"amount"`
//}{}
//if err = json.Unmarshal(resp, &linkResponse); err != nil {
//return err
//}
//logger.Info().Str("amount", linkResponse.Amount).Msg("Queried link available for payment")

//availableLink, success := new(big.Int).SetString(linkResponse.Amount, 10)
//require.True(t, success, "Could not convert link_available_for_payment response")
//require.True(t, availableLink.Cmp(big.NewInt(0)) > 0, "Aggregator should have non-zero balance")

//// TODO(BCI-1767): this needs to be able to support different readers
//ocrLogger, err := relaylogger.New()
//require.NoError(t, err, "Failed to create OCR relay logger")
//ocrReader := cosmwasm.NewOCR2Reader(ocrAddress, cosmosClient, ocrLogger)

//type TransmissionDetails struct {
//ConfigDigest    ocrtypes.ConfigDigest
//Epoch           uint32
//Round           uint8
//LatestAnswer    *big.Int
//LatestTimestamp time.Time
//}

//previous := TransmissionDetails{}

//for start := time.Now(); time.Since(start) < testDuration; {
//logger.Info().Msg(fmt.Sprintf("Elapsed time: %s, Round wait: %s ", time.Since(start), testDuration))
//configDigest, epoch, round, latestAnswer, latestTimestamp, err2 := ocrReader.LatestTransmissionDetails(ctx)
//require.NoError(t, err2, "Failed to get latest transmission details")
//// end condition: enough rounds have occurred
//if !isSoak && increasing >= rounds && positive {
//break
//}

//// end condition: rounds have been stuck
//if stuck && stuckCount > 50 {
//logger.Debug().Msg("failing to fetch transmissions means blockchain may have stopped")
//break
//}

//// try to fetch rounds
//time.Sleep(5 * time.Second)

//if err2 != nil {
//logger.Error().Msg(fmt.Sprintf("Transmission Error: %+v", err2))
//continue
//}
//logger.Info().Msg(fmt.Sprintf("Transmission Details: configDigest: %+v, epoch: %+v, round: %+v, latestAnswer: %+v, latestTimestamp: %+v", configDigest, epoch, round, latestAnswer, latestTimestamp))

//// continue if no changes
//if epoch == 0 && round == 0 {
//continue
//}

//ansCmp := latestAnswer.Cmp(big.NewInt(0))
//positive = ansCmp == 1 || positive

//// if changes from zero values set (should only initially)
//if epoch > 0 && previous.Epoch == 0 {
//if !isSoak {
//require.Greater(t, epoch, previous.Epoch)
//require.GreaterOrEqual(t, round, previous.Round)
//require.NotEqual(t, ansCmp, 0) // require changed from 0
//require.NotEqual(t, configDigest, previous.ConfigDigest)
//require.Equal(t, previous.LatestTimestamp.Before(latestTimestamp), true)
//}
//previous = TransmissionDetails{
//ConfigDigest:    configDigest,
//Epoch:           epoch,
//Round:           round,
//LatestAnswer:    latestAnswer,
//LatestTimestamp: latestTimestamp,
//}
//continue
//}
//// check increasing rounds
//if !isSoak {
//require.Equal(t, configDigest, previous.ConfigDigest, "Config digest should not change")
//} else {
//if configDigest != previous.ConfigDigest {
//logger.Error().Msg(fmt.Sprintf("Config digest should not change, expected %s got %s", previous.ConfigDigest, configDigest))
//}
//}
//if (epoch > previous.Epoch || (epoch == previous.Epoch && round > previous.Round)) && previous.LatestTimestamp.Before(latestTimestamp) {
//increasing++
//stuck = false
//stuckCount = 0 // reset counter
//continue
//}

//// reach this point, answer has not changed
//stuckCount++
//if stuckCount > 30 {
//stuck = true
//increasing = 0
//}
//}
//if !isSoak {
//require.GreaterOrEqual(t, increasing, rounds, "Round + epochs should be increasing")
//require.Equal(t, positive, true, "Positive value should have been submitted")
//require.Equal(t, stuck, false, "Round + epochs should not be stuck")
//}

//// Test proxy reading
//// TODO: would be good to test proxy switching underlying feeds
//resp, err = cosmosClient.ContractState(ocrProxyAddress, []byte(`{"latest_round_data":{}}`))
//if !isSoak {
//require.NoError(t, err, "Reading round data from proxy should not fail")
////assert.Equal(t, len(roundDataRaw), 5, "Round data from proxy should match expected size")
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

//return nil
//}
