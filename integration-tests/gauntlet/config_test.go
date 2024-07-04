package gauntlet

import (
	"fmt"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestWithDescription(t *testing.T) {
	desc := "test description"
	c := &Config{}
	WithDescription(desc)(c)

	assert.Equal(t, desc, c.Description)
}

func TestWithChainID(t *testing.T) {
	chainID := uint64(5)
	c := &Config{}
	WithChainID(chainID)(c)

	assert.Equal(t, chainID, c.ChainID)
}

func TestWithGauntletHTTP(t *testing.T) {
	url := "http://localhost:8080"
	c := &Config{}
	WithGauntletHTTP(url)(c)

	assert.Equal(t, url, c.GauntletHTTP)
}

func TestWithRPCContractAddress(t *testing.T) {
	addr := "0x1234567890abcdef1234567890abcdef12345678"
	c := &Config{}
	WithRPCContractAddress(addr)(c)

	assert.Equal(t, addr, c.RPCContractAddress)
}

func TestWithContractAddress(t *testing.T) {
	addr := "0xabcdef1234567890abcdef1234567890abcdef12"
	c := &Config{}
	WithContractAddress(addr)(c)

	assert.Equal(t, addr, c.ContractAddress)
}

func TestWithGlobalVarsContractAddress(t *testing.T) {
	addr := "0xabcdef1234567890abcdef1234567890abcdef12"
	c := &Config{}
	WithGlobalVarsContractAddress(addr)(c)

	assert.Equal(t, addr, c.GlobalVarsContractAddress)
}

func TestWithForceLegacyTx(t *testing.T) {
	c := &Config{}
	WithForceLegacyTx(true)(c)

	assert.True(t, c.ForceLegacyTx)
}

func TestWithForceDynamicTx(t *testing.T) {
	c := &Config{}
	WithForceDynamicTx(true)(c)

	assert.True(t, c.ForceDynamicTx)
}

func TestWithGasLimit(t *testing.T) {
	gasLimit := uint64(21000)
	c := &Config{}
	WithGasLimit(gasLimit)(c)

	assert.Equal(t, gasLimit, c.GasLimit)
}

func TestWithGasPrice(t *testing.T) {
	gasPrice := uint64(20000000000)
	c := &Config{}
	WithGasPrice(gasPrice)(c)

	assert.Equal(t, gasPrice, c.GasPrice)
}

func TestWithMaxPriorityFeePerGas(t *testing.T) {
	fee := uint64(1500000000)
	c := &Config{}
	WithMaxPriorityFeePerGas(fee)(c)

	assert.Equal(t, fee, c.MaxPriorityFeePerGas)
}

func TestWithCancelPendingTxs(t *testing.T) {
	c := &Config{}
	WithCancelPendingTxs(true)(c)

	assert.True(t, c.CancelPendingTxs)
}

func TestWithJSONOutput(t *testing.T) {
	c := &Config{}
	WithJSONOutput(true)(c)

	assert.True(t, c.JSONOutput)
}

func TestWithVerifiableReport(t *testing.T) {
	c := &Config{}
	WithVerifiableReport(true)(c)

	assert.True(t, c.VerifiableReport)
}

func TestParseConfigFile(t *testing.T) {
	expectedConfig := &Config{
		Description:          "test description",
		ChainID:              5,
		SolidityNode:         "http://localhost:8545",
		FullNode:             "ws://localhost:8546",
		RPCContractAddress:   "0x1234567890abcdef1234567890abcdef12345678",
		ContractAddress:      "0xabcdef1234567890abcdef1234567890abcdef12",
		ForceLegacyTx:        true,
		ForceDynamicTx:       false,
		GasLimit:             21000,
		GasPrice:             20000000000,
		MaxPriorityFeePerGas: 1500000000,
		CancelPendingTxs:     false,
	}

	t.Run("parsing known config file", func(t *testing.T) {
		// convert the sample configuration into YAML format.
		yamlData, err := yaml.Marshal(expectedConfig)
		require.NoError(t, err)

		// create a temporary file to write the YAML data.
		tmpFile, err := os.CreateTemp("", "config-*.yaml")
		require.NoError(t, err)
		defer os.Remove(tmpFile.Name())

		_, err = tmpFile.Write(yamlData)
		require.NoError(t, err)

		tmpFile.Close()

		parsedConfig := &Config{}
		err = parseConfigFile(tmpFile.Name(), parsedConfig)
		require.NoError(t, err)

		require.Equal(t, expectedConfig, parsedConfig)
	})
}

const configContent = `
description: "goerli ethereum testnet"
chain_id: 5
rpc_http: "http://localhost:8545"
rpc_web_socket: "ws://localhost:8546"
rpc_contract_address: "0x1234567890abcdef1234567890abcdef12345678"
evm_contract_address: "0xabcdef1234567890abcdef1234567890abcdef12"
globalvars_contract_address: "0xabcdef1234567890abcdef1234567890abcdef12"
fail_fast: true
force_legacy_tx: false
force_dynamic_tx: false
gas_limit: 21000
gas_price: 20000000000
max_priority_fee_per_gas: 1500000000
cancel_pending_txs: false
`

func TestNew(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "config-*.yaml")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	_, err = tmpFile.WriteString(configContent)
	require.NoError(t, err)

	err = tmpFile.Close()
	require.NoError(t, err)

	t.Run("with private key from env", func(t *testing.T) {
		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)
		privKey := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))

		t.Setenv(privateKeyEnv, privKey)

		config, err := New(tmpFile.Name())
		require.NoError(t, err)
		require.NotNil(t, config.PrivateKey)
	})

	t.Run("with all fields and private key from env", func(t *testing.T) {
		privateKey, err := crypto.GenerateKey()
		require.NoError(t, err)

		privKey := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))
		t.Setenv(privateKeyEnv, privKey)

		expectedConfig := &Config{
			PrivateKey:                os.Getenv(privateKeyEnv),
			Description:               "test description",
			ChainID:                   5,
			RPCContractAddress:        "0x1234567890abcdef1234567890abcdef12345678",
			ContractAddress:           "0xabcdef1234567890abcdef1234567890abcdef12",
			GlobalVarsContractAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
			ForceLegacyTx:             true,
			ForceDynamicTx:            false,
			GasLimit:                  21000,
			GasPrice:                  20000000000,
			MaxPriorityFeePerGas:      1500000000,
			CancelPendingTxs:          false,
		}

		opts := []Option{
			WithDescription(expectedConfig.Description),
			WithChainID(expectedConfig.ChainID),
			WithRPCContractAddress(expectedConfig.RPCContractAddress),
			WithContractAddress(expectedConfig.ContractAddress),
			WithGlobalVarsContractAddress(expectedConfig.GlobalVarsContractAddress),
			WithForceLegacyTx(expectedConfig.ForceLegacyTx),
			WithForceDynamicTx(expectedConfig.ForceDynamicTx),
			WithGasLimit(expectedConfig.GasLimit),
			WithGasPrice(expectedConfig.GasPrice),
			WithMaxPriorityFeePerGas(expectedConfig.MaxPriorityFeePerGas),
			WithCancelPendingTxs(expectedConfig.CancelPendingTxs),
		}

		config, err := New(tmpFile.Name(), opts...)

		require.NoError(t, err)
		require.Equal(t, expectedConfig, config)
	})
}

func TestParsePrivateKey(t *testing.T) {
	privateKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	privKey := fmt.Sprintf("%x", crypto.FromECDSA(privateKey))

	parsedKey, err := ParsePrivateKey(privKey)
	require.NoError(t, err)

	assert.Equal(t, privateKey, parsedKey)
}

func TestConfigValidate(t *testing.T) {
	privKey, err := crypto.GenerateKey()
	require.NoError(t, err)

	// valid configuration
	validCfg := &Config{
		PrivateKey:                PrivateKeyStr(privKey),
		Description:               "test description",
		ChainID:                   5,
		SolidityNode:              "http://localhost:8545",
		FullNode:                  "ws://localhost:8546",
		RPCContractAddress:        "0x1234567890abcdef1234567890abcdef12345678",
		ContractAddress:           "0xabcdef1234567890abcdef1234567890abcdef12",
		GlobalVarsContractAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
		ForceLegacyTx:             true,
		ForceDynamicTx:            false,
		GasLimit:                  21000,
		GasPrice:                  20000000000,
		MaxPriorityFeePerGas:      1500000000,
		CancelPendingTxs:          false,
	}
	assert.NoError(t, validCfg.Validate())

	// invalid configuration: both ForceDynamicTx and ForceLegacyTx set to true
	invalidCfg := &Config{
		PrivateKey:     PrivateKeyStr(privKey),
		SolidityNode:   "http://localhost:8545",
		FullNode:       "ws://localhost:8546",
		ForceDynamicTx: true,
		ForceLegacyTx:  true,
	}
	assert.Error(t, invalidCfg.Validate())
}
