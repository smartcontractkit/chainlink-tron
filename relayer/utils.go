package relayer

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
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
