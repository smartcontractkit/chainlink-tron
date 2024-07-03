package relayer

import (
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
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

// Convert a Tron address to an EVM-formatted address
func TronToEVMAddress(addr address.Address) common.Address {
	return common.BytesToAddress(addr.Bytes()[1:])
}

// Convert an EVM address to a Tron-formatted address
func EVMToTronAddress(addr common.Address) address.Address {
	return address.HexToAddress("0x41" + addr.String()[2:])
}
