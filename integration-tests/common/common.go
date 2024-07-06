package common

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-testing-framework/k8s/environment"
	"github.com/smartcontractkit/chainlink-testing-framework/k8s/pkg/alias"
)

// TODO: those should be moved as a common part of chainlink-testing-framework

const (
	chainName             = "tron"
	ChainBlockTime        = "200ms"
	ChainBlockTimeSoak    = "2s"
	defaultTTLValue       = "1m"
	defaultNodeCountValue = "4"
)

var (
	observationSource = `
			val [type="bridge" name="bridge-coinmetrics" requestData=<{"data": {"from":"LINK","to":"USD"}}>]
			parse [type="jsonparse" path="result"]
			val -> parse
			`
	juelsPerFeeCoinSource = `"""
			sum  [type="sum" values=<[451000]> ]
			sum
			"""
			`
)

type Common struct {
	IsSoak                bool
	P2PPort               string
	ChainName             string
	ChainId               string
	NodeCount             int
	TTL                   time.Duration
	TestDuration          time.Duration
	MockUrl               string
	Mnemonic              string
	ObservationSource     string
	JuelsPerFeeCoinSource string
	ChainlinkConfig       string
	Env                   *environment.Environment
}

// getEnv gets the environment variable if it exists and sets it for the remote runner
func getEnv(v string) string {
	val := os.Getenv(v)
	if val != "" {
		os.Setenv(fmt.Sprintf("TEST_%s", v), val)
	}
	return val
}

func getNodeCount() int {
	// Checking if count of OCR nodes is defined in ENV
	nodeCountSet := getEnv("NODE_COUNT")
	if nodeCountSet == "" {
		nodeCountSet = defaultNodeCountValue
	}
	nodeCount, err := strconv.Atoi(nodeCountSet)
	if err != nil {
		panic(fmt.Sprintf("Please define a proper node count for the test: %v", err))
	}
	return nodeCount
}

func getTTL() time.Duration {
	ttlValue := getEnv("TTL")
	if ttlValue == "" {
		ttlValue = defaultTTLValue
	}
	duration, err := time.ParseDuration(ttlValue)
	if err != nil {
		panic(fmt.Sprintf("Please define a proper TTL for the test: %v", err))
	}
	t, err := time.ParseDuration(*alias.ShortDur(duration))
	if err != nil {
		panic(fmt.Sprintf("Please define a proper TTL for the test: %v", err))
	}
	return t
}

func getTestDuration() time.Duration {
	testDurationValue := getEnv("TEST_DURATION")
	if testDurationValue == "" {
		return time.Duration(time.Minute * 15)
	}
	duration, err := time.ParseDuration(testDurationValue)
	if err != nil {
		panic(fmt.Sprintf("Please define a proper duration for the test: %v", err))
	}
	t, err := time.ParseDuration(*alias.ShortDur(duration))
	if err != nil {
		panic(fmt.Sprintf("Please define a proper duration for the test: %v", err))
	}
	return t
}

func NewCommon(t *testing.T, chainID, internalGrpcUrl string) *Common {
	chainlinkConfig := fmt.Sprintf(`
[[Tron]]
Enabled = true
ChainID = '%s'

[[Tron.Nodes]]
Name = 'primary'
URL = '%s'

[OCR2]
Enabled = true

[P2P]
[P2P.V2]
Enabled = true
DeltaDial = '5s'
DeltaReconcile = '5s'
ListenAddresses = ['0.0.0.0:6691']

[WebServer]
HTTPPort = 6688
[WebServer.TLS]
HTTPSPort = 0
`, chainID, internalGrpcUrl)
	log.Debug().Str("toml", chainlinkConfig).Msg("Created chainlink config")

	ttl := getTTL()

	envConfig := &environment.Config{
		NamespacePrefix: "tron-ocr",
		TTL:             ttl,
		Test:            t,
	}
	c := &Common{
		IsSoak:                getEnv("SOAK") != "",
		ChainName:             chainName,
		ChainId:               chainID,
		NodeCount:             getNodeCount(),
		TTL:                   getTTL(),
		TestDuration:          getTestDuration(),
		MockUrl:               "http://host.docker.internal:6060",
		Mnemonic:              getEnv("MNEMONIC"),
		ObservationSource:     observationSource,
		JuelsPerFeeCoinSource: juelsPerFeeCoinSource,
		ChainlinkConfig:       chainlinkConfig,
		Env:                   environment.New(envConfig),
	}
	return c
}

func (c *Common) SetLocalEnvironment(t *testing.T, genesisAddress string) {
	// Run scripts to set up local test environment
	log.Info().Msg("Starting postgres container...")
	err := exec.Command("../scripts/postgres.sh").Run()
	require.NoError(t, err, "Could not start postgres container")
	log.Info().Msg("Starting mock adapter...")
	err = exec.Command("../scripts/mock-adapter.sh").Run()
	require.NoError(t, err, "Could not start mock adapter")
	log.Info().Msg("Starting core nodes...")
	cmd := exec.Command("../scripts/core.sh")
	cmd.Env = append(os.Environ(), fmt.Sprintf("CL_CONFIG=%s", c.ChainlinkConfig))
	output, err := cmd.CombinedOutput()
	require.NoError(t, err, fmt.Sprintf("Could not start core nodes: %s", string(output)))
	log.Info().Msg("Set up local stack complete.")

	// Set ChainlinkNodeDetails
	var nodeDetails []*environment.ChainlinkNodeDetail
	var basePort = 50100
	for i := 0; i < c.NodeCount; i++ {
		dbLocalIP := fmt.Sprintf("postgresql://postgres:postgres@host.docker.internal:5432/tron_test_%d?sslmode=disable", i+1)
		nodeDetails = append(nodeDetails, &environment.ChainlinkNodeDetail{
			ChartName:  "unused",
			PodName:    "unused",
			LocalIP:    "http://127.0.0.1:" + strconv.Itoa(basePort+i),
			InternalIP: "http://host.docker.internal:" + strconv.Itoa(basePort+i),
			DBLocalIP:  dbLocalIP,
		})
	}
	c.Env.ChainlinkNodeDetails = nodeDetails
}

func (c *Common) TearDownLocalEnvironment(t *testing.T) {
	log.Info().Msg("Tearing down core nodes...")
	err := exec.Command("../scripts/core.down.sh").Run()
	require.NoError(t, err, "Could not tear down core nodes")
	log.Info().Msg("Tearing down mock adapter...")
	err = exec.Command("../scripts/mock-adapter.down.sh").Run()
	require.NoError(t, err, "Could not tear down mock adapter")
	log.Info().Msg("Tearing down postgres container...")
	err = exec.Command("../scripts/postgres.down.sh").Run()
	require.NoError(t, err, "Could not tear down postgres container")
	log.Info().Msg("Tear down local stack complete.")
}
