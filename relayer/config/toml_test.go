package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDefaults(t *testing.T) {
	c := Defaults()
	require.Len(t, c.Nodes, 0)
}

func TestTOMLConfig_SetDefaults(t *testing.T) {
	var c TOMLConfig
	require.Len(t, c.Nodes, 0)
	c.SetDefaults()
	require.Len(t, c.Nodes, 0)
}
