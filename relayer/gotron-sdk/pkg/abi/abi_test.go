package abi

import (
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetPaddedParam(t *testing.T) {
	params := []any{
		"string", "KLV Test Token",
		"string", "KLV",
		"uint8", "6",
		"uint256", "100000000000000000000",
	}
	b, err := GetPaddedParam(params)
	require.Nil(t, err)
	assert.Len(t, b, 256, fmt.Sprintf("Wrong length %d/%d", len(b), 256))
}

func TestGetPaddedParam_OddLen(t *testing.T) {
	params := []any{
		"string", "KLV Test Token",
		"string", "KLV",
		"uint8", "6",
		"uint256",
	}
	_, err := GetPaddedParam(params)
	require.NotNil(t, err)
	assert.ErrorContains(t, err, "expected even number of params, got 7")
}

func TestGetPaddedParam_AddressArray(t *testing.T) {
	b, err := GetPaddedParam([]any{
		"address[2]", []string{"TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R", "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"},
	})
	require.Nil(t, err)
	assert.Len(t, b, 64, fmt.Sprintf("Wrong length %d/%d", len(b), 64))
	assert.Equal(t, "000000000000000000000000364b03e0815687edaf90b81ff58e496dea7383d7000000000000000000000000364b03e0815687edaf90b81ff58e496dea7383d7", hex.EncodeToString(b))
}

func TestGetPaddedParam_Uint256Array(t *testing.T) {
	b, err := GetPaddedParam([]any{
		"uint256[2]", []string{"100000000000000000000", "200000000000000000000"},
	})
	require.Nil(t, err)
	assert.Len(t, b, 64, fmt.Sprintf("Wrong length %d/%d", len(b), 64))
	assert.Equal(t, "0000000000000000000000000000000000000000000000056bc75e2d6310000000000000000000000000000000000000000000000000000ad78ebc5ac6200000", hex.EncodeToString(b))
}

func TestGetPaddedParam_Bytes32(t *testing.T) {
	b, err := GetPaddedParam([]any{
		"bytes32", [32]byte{00, 01, 02, 00, 01, 02, 00, 01, 02, 00, 01, 02, 00, 01, 02, 00, 01, 02, 00, 01, 02, 00, 01, 02, 00, 01, 02, 00, 01, 02, 00, 01},
	})
	require.Nil(t, err)
	assert.Len(t, b, 32, fmt.Sprintf("Wrong length %d/%d", len(b), 64))
	assert.Equal(t, "0001020001020001020001020001020001020001020001020001020001020001", hex.EncodeToString(b))
}

func TestGetPaddedParam_HexUint256(t *testing.T) {
	params := []any{
		"uint256", "43981",
		"uint256", "0xABCD",
	}
	b, err := GetPaddedParam(params)
	require.Nil(t, err)
	assert.Len(t, b, 64, fmt.Sprintf("Wrong length %d/%d", len(b), 256))
	assert.Equal(t, "000000000000000000000000000000000000000000000000000000000000abcd000000000000000000000000000000000000000000000000000000000000abcd", hex.EncodeToString(b))
}

func TestGetPaddedParam_BytesArray(t *testing.T) {
	tests := []struct {
		name        string
		byteArray   [][]byte
		expected    string
		expectedLen int
	}{
		{
			name:      "Empty byte array",
			byteArray: [][]byte{},
			// The expected value is the 32-byte padded representation of an empty bytes array,
			// which consists of "0x20" (offset value) followed by "0x80" (length of dynamic data),
			// and then padded with zeros to make it 32 bytes long.
			expected:    "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000000",
			expectedLen: 64,
		},
		{
			name:        "Single element byte array",
			byteArray:   [][]byte{{0x01, 0x02, 0x03, 0x04}},
			expected:    "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000001000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000000040102030400000000000000000000000000000000000000000000000000000000",
			expectedLen: 160,
		},
		{
			name:        "Multiple element byte array",
			byteArray:   [][]byte{{01}, {02}, {03}},
			expected:    "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000e0000000000000000000000000000000000000000000000000000000000000000101000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000001020000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000010300000000000000000000000000000000000000000000000000000000000000",
			expectedLen: 352,
		},
		{
			name:        "Mixed content in byte array",
			byteArray:   [][]byte{{0x01, 0x02}, {0x03, 0x04, 0x05}},
			expected:    "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000002000000000000000000000000000000000000000000000000000000000000004000000000000000000000000000000000000000000000000000000000000000800000000000000000000000000000000000000000000000000000000000000002010200000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000030304050000000000000000000000000000000000000000000000000000000000",
			expectedLen: 256,
		},

		{
			name:        "Mixed content with empty hex string",
			byteArray:   [][]byte{{0x01, 0x02}, {}, {0x03, 0x04, 0x05}},
			expected:    "00000000000000000000000000000000000000000000000000000000000000200000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000006000000000000000000000000000000000000000000000000000000000000000a000000000000000000000000000000000000000000000000000000000000000c000000000000000000000000000000000000000000000000000000000000000020102000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000030304050000000000000000000000000000000000000000000000000000000000",
			expectedLen: 320,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b, err := GetPaddedParam([]any{
				"bytes[]", tt.byteArray,
			})
			require.Nil(t, err)
			assert.Len(t, b, tt.expectedLen, fmt.Sprintf("Wrong length %d/%d", len(b), tt.expectedLen))
			if tt.expectedLen > 0 {
				assert.Equal(t, tt.expected, hex.EncodeToString(b))
			}
		})
	}
}

func TestLoadFromJSON(t *testing.T) {
	jsonInput := `[{"bytes[]":"[\"01020304\"]"}]`
	params, err := LoadFromJSON(jsonInput)
	require.Nil(t, err)
	assert.Len(t, params, 1)
	assert.Equal(t, Param{"bytes[]": "[\"01020304\"]"}, params[0])

	jsonInput = `[{"address[2]":["TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R", "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"]},{"uint256[2]":["100000000000000000000", "200000000000000000000"]}]`
	params, err = LoadFromJSON(jsonInput)
	require.Nil(t, err)
	assert.Len(t, params, 2)
	assert.Equal(t, Param{"address[2]": []interface{}{"TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R", "TEvHMZWyfjCAdDJEKYxYVL8rRpigddLC1R"}}, params[0])
	assert.Equal(t, Param{"uint256[2]": []interface{}{"100000000000000000000", "200000000000000000000"}}, params[1])
}

func TestSelector(t *testing.T) {
	assert.Equal(t, "a9059cbb", hex.EncodeToString(Selector("transfer(address,uint256)")))
	assert.Equal(t, "581f3c50", hex.EncodeToString(Selector("createAndOpen(address,address)")))
	assert.Equal(t, "23b872dd", hex.EncodeToString(Selector("transferFrom(address,address,uint256)")))
}

func TestPack(t *testing.T) {
	packed, err := Pack("transferFrom(address,address,uint256)", []any{
		"address", "0x364b03e0815687edaf90b81ff58e496dea7383d7",
		"address", "0x364b03e0815687edaf90b81ff58e496dea7383d7",
		"uint256", "10000000000000000",
	})
	assert.NoError(t, err)
	assert.Equal(t, "23b872dd000000000000000000000000364b03e0815687edaf90b81ff58e496dea7383d7000000000000000000000000364b03e0815687edaf90b81ff58e496dea7383d7000000000000000000000000000000000000000000000000002386f26fc10000", hex.EncodeToString(packed))
}
