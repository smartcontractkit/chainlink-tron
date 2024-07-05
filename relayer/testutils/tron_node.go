package testutils

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
)

func StartTronNode(genesisAddress string) error {
	gitRoot, err := FindGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find Git root: %+w", err)
	}

	scriptPath := filepath.Join(gitRoot, "tron/scripts/java-tron.sh")
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

	scriptPath := filepath.Join(gitRoot, "tron/scripts/java-tron.down.sh")
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
