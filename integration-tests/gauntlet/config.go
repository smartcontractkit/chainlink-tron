package gauntlet

import (
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"os"

	"github.com/ethereum/go-ethereum/crypto"
	"golang.org/x/exp/slog"
	"gopkg.in/yaml.v3"
)

const (
	privateKeyEnv = `PRIVATE_KEY`
)

// Config express all configuration possible via yaml file or via flags
// to be used in any compatibility tests.
//
// an example of a goerli.yaml:
// ```yaml
// description: "goerli ethereum testnet"
// chain_id: 5
// rpc_http: "http://localhost:8545"
// rpc_web_socket: "ws://localhost:8546"
// gauntlet_http: "http://localhost:8080"
// rpc_contract_address: "0x1234567890abcdef1234567890abcdef12345678"
// contract_address: "0xabcdef1234567890abcdef1234567890abcdef12"
// globalvars_contract_address: "0xabcdef1234567890abcdef1234567890abcdef12"
// fail_fast: true
// force_legacy_tx: false
// force_dynamic_tx: true
// gas_limit: 21000
// gas_price: 20000000000
// max_priority_fee_per_gas: 1500000000
// cancel_pending_txs: false
// json_output: false
// ```.
type Config struct {
	PrivateKey                string `json:"-"                         yaml:"private_key"` // it should come from the env and never be marshalled/unmarshalled
	Description               string `json:"description"               yaml:"description"`
	ChainID                   uint64 `json:"chainId"                   yaml:"chain_id"`
	GauntletHTTP              string `json:"gauntletHttp"              yaml:"gauntlet_http"`
	FullNode                  string `json:"fullNodeHttp"              yaml:"full_node_http"`
	SolidityNode              string `json:"solidityNode"              yaml:"solidity_node"`
	RPCContractAddress        string `json:"rpcContractAddress"        yaml:"rpc_contract_address"`
	ContractAddress           string `json:"contractAddress"           yaml:"contract_address"`
	GlobalVarsContractAddress string `json:"globalVarsContractAddress" yaml:"globalvars_contract_address"`
	ForceLegacyTx             bool   `json:"forceLegacyTx"             yaml:"force_legacy_tx"`
	ForceDynamicTx            bool   `json:"forceDynamicTx"            yaml:"force_dynamic_tx"`
	GasLimit                  uint64 `json:"gasLimit"                  yaml:"gas_limit"`
	GasPrice                  uint64 `json:"gasPrice"                  yaml:"gas_price"`
	MaxPriorityFeePerGas      uint64 `json:"maxPriorityFeePerGas"      yaml:"max_priority_fee_per_gas"`
	CancelPendingTxs          bool   `json:"cancelPendingTxs"          yaml:"cancel_pending_txs"`
	JSONOutput                bool   `json:"jsonOutput"                yaml:"json_output"`
	VerifiableReport          bool   `json:"verifiableReport"          yaml:"verifiable_report"`
}

type Option func(*Config)

func WithDescription(description string) Option {
	return func(c *Config) {
		if description != "" {
			c.Description = description
		}
	}
}

func WithChainID(chainID uint64) Option {
	return func(c *Config) {
		if chainID != 0 {
			c.ChainID = chainID
		}
	}
}

func WithGauntletHTTP(url string) Option {
	return func(c *Config) {
		if url != "" {
			c.GauntletHTTP = url
		}
	}
}

func WithRPCContractAddress(rpcContractAddress string) Option {
	return func(c *Config) {
		if rpcContractAddress != "" {
			c.RPCContractAddress = rpcContractAddress
		}
	}
}

func WithContractAddress(contractAddress string) Option {
	return func(c *Config) {
		if contractAddress != "" {
			c.ContractAddress = contractAddress
		}
	}
}

func WithGlobalVarsContractAddress(globalVarsContractAddress string) Option {
	return func(c *Config) {
		if globalVarsContractAddress != "" {
			c.GlobalVarsContractAddress = globalVarsContractAddress
		}
	}
}

func WithForceLegacyTx(forceLegacyTx bool) Option {
	return func(c *Config) {
		if forceLegacyTx {
			c.ForceLegacyTx = forceLegacyTx
		}
	}
}

func WithForceDynamicTx(forceDynamicTx bool) Option {
	return func(c *Config) {
		if forceDynamicTx {
			c.ForceDynamicTx = forceDynamicTx
		}
	}
}

func WithGasLimit(gasLimit uint64) Option {
	return func(c *Config) {
		if gasLimit != 0 {
			c.GasLimit = gasLimit
		}
	}
}

func WithGasPrice(gasPrice uint64) Option {
	return func(c *Config) {
		if gasPrice != 0 {
			c.GasPrice = gasPrice
		}
	}
}

func WithMaxPriorityFeePerGas(maxPriorityFeePerGas uint64) Option {
	return func(c *Config) {
		if maxPriorityFeePerGas != 0 {
			c.MaxPriorityFeePerGas = maxPriorityFeePerGas
		}
	}
}

func WithCancelPendingTxs(cancelPendingTxs bool) Option {
	return func(c *Config) {
		if cancelPendingTxs {
			c.CancelPendingTxs = cancelPendingTxs
		}
	}
}

func WithJSONOutput(jsonOutput bool) Option {
	return func(c *Config) {
		if jsonOutput {
			c.JSONOutput = jsonOutput
		}
	}
}

func WithVerifiableReport(verifiable bool) Option {
	return func(c *Config) {
		if verifiable {
			c.VerifiableReport = verifiable
		}
	}
}

func ParsePrivateKey(privateKey string) (*ecdsa.PrivateKey, error) {
	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, fmt.Errorf("error on parse privateKey: %w", err)
	}

	return pk, nil
}

func parseConfigFile(filepath string, config *Config) error {
	fileContent, err := os.ReadFile(filepath)
	if err != nil {
		return fmt.Errorf("error reading YAML config file: %w", err)
	}
	err = yaml.Unmarshal(fileContent, config)
	if err != nil {
		return fmt.Errorf("error parsing YAML config file: %w", err)
	}
	return nil
}

// New initialize a Config using YAML config file and flags,
// using flags as precedence of the config file, which means
// that you can override any config from file using flags.
func New(filePath string, opts ...Option) (*Config, error) {
	config := Config{}

	if filePath != "" {
		slog.Debug("parsing YAML config file (" + filePath + ")")
		if err := parseConfigFile(filePath, &config); err != nil {
			return nil, err
		}
	}

	for _, opt := range opts {
		opt(&config)
	}

	slog.Debug("the config structure parsed without private key: " + fmt.Sprintf("%+v", config))

	if p := os.Getenv(privateKeyEnv); p != "" {
		config.PrivateKey = p
	}

	return &config, nil
}

func validateURL(addr string) error {
	_, err := url.ParseRequestURI(addr)
	if err != nil {
		return err
	}

	u, err := url.Parse(addr)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("error happening on URL parsing: %w", err)
	}

	return nil
}

// Validate checks if the minimum required config is present and valid.
func (c *Config) Validate() error {
	if c.TryParsePrivateKey() == nil {
		return errors.New("private key should be provided by env (" + privateKeyEnv + ") or yaml config file")
	}

	if c.SolidityNode == "" {
		return errors.New("solidityNode is empty")
	}

	if c.FullNode == "" {
		return errors.New("fullHost is empty")
	}

	if c.ForceDynamicTx && c.ForceLegacyTx {
		return errors.New("both ForceDynamicTx and ForceLegacyTx are true, choose one")
	}

	return nil
}

func PrivateKeyStr(p *ecdsa.PrivateKey) string {
	// convert the private key to a byte slice
	privBytes := p.D.Bytes()

	// convert the byte slice to a hexadecimal string
	return hex.EncodeToString(privBytes)
}

func (c *Config) TryParsePrivateKey() *ecdsa.PrivateKey {
	pk, err := ParsePrivateKey(c.PrivateKey)
	if err != nil {
		return nil
	}
	return pk
}
