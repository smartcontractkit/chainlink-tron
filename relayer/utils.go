package relayer

import (
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"golang.org/x/crypto/sha3"
)

func GetEventTopicHash(eventSignature string) string {
	return hex.EncodeToString(crypto.Keccak256([]byte(eventSignature)))
}

func ByteArrayToStr(b [][]byte) string {
	if len(b) == 0 {
		return "[]"
	}

	var str string = "["
	for _, v := range b {
		str += "0x" + hex.EncodeToString(v[:]) + ","
	}
	return str[:len(str)-1] + "]"
}

func ParseTronAddress(input string) (address.Address, error) {
	if input == "" {
		return nil, fmt.Errorf("input cannot be empty")
	}

	// Try Base58 first
	addr, err := address.StringToAddress(input)
	if err == nil {
		return addr, nil
	}

	// Otherwise, treat as pubkey
	pubKeyBytes, err := hex.DecodeString(input)
	if err != nil {
		return nil, fmt.Errorf("invalid pubkey: %w", err)
	}
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubKeyBytes[1:])
	hashed := hash.Sum(nil)
	addressBytes := hashed[len(hashed)-20:]
	tronHexAddress := "41" + hex.EncodeToString(addressBytes)
	return address.HexToAddress(tronHexAddress)
}
