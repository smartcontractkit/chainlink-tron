package relayer_test

import (
	"encoding/hex"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/stretchr/testify/require"
)

func TestGetFunctionSignature_Simple(t *testing.T) {
	abi := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "foo",
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{
						Type: "uint64",
					},
					{
						Type: "uint64",
					},
				},
			},
		},
	}
	sigHash, err := relayer.GetFunctionSignature(abi, "foo")
	require.NoError(t, err)
	require.Equal(t, "foo(uint64,uint64)", sigHash)
}

func TestGetFunctionSignature_NotFound(t *testing.T) {
	abi := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "foo",
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{
						Type: "uint64",
					},
					{
						Type: "uint64",
					},
				},
			},
		},
	}

	_, err := relayer.GetFunctionSignature(abi, "foo()") // parentheses not required
	require.ErrorContains(t, err, "entry with name foo() not found in abi")

	_, err = relayer.GetFunctionSignature(abi, "bar") // method doesnt exist
	require.ErrorContains(t, err, "entry with name bar not found in abi")
}

func TestGetFunctionSignature_TuplesAndArrays(t *testing.T) {
	abi := &core.SmartContract_ABI{
		Entrys: []*core.SmartContract_ABI_Entry{
			{
				Name: "foo",
				Inputs: []*core.SmartContract_ABI_Entry_Param{
					{
						Type: "uint64",
					},
					{
						Type: "uint64",
					},
					{
						Type: "uint256[]",
					},
					{
						Type: "(uint256,uint256)",
					},
				},
			},
		},
	}
	sigHash, err := relayer.GetFunctionSignature(abi, "foo")
	require.NoError(t, err)
	require.Equal(t, "foo(uint64,uint64,uint256[],(uint256,uint256))", sigHash)
}

func TestGetEventTopicHash(t *testing.T) {
	topicHash := relayer.GetEventTopicHash("Transfer(address,address,uint256)")
	expectedHash, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	require.Equal(t, expectedHash, topicHash)
}

func TestTronToEVMAddress(t *testing.T) {
	tronAddr, err := address.Base58ToAddress(relayer.TRON_ZERO_ADDR_B58)
	require.NoError(t, err)
	evmAddr := relayer.TronToEVMAddress(tronAddr)
	expectedEvmAddr := "0x0000000000000000000000000000000000000000"
	require.Equal(t, expectedEvmAddr, evmAddr.Hex())

	tronAddr, err = address.Base58ToAddress("TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8")
	require.NoError(t, err)
	evmAddr = relayer.TronToEVMAddress(tronAddr)
	expectedEvmAddr = "0x3487b63D30B5B2C87fb7fFa8bcfADE38EAaC1abe"
	require.Equal(t, expectedEvmAddr, evmAddr.Hex())

	tronAddr = address.HexToAddress("411234")
	evmAddr = relayer.TronToEVMAddress(tronAddr)
	expectedEvmAddr = "0x0000000000000000000000000000000000001234"
	require.Equal(t, expectedEvmAddr, evmAddr.Hex())
}

func TestEVMToTronAddress(t *testing.T) {
	evmAddr := common.HexToAddress("0x0000000000000000000000000000000000000000")
	tronAddr := relayer.EVMToTronAddress(evmAddr)
	expectedTronAddr, _ := address.Base58ToAddress(relayer.TRON_ZERO_ADDR_B58)
	require.Equal(t, expectedTronAddr, tronAddr)

	evmAddr = common.HexToAddress("0x3487b63D30B5B2C87fb7fFa8bcfADE38EAaC1abe")
	tronAddr = relayer.EVMToTronAddress(evmAddr)
	expectedTronAddr, _ = address.Base58ToAddress("TEkxiTehnzSmSe2XqrBj4w32RUN966rdz8")
	require.Equal(t, expectedTronAddr, tronAddr)

	evmAddr = common.HexToAddress("0x1234")
	tronAddr = relayer.EVMToTronAddress(evmAddr)
	expectedTronAddr = address.HexToAddress("410000000000000000000000000000000000001234")
	require.Equal(t, expectedTronAddr, tronAddr)
}
