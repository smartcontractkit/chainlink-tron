package relayer

import (
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/smartcontractkit/chainlink-tron/relayer/config"
)

var (
	//go:embed CONFIG.md
	configMD string
)

//go:generate go run ./cmd/config-docs
func TestConfigDocs(t *testing.T) {
	cfg, err := config.GenerateDocs()
	assert.NoError(t, err, "invalid config docs")
	assert.Equal(t, configMD, cfg, "CONFIG.md is out of date. Run 'go generate .' to regenerate.")
}
