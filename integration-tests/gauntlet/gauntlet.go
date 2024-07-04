package gauntlet

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/contract"
	gauntletgen "github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/gauntlet/gauntletgen"
	"golang.org/x/exp/slog"
)

type operationArgs struct {
	// Abi Contract ABI JSON string.
	Abi string `json:"abi"`

	// Address The address of the contract to call.
	Address string `json:"address"`

	// Args The args used for this call.
	Args *[]interface{} `json:"args,omitempty"`

	// Method The method to call.
	Method string `json:"method"`

	From string `json:"from,omitempty"`
}

type artifactArgs struct {
	Abi      string `json:"abi"`
	Bytecode string `json:"bytecode"`
}

// TRON specific deploy args
type deployArgs struct {
	Artifact artifactArgs   `json:"artifact"`
	Args     *[]interface{} `json:"args,omitempty"`
}

func providers(config *Config) []gauntletgen.Provider {
	var (
		developmentBool interface{} = true
		fullHostUrl     interface{} = config.FullNode
		solidityNodeUrl interface{} = config.SolidityNode
		pk              interface{} = func() string {
			if !strings.HasPrefix(config.PrivateKey, "0x") {
				return "0x" + config.PrivateKey
			}
			return config.PrivateKey
		}()
	)
	return []gauntletgen.Provider{
		{
			Name: "raw-pk",
			Type: "@gauntlet/tron/signer",
			Input: map[string]*interface{}{
				"privateKey":  &pk,
				"development": &developmentBool,
			},
		},
		{
			Name: "tronweb",
			Type: "@gauntlet/tron/lib/tronweb",
			Input: map[string]*interface{}{
				"fullHost":     &fullHostUrl,
				"solidityNode": &solidityNodeUrl,
				"privateKey":   &pk,
			},
		},
		{
			Name:  "basic-estimator",
			Type:  "@gauntlet/tron/energy-estimator",
			Input: map[string]*interface{}{},
		},
	}
}

// QueryContractBody is the gauntlet++ wrapper to query the contract.
func QueryContractBody(config *Config, contractAddress string, contractName contract.Name, method string, fromAddress string, contractArgs ...interface{}) (io.Reader, error) {
	artifact, err := contract.ArtifactFromContract(contractName)
	if err != nil {
		return nil, err
	}

	// contractArgs must not be null for the G++ body
	if contractArgs == nil {
		contractArgs = []interface{}{}
	}

	// PostQueryJSONBody.Config.Providers
	providers := providers(config)

	operationArgsInput := operationArgs{
		Abi:     string(artifact.Abi),
		Address: contractAddress,
		Method:  method,
		Args:    &contractArgs,
	}

	if fromAddress != "" {
		operationArgsInput.From = fromAddress
	}

	// transform args into interface
	var iOperationArgs interface{} = operationArgsInput

	body := gauntletgen.PostQueryJSONRequestBody{
		Config: &gauntletgen.Config{
			Providers: providers,
			// TODO: this field should be optional, waiting for go-gauntlet fix it
			Datasources: []gauntletgen.Datasource{},
		},
		Operation: gauntletgen.Operation{
			Name: "tron/chain/contract:call",
			Args: &iOperationArgs,
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	slog.Debug("QueryContractBody():", slog.String("body", string(buf)))

	return bytes.NewReader(buf), nil
}

// InvokeContractBody is the gauntlet++ wrapper to call (i.e change the state) the contract.
func InvokeContractBody(config *Config, contractAddress string, contractName contract.Name, method string, contractArgs ...interface{}) (io.Reader, error) {
	artifact, err := contract.ArtifactFromContract(contractName)
	if err != nil {
		return nil, err
	}

	// contractArgs must not be null for the G++ body
	if contractArgs == nil {
		contractArgs = []interface{}{}
	}

	// PostQueryJSONBody.Config.Providers
	providers := providers(config)

	operationArgsInput := operationArgs{
		Abi:     string(artifact.Abi),
		Address: contractAddress,
		Method:  method,
		Args:    &contractArgs,
	}

	// transform args into interface
	var iOperationArgs interface{} = operationArgsInput

	body := gauntletgen.PostExecuteJSONRequestBody{
		Config: &gauntletgen.Config{
			Providers: providers,
			// TODO: this field should be optional, waiting for go-gauntlet fix it
			Datasources: []gauntletgen.Datasource{},
		},
		Operation: gauntletgen.Operation{
			Name: "tron/chain/contract:invoke",
			Args: &iOperationArgs,
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	slog.Debug("InvokeContractBody():", slog.String("body", string(buf)))

	return bytes.NewReader(buf), nil
}

// DeployContract is the gauntlet++ wrapper to deploy the contract.
func DeployContract(config *Config, contractName contract.Name, args ...interface{}) (io.Reader, error) {
	artifact, err := contract.ArtifactFromContract(contractName)
	if err != nil {
		return nil, err
	}

	// args must not be null for the G++ body
	if args == nil {
		args = []interface{}{}
	}

	providers := providers(config)

	deployArgsInput := deployArgs{
		Artifact: artifactArgs{
			Abi:      string(artifact.Abi),
			Bytecode: artifact.Bytecode,
		},
		Args: &args,
	}

	var iDeployArgs interface{} = deployArgsInput

	body := gauntletgen.PostExecuteJSONRequestBody{
		Config: &gauntletgen.Config{
			Providers: providers,
			// TODO: this field should be optional, waiting for go-gauntlet fix it
			Datasources: []gauntletgen.Datasource{},
		},
		Operation: gauntletgen.Operation{
			Name: "tron/chain/contract:deploy",
			Args: &iDeployArgs,
		},
	}

	buf, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	return bytes.NewReader(buf), nil
}
