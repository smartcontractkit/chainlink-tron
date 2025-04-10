package relayer_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/stretchr/testify/require"

	"github.com/smartcontractkit/chainlink-tron/relayer"
)

func TestGetFunctionSignature_Simple(t *testing.T) {
	abi := &common.JSONABI{
		Entrys: []common.Entry{
			{
				Name: "foo",
				Inputs: []common.EntryInput{
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
	sigHash, err := abi.GetFunctionSignature("foo")
	require.NoError(t, err)
	require.Equal(t, "foo(uint64,uint64)", sigHash)
}

func TestGetFunctionSignature_NotFound(t *testing.T) {
	abi := &common.JSONABI{
		Entrys: []common.Entry{
			{
				Name: "foo",
				Inputs: []common.EntryInput{
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

	_, err := abi.GetFunctionSignature("foo()") // parentheses not required
	require.ErrorContains(t, err, "entry with name foo() not found in abi")

	_, err = abi.GetFunctionSignature("bar") // method doesnt exist
	require.ErrorContains(t, err, "entry with name bar not found in abi")
}

func TestGetFunctionSignature_TuplesAndArrays(t *testing.T) {
	abi := &common.JSONABI{
		Entrys: []common.Entry{
			{
				Name: "foo",
				Inputs: []common.EntryInput{
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
	sigHash, err := abi.GetFunctionSignature("foo")
	require.NoError(t, err)
	require.Equal(t, "foo(uint64,uint64,uint256[],(uint256,uint256))", sigHash)
}

func TestGetEventTopicHash(t *testing.T) {
	topicHash := relayer.GetEventTopicHash("Transfer(address,address,uint256)")
	expectedHash := "ddf252ad1be2c89b69c2b068fc378daa952ba7f163c4a11628f55a4df523b3ef"
	require.Equal(t, expectedHash, topicHash)
}

func TestByteArrToStr(t *testing.T) {
	b := [][]byte{
		{0x01, 0x02, 0x03},
		{0x04, 0x05, 0x06},
	}
	str := relayer.ByteArrayToStr(b)
	require.Equal(t, "[0x010203,0x040506]", str)
}
