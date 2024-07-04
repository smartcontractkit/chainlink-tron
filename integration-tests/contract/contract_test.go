package contract

import (
	"testing"
)

func TestArtifactFromContract(t *testing.T) {
	testCases := []struct {
		name          Name
		expectedError bool
	}{
		{Storage, false},
		{OpCodes, false},
		{GlobalVariables, false},
		{Ecrecover, false},
		{Sha256hash, false},
		{Ripemd160hash, false},
		{Datacopy, false},
		{Bigmodexp, false},
		{LinkToken, false},
		{Name("InvalidName"), true},
	}

	for _, tc := range testCases {
		t.Run(string(tc.name), func(t *testing.T) {
			artifact, err := ArtifactFromContract(tc.name)
			if tc.expectedError {
				if err == nil {
					t.Errorf("Expected an error, but got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if len(artifact.Abi) == 0 {
					t.Error("Expected non-empty ABI")
				}
				if artifact.Bytecode == "" {
					t.Error("Expected non-empty bytecode")
				}
			}
		})
	}
}
