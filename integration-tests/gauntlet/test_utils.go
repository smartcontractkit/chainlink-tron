package gauntlet

import (
	"context"
	"encoding/json"
	"fmt"
	"runtime"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/contract"
	"github.com/valyala/fastjson"
	"golang.org/x/exp/slog"
)

type ContractTest struct {
	name            string
	config          *Config
	provider        Provider
	contractAddress address.Address
}

func NewContractTest(name string, config *Config,
	provider Provider, contractAddress address.Address) *ContractTest {
	return &ContractTest{
		name:            name,
		config:          config,
		provider:        provider,
		contractAddress: contractAddress,
	}
}

func (t *ContractTest) Name() string {
	return t.name
}

func (t *ContractTest) ContractAddress() address.Address {
	return t.contractAddress
}

// Initialises tests and deploys contract
func (t *ContractTest) Setup(ctx context.Context, contractName contract.Name, constructorArgs ...interface{}) error {
	if err := t.config.Validate(); err != nil {
		return fmt.Errorf("validating config, err: %w", err)
	}

	err := t.provider.Init(t.config)
	if err != nil {
		return fmt.Errorf("provider initialization err: %w", err)
	}
	// Using the config address as the contract address
	if t.config.ContractAddress == "" {
		t.contractAddress, err = t.DeployContract(ctx, contractName, constructorArgs...)
		if err != nil {
			return fmt.Errorf("contract deployment err: %w", err)
		}
	} else {
		t.contractAddress = address.HexToAddress(t.config.ContractAddress)
	}

	return nil
}

func (t *ContractTest) Teardown(test *testing.T, logger logger.Logger) error {
	t.provider.Close()
	return nil
}

func (t *ContractTest) DeployContract(ctx context.Context, contractName contract.Name, constructorArgs ...interface{}) (address.Address, error) {
	slog.Info("Deploying contract: " + string(contractName))

	body, err := DeployContract(t.config, contractName, constructorArgs...)
	if err != nil {
		return address.Address{}, fmt.Errorf("error deploying contract: %w", err)
	}

	output, outputErr, err := t.provider.PostExecute(ctx, body)
	if err != nil {
		return address.Address{}, fmt.Errorf("error on calling provider.POSTExecute(), err: %w", err)
	}

	if outputErr != nil {
		return address.Address{}, fmt.Errorf("error deploying the contract %s, err: %v", contractName, outputErr)
	}

	if output == nil {
		return address.Address{}, fmt.Errorf("expecting output inside the response body, but got nil")
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		return address.Address{}, fmt.Errorf("output marshal to []byte error: %w", err)
	}

	v, err := fastjson.ParseBytes(jsonData)
	if err != nil {
		return address.Address{}, fmt.Errorf("fastjson parsing error: %w", err)
	}

	contractAddress := v.GetStringBytes("address")
	if contractAddress == nil {
		return address.Address{}, fmt.Errorf("expecting contract address inside the response body, but got nil")
	}

	return address.HexToAddress(string(contractAddress[:])), nil
}

func (t *ContractTest) InvokeOperation(ctx context.Context, contractAddress string, contractName contract.Name, method string, args ...interface{}) error {
	slog.Info("Invoking contract: " + string(contractName))

	body, err := InvokeContractBody(t.config, contractAddress, contractName, method, args...)
	if err != nil {
		return fmt.Errorf("error invoking contract: %w", err)
	}

	output, outputErr, err := t.provider.PostExecute(ctx, body)
	if err != nil {
		return fmt.Errorf("error on calling provider.POSTExecute(), err: %w", err)
	}

	if outputErr != nil {
		return fmt.Errorf("error invoking the contract %s, err: %v", contractName, outputErr)
	}

	if output == nil {
		return fmt.Errorf("expecting output inside the response body, but got nil")
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		return fmt.Errorf("output marshal to []byte error: %w", err)
	}

	_, err = fastjson.ParseBytes(jsonData)
	if err != nil {
		return fmt.Errorf("fastjson parsing error: %w", err)
	}

	slog.Info("Contract " + string(contractName) + " invoked successfully")

	return nil
}

func (t *ContractTest) QueryContract(ctx context.Context, contractAddress string, contractName contract.Name, method, callerAddress string, args ...interface{}) (*fastjson.Value, error) {
	slog.Info("Querying contract: " + string(contractName))

	body, err := QueryContractBody(t.config, contractAddress, contractName, method, callerAddress, args...)
	if err != nil {
		return nil, fmt.Errorf("error on gauntlet.QueryContractBody() with contract (%s) and method (%s), err: %+w", contractName, method, err)
	}

	output, err := t.provider.PostQuery(ctx, body)
	if err != nil {
		return nil, fmt.Errorf("error on calling provider.POSTExecute(), err: %w", err)
	}

	if output == nil {
		return nil, fmt.Errorf("expecting output inside the response body, but got nil")
	}

	jsonData, err := json.Marshal(output)
	if err != nil {
		return nil, fmt.Errorf("output marshal to []byte error: %w", err)
	}

	jsonOutput, err := fastjson.ParseBytes(jsonData)
	if err != nil {
		return nil, fmt.Errorf("fastjson parsing error: %w", err)
	}

	return jsonOutput, nil
}

// Creates a test config for Gauntlet ++ deployment tests
func NewDeploymentLocalTestConfig(privateKeyHex string) Config {
	var config Config

	const fullNodeHTTPPort = 16667
	const solidityNodeHTTPPort = 16668
	const GauntletPort = 8080

	var nodeIp string

	if runtime.GOOS == "darwin" {
		nodeIp = "localhost" // Mac OS needs port forwarding for docker
	} else {
		nodeIp = "172.255.0.101" // Linux does not need port forwarding
	}

	config.PrivateKey = privateKeyHex
	config.FullNode = fmt.Sprintf("http://%s:%d", nodeIp, fullNodeHTTPPort)
	config.SolidityNode = fmt.Sprintf("http://%s:%d", nodeIp, solidityNodeHTTPPort)
	config.GauntletHTTP = fmt.Sprintf("http://%s:%d", "localhost", GauntletPort)

	return config
}
