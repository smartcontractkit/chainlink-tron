package common

import (
	"crypto/ed25519"
	"encoding/hex"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/curve25519"
	"gopkg.in/guregu/null.v4"

	"github.com/smartcontractkit/chainlink-testing-framework/k8s/environment"
	"github.com/smartcontractkit/chainlink/integration-tests/client"
	"github.com/smartcontractkit/chainlink/v2/core/services/job"
	"github.com/smartcontractkit/chainlink/v2/core/services/relay"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/confighelper"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/utils"
)

type ChainlinkClient struct {
	ChainlinkNodes []*client.ChainlinkClient
	NodeKeys       []client.NodeKeysBundle
	bTypeAttr      *client.BridgeTypeAttributes
	bootstrapPeers []client.P2PData
}

var _ ChainlinkClient = ChainlinkClient{bTypeAttr: nil} // fix "field `bTypeAttr` is unused" lint

// CreateKeys Creates node keys and defines chain and nodes for each node
func NewChainlinkClient(env *environment.Environment, nodeName string, chainId string) (*ChainlinkClient, error) {
	nodes, err := connectChainlinkNodes(env)
	if err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, errors.New("No connected nodes")
	}

	nodeKeys, _, err := client.CreateNodeKeysBundle(nodes, chainName, chainId)
	if err != nil {
		return nil, err
	}

	if len(nodeKeys) == 0 {
		return nil, errors.New("No node keys")
	}

	return &ChainlinkClient{
		ChainlinkNodes: nodes,
		NodeKeys:       nodeKeys,
	}, nil
}

func (cc *ChainlinkClient) GetNodeAddresses() []string {
	var addresses []string
	for _, nodeKey := range cc.NodeKeys {
		addresses = append(addresses, nodeKey.TXKey.Data.ID)
	}
	return addresses
}

func (cc *ChainlinkClient) LoadOCR2Config(proposalId string) (*OCR2Config, error) {
	var offChainKeys []string
	var onChainKeys []string
	var peerIds []string
	var cfgKeys []string
	for _, key := range cc.NodeKeys {
		offChainKeys = append(offChainKeys, key.OCR2Key.Data.Attributes.OffChainPublicKey)
		peerIds = append(peerIds, key.PeerID)
		onChainKeys = append(onChainKeys, key.OCR2Key.Data.Attributes.OnChainPublicKey)
		cfgKeys = append(cfgKeys, key.OCR2Key.Data.Attributes.ConfigPublicKey)
	}
	var payload = TestOCR2Config
	payload.ProposalId = proposalId
	payload.Signers = onChainKeys
	addresses := cc.GetNodeAddresses()
	payload.Transmitters = addresses
	payload.Payees = addresses // Set payees to same addresses as transmitters
	payload.OffchainConfig.OffchainPublicKeys = offChainKeys
	payload.OffchainConfig.PeerIds = peerIds
	payload.OffchainConfig.ConfigPublicKeys = cfgKeys
	return &payload, nil
}

func (cc *ChainlinkClient) GetSetConfigArgs(t *testing.T) (
	signers []types.OnchainPublicKey,
	transmitters []types.Account,
	f uint8,
	onchainConfig []byte,
	offchainConfigVersion uint64,
	offchainConfig []byte,
) {
	oracleIdentities := []confighelper.OracleIdentityExtra{}
	S := []int{}

	for _, key := range cc.NodeKeys {
		S = append(S, 1)

		offchainPubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(key.OCR2Key.Data.Attributes.OffChainPublicKey, "ocr2off_tron_"))
		if err != nil {
			t.Fatal(err)
		}

		// Check if the decoded bytes have the correct length for an Ed25519 public key
		if len(offchainPubKeyBytes) != ed25519.PublicKeySize {
			t.Fatal(fmt.Sprintf("Invalid offchain public key length. Expected %d bytes, got %d bytes", ed25519.PublicKeySize, len(offchainPubKeyBytes)))
			return
		}

		var offchainPubKey [ed25519.PublicKeySize]byte
		copy(offchainPubKey[:], offchainPubKeyBytes)

		ethAddress := utils.MustConvertToEthAddress(t, key.TXKey.Data.ID)

		configPubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(key.OCR2Key.Data.Attributes.ConfigPublicKey, "ocr2cfg_tron_"))
		if len(configPubKeyBytes) != curve25519.PointSize {
			t.Fatal(fmt.Sprintf("Invalid config public key length. Expected %d bytes, got %d bytes", curve25519.PointSize, len(configPubKeyBytes)))
			return
		}

		var configPubKey [curve25519.PointSize]byte
		copy(configPubKey[:], configPubKeyBytes)

		oracleIdentity := confighelper.OracleIdentity{
			OffchainPublicKey: offchainPubKey,
			// this is an address for EVM
			// https://github.com/smartcontractkit/libocr/blob/063ceef8c42eeadbe94221e55b8892690d36099a/offchainreporting2plus/confighelper/confighelper.go#L23
			OnchainPublicKey: ethAddress.Bytes(),
			PeerID:           key.PeerID,
			TransmitAccount:  types.Account(key.TXKey.Data.ID),
		}

		oracleIdentityExtra := confighelper.OracleIdentityExtra{
			OracleIdentity:            oracleIdentity,
			ConfigEncryptionPublicKey: configPubKey,
		}

		oracleIdentities = append(oracleIdentities, oracleIdentityExtra)
	}

	var err error
	signers, transmitters, f, onchainConfig, offchainConfigVersion, offchainConfig, err = confighelper.ContractSetConfigArgsForTests(
		30*time.Second,   // deltaProgress time.Duration,
		30*time.Second,   // deltaResend time.Duration,
		10*time.Second,   // deltaRound time.Duration,
		20*time.Second,   // deltaGrace time.Duration,
		20*time.Second,   // deltaStage time.Duration,
		3,                // rMax uint8,
		S,                // s []int,
		oracleIdentities, // oracles []OracleIdentityExtra,
		median.OffchainConfig{
			AlphaReportInfinite: false,
			AlphaReportPPB:      1,
			AlphaAcceptInfinite: false,
			AlphaAcceptPPB:      1,
			DeltaC:              time.Minute * 30,
		}.Encode(), // reportingPluginConfig []byte,
		5*time.Second, // maxDurationQuery time.Duration,
		5*time.Second, // maxDurationObservation time.Duration,
		5*time.Second, // maxDurationReport time.Duration,
		5*time.Second, // maxDurationShouldAcceptFinalizedReport time.Duration,
		5*time.Second, // maxDurationShouldTransmitAcceptedReport time.Duration,
		1,             // f int,
		nil,           // The median reporting plugin has an empty onchain config
	)

	if err != nil {
		t.Fatal(err)
	}

	return
}

// CreateJobsForContract Creates and sets up the boostrap jobs as well as OCR jobs
func (cc *ChainlinkClient) CreateJobsForContract(chainId, nodeName, p2pPort, mockUrl string, juelsPerFeeCoinSource string, ocrControllerAddress string) error {
	// Define node[0] as bootstrap node
	cc.bootstrapPeers = []client.P2PData{
		{
			InternalIP:   cc.ChainlinkNodes[0].InternalIP(),
			InternalPort: p2pPort,
			PeerID:       cc.NodeKeys[0].PeerID,
		},
	}

	// Defining relay config
	bootstrapRelayConfig := job.JSONConfig{
		"nodeName": nodeName,
		"chainID":  chainId,
	}

	oracleSpec := job.OCR2OracleSpec{
		ContractID:                  ocrControllerAddress,
		Relay:                       relay.NetworkTron,
		RelayConfig:                 bootstrapRelayConfig,
		ContractConfigConfirmations: 1, // don't wait for confirmation on devnet
	}
	// Setting up bootstrap node
	jobSpec := &client.OCR2TaskJobSpec{
		Name:           fmt.Sprintf("tron-OCRv2-%s-%s", "bootstrap", uuid.NewString()),
		JobType:        "bootstrap",
		OCR2OracleSpec: oracleSpec,
	}

	_, err := cc.ChainlinkNodes[0].MustCreateJob(jobSpec)
	if err != nil {
		return err
	}

	var p2pBootstrappers []string

	for i := range cc.bootstrapPeers {
		p2pBootstrappers = append(p2pBootstrappers, cc.bootstrapPeers[i].P2PV2Bootstrapper())
	}

	sourceValueBridge := &client.BridgeTypeAttributes{
		Name:        "mockserver-bridge",
		URL:         fmt.Sprintf("%s/%s", mockUrl, "five"),
		RequestData: "{}",
	}

	// Setting up job specs
	for nIdx, n := range cc.ChainlinkNodes {
		if nIdx == 0 {
			continue
		}
		_, err := n.CreateBridge(sourceValueBridge)
		if err != nil {
			return err
		}
		relayConfig := job.JSONConfig{
			"nodeName": bootstrapRelayConfig["nodeName"],
			"chainID":  bootstrapRelayConfig["chainID"],
		}

		oracleSpec = job.OCR2OracleSpec{
			ContractID:                  ocrControllerAddress,
			Relay:                       relay.NetworkTron,
			RelayConfig:                 relayConfig,
			PluginType:                  "median",
			OCRKeyBundleID:              null.StringFrom(cc.NodeKeys[nIdx].OCR2Key.Data.ID),
			TransmitterID:               null.StringFrom(cc.NodeKeys[nIdx].TXKey.Data.ID),
			P2PV2Bootstrappers:          pq.StringArray{strings.Join(p2pBootstrappers, ",")},
			ContractConfigConfirmations: 1, // don't wait for confirmation on devnet
			PluginConfig: job.JSONConfig{
				"juelsPerFeeCoinSource": juelsPerFeeCoinSource,
			},
		}

		jobSpec = &client.OCR2TaskJobSpec{
			Name:              fmt.Sprintf("tron-OCRv2-%d-%s", nIdx, uuid.NewString()),
			JobType:           "offchainreporting2",
			OCR2OracleSpec:    oracleSpec,
			ObservationSource: client.ObservationSourceSpecBridge(sourceValueBridge),
		}

		_, err = n.MustCreateJob(jobSpec)
		if err != nil {
			return err
		}
	}
	return nil
}

// connectChainlinkNodes creates a chainlink client for each node in the environment
// This is a non k8s version of the function in chainlink_k8s.go
func connectChainlinkNodes(e *environment.Environment) ([]*client.ChainlinkClient, error) {
	var clients []*client.ChainlinkClient
	for _, nodeDetails := range e.ChainlinkNodeDetails {
		c, err := client.NewChainlinkClient(&client.ChainlinkConfig{
			URL:        nodeDetails.LocalIP,
			Email:      "notreal@fakeemail.ch",
			Password:   "fj293fbBnlQ!f9vNs",
			InternalIP: parseHostname(nodeDetails.InternalIP),
		}, log.Logger)
		if err != nil {
			return nil, err
		}
		log.Debug().
			Str("URL", c.Config.URL).
			Str("Internal IP", c.Config.InternalIP).
			Str("Chart Name", nodeDetails.ChartName).
			Str("Pod Name", nodeDetails.PodName).
			Msg("Connected to Chainlink node")
		clients = append(clients, c)
	}
	return clients, nil
}

func parseHostname(s string) string {
	r := regexp.MustCompile(`://(?P<Host>.*):`)
	return r.FindStringSubmatch(s)[1]
}

func createNodeAddress(publicKeyHex string) string {
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		panic(err)
	}
	publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		panic(err)
	}
	return address.PubkeyToAddress(*publicKey).String()
}
