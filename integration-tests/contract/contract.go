package contract

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
)

// BasePath represents the base directory where JSON files are stored
var BasePath = "../artifacts/"

// ABI is the golang representation of the json file generated by solc.
type ABI struct {
	Format       string `json:"_format"`
	ContractName string `json:"contractName"`
	SourceName   string `json:"sourceName"`
	Abi          []struct {
		Anonymous bool `json:"anonymous,omitempty"`
		Inputs    []struct {
			Indexed      bool   `json:"indexed,omitempty"`
			InternalType string `json:"internalType"`
			Name         string `json:"name"`
			Type         string `json:"type"`
		} `json:"inputs"`
		Name    string `json:"name"`
		Type    string `json:"type"`
		Outputs []struct {
			InternalType string `json:"internalType"`
			Name         string `json:"name"`
			Type         string `json:"type"`
		} `json:"outputs,omitempty"`
		StateMutability string `json:"stateMutability,omitempty"`
	} `json:"abi"`
	Bytecode               string   `json:"bytecode"`
	DeployedBytecode       string   `json:"deployedBytecode"`
	LinkReferences         struct{} `json:"linkReferences"`
	DeployedLinkReferences struct{} `json:"deployedLinkReferences"`
}

type Artifact struct {
	Abi      abi.ABI
	AbiJson  string `json:"abi"`
	Bytecode string `json:"bytecode"`
}

func loadContract(jsonPath string) (*ABI, error) {
	gitRoot, err := testutils.FindGitRoot()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(filepath.Join(gitRoot, "tron", "integration-tests", "artifacts", jsonPath))
	if err != nil {
		return nil, err
	}

	var abi ABI
	if err := json.Unmarshal(data, &abi); err != nil {
		return nil, err
	}
	return &abi, nil
}

func MustLoadArtifact(t *testing.T, jsonPath string) *Artifact {
	contractJSON, err := loadContract(jsonPath)
	if err != nil {
		t.Fatal(err)
	}

	abiJsonBytes, err := json.Marshal(contractJSON.Abi)
	if err != nil {
		t.Fatal(err)
	}
	abiJson := string(abiJsonBytes)

	parsedAbi, err := abi.JSON(bytes.NewReader([]byte(abiJson)))
	if err != nil {
		t.Fatal(err)
	}

	return &Artifact{
		Abi:      parsedAbi,
		AbiJson:  abiJson,
		Bytecode: contractJSON.Bytecode,
	}
}
