package relayer

import (
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"golang.org/x/crypto/sha3"
)

func GetFunctionSignature(abi *core.SmartContract_ABI, name string) (string, error) {
	for _, entry := range abi.Entrys {
		if entry.Name == name {
			var types = make([]string, len(entry.Inputs))
			for i, input := range entry.Inputs {
				types[i] = input.GetType()
			}
			return fmt.Sprintf("%v(%v)", name, strings.Join(types, ",")), nil
		}
	}
	return "", fmt.Errorf("entry with name %v not found in abi", name)
}

func GetEventTopicHash(eventSignature string) []byte {
	return crypto.Keccak256([]byte(eventSignature))
}

// Convert a Tron address to an EVM-formatted address
func TronToEVMAddress(addr address.Address) common.Address {
	return common.BytesToAddress(addr.Bytes()[1:])
}

// Convert an EVM address to a Tron-formatted address
func EVMToTronAddress(addr common.Address) address.Address {
	return address.HexToAddress("0x41" + addr.String()[2:])
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

func PublicKeyToTronAddress(pubKey string) (address.Address, error) {
	pubKeyBytes, err := hex.DecodeString(pubKey)
	if err != nil {
		return nil, err
	}
	hash := sha3.NewLegacyKeccak256()
	hash.Write(pubKeyBytes[1:]) // remove the 0x04 format identifier prefix
	hashed := hash.Sum(nil)
	addressBytes := hashed[len(hashed)-20:]
	tronHexAddress := "41" + hex.EncodeToString(addressBytes)
	return address.HexToAddress(tronHexAddress), nil
}
