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

	"github.com/ethereum/go-ethereum/common"
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
)

type ChainlinkClient struct {
	ChainlinkNodes []*client.ChainlinkClient
	NodeKeys       []client.NodeKeysBundle
	bTypeAttr      *client.BridgeTypeAttributes
	bootstrapPeers []client.P2PData
}

var _ ChainlinkClient = ChainlinkClient{bTypeAttr: nil} // fix "field `bTypeAttr` is unused" lint

// NewChainlinkClient creates node keys and defines chain and nodes for each node
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

func (cc *ChainlinkClient) GetNodeAddresses() []address.Address {
	var addresses []address.Address
	for _, nodeKey := range cc.NodeKeys {
		nodeAddress, err := address.StringToAddress(nodeKey.TXKey.Data.ID)
		if err != nil {
			panic(err)
		}
		addresses = append(addresses, nodeAddress)
	}
	return addresses
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

		// this is the OCR2 report signing public key, but in ethereum address format already.
		// ref: https://github.com/smartcontractkit/chainlink/blob/286b02739a8638be0d8d5cd8673da18fb207a1ed/core/services/keystore/keys/ocr2key/evm_keyring.go#L31
		onchainPubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(key.OCR2Key.Data.Attributes.OnChainPublicKey, "ocr2on_tron_"))
		if err != nil {
			t.Fatal(err)
		}
		if len(onchainPubKeyBytes) != common.AddressLength {
			t.Fatal(fmt.Sprintf("Invalid offchain public key length. Expected %d bytes, got %d bytes", common.AddressLength, len(onchainPubKeyBytes)))
			return
		}

		var offchainPubKey [ed25519.PublicKeySize]byte
		copy(offchainPubKey[:], offchainPubKeyBytes)

		configPubKeyBytes, err := hex.DecodeString(strings.TrimPrefix(key.OCR2Key.Data.Attributes.ConfigPublicKey, "ocr2cfg_tron_"))
		if err != nil {
			t.Fatal(err)
		}
		if len(configPubKeyBytes) != curve25519.PointSize {
			t.Fatal(fmt.Sprintf("Invalid config public key length. Expected %d bytes, got %d bytes", curve25519.PointSize, len(configPubKeyBytes)))
			return
		}

		var configPubKey [curve25519.PointSize]byte
		copy(configPubKey[:], configPubKeyBytes)

		// Convert TRON base58 transmitter address to hex format in tests for computing the offchain config hash
		transmitterAddress, err := address.StringToAddress(key.TXKey.Data.ID)
		if err != nil {
			t.Fatal(err)
		}

		oracleIdentity := confighelper.OracleIdentity{
			OffchainPublicKey: offchainPubKey,
			OnchainPublicKey:  onchainPubKeyBytes,
			PeerID:            key.PeerID,
			TransmitAccount:   types.Account(transmitterAddress.EthAddress().Hex()),
		}

		oracleIdentityExtra := confighelper.OracleIdentityExtra{
			OracleIdentity:            oracleIdentity,
			ConfigEncryptionPublicKey: configPubKey,
		}

		oracleIdentities = append(oracleIdentities, oracleIdentityExtra)
	}

	var err error
	signers, transmitters, f, onchainConfig, offchainConfigVersion, offchainConfig, err = confighelper.ContractSetConfigArgsForTests(
		60*time.Second,   // deltaProgress time.Duration,
		30*time.Second,   // deltaResend time.Duration,
		10*time.Second,   // deltaRound time.Duration,
		5*time.Second,    // deltaGrace time.Duration,
		60*time.Second,   // deltaStage time.Duration,
		6,                // rMax uint8,
		S,                // s []int,
		oracleIdentities, // oracles []OracleIdentityExtra,
		median.OffchainConfig{
			AlphaReportInfinite: false,
			AlphaReportPPB:      5000000,
			AlphaAcceptInfinite: false,
			AlphaAcceptPPB:      5000000,
			DeltaC:              time.Hour * 24,
		}.Encode(), // reportingPluginConfig []byte,
		nil,            // maxDurationInitialization *time.Duration,
		2*time.Second,  // maxDurationQuery time.Duration,
		12*time.Second, // maxDurationObservation time.Duration,
		20*time.Second, // maxDurationReport time.Duration,
		20*time.Second, // maxDurationShouldAcceptFinalizedReport time.Duration,
		20*time.Second, // maxDurationShouldTransmitAcceptedReport time.Duration,
		1,              // f int,
		nil,            // The median reporting plugin has an empty onchain config
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
		URL:         fmt.Sprintf("%s/%s", mockUrl, "random"),
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
