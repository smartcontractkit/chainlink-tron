package relayer

import (
	"encoding/hex"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
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
	sigHash, err := GetFunctionSignature(abi, "foo")
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

	_, err := GetFunctionSignature(abi, "foo()") // parentheses not required
	require.ErrorContains(t, err, "method foo() not found in abi")

	_, err = GetFunctionSignature(abi, "bar") // method doesnt exist
	require.ErrorContains(t, err, "method bar not found in abi")
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
	sigHash, err := GetFunctionSignature(abi, "foo")
	require.NoError(t, err)
	require.Equal(t, "foo(uint64,uint64,uint256[],(uint256,uint256))", sigHash)
}

func TestGetEventTopicHash(t *testing.T) {
	topicHash := GetEventTopicHash("Transfer(address,address,uint256)")
	expectedHash, _ := hex.DecodeString("ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef")
	require.Equal(t, expectedHash, topicHash)
}
