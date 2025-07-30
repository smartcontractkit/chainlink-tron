package testutils

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"
)

const (
	// Local RPCs
	DefaultInternalFullNodeUrl     = "http://172.255.0.101:16667/wallet"
	DefaultInternalSolidityNodeUrl = "http://172.255.0.101:16668/walletsolidity"
	FullNodePort                   = "16667"
	SolidityNodePort               = "16668"

	// Testnet RPCs
	// Urls can be found at https://developers.tron.network/reference/background
	ShastaFullNodeUrl     = "https://api.shasta.trongrid.io/wallet"
	ShastaSolidityNodeUrl = "https://api.shasta.trongrid.io/walletsolidity"

	NileFullNodeUrl     = "https://nile.trongrid.io/wallet"
	NileSolidityNodeUrl = "https://nile.trongrid.io/walletsolidity"

	// Configs for TXM
	DevnetFeeLimit                  = 1_000_000_000
	DevnetMaxWaitTime               = 30 //seconds
	DevnetPollFrequency             = 1  //seconds
	DevnetOcrTransmissionFrequency  = 5 * time.Second
	TestnetFeeLimit                 = 10_000_000_000
	TestnetMaxWaitTime              = 90 //seconds
	TestnetPollFrequency            = 5  //seconds
	TestnetOcrTransmissionFrequency = 10 * time.Second

	// Testing network names
	Shasta = "shasta"
	Nile   = "nile"
	Devnet = "devnet"
)

func StartTronNode(genesisAddress string) error {
	gitRoot, err := FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find Git root: %+w", err)
	}

	scriptPath := filepath.Join(gitRoot, "scripts/java-tron.sh")
	cmd := exec.Command(scriptPath, genesisAddress)

	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Printf("Failed to start java-tron, dumping output:\n%s\n", string(output))
			return fmt.Errorf("Failed to start java-tron, bad exit code: %v", exitError.ExitCode())
		}
		return fmt.Errorf("Failed to start java-tron: %+w", err)
	}

	return nil
}

func StopTronNode() error {
	gitRoot, err := FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find Git root: %+w", err)
	}

	scriptPath := filepath.Join(gitRoot, "scripts/java-tron.down.sh")
	cmd := exec.Command(scriptPath)

	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Printf("Failed to stop java-tron, dumping output:\n%s\n", string(output))
			return fmt.Errorf("Failed to start java-tron, bad exit code: %v", exitError.ExitCode())
		}
		return fmt.Errorf("Failed to stop java-tron: %+w", err)
	}

	return nil
}

func GetTronNodeIpAddress() string {
	if runtime.GOOS == "darwin" {
		return "127.0.0.1"
	} else {
		return "172.255.0.101"
	}
}
