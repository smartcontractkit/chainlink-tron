package config

import (
	_ "embed"
	"testing"
	"time"

	"github.com/smartcontractkit/chainlink-common/pkg/config"
	"github.com/smartcontractkit/chainlink-common/pkg/config/configtest"
)

func TestDefaults_fieldsNotNil(t *testing.T) {
	configtest.AssertFieldsNotNil(t, Defaults())
}

func TestDocsTOMLComplete(t *testing.T) {
	configtest.AssertDocsTOMLComplete[TOMLConfig](t, docsTOML)
}

//go:embed testdata/config-full.toml
var fullTOML string

func TestTOMLConfig_FullMarshal(t *testing.T) {
	full := TOMLConfig{
		ChainID: ptr("fake"),
		Enabled: ptr(false),
		ChainConfig: ChainConfig{
			BroadcastChanSize:   ptr[uint64](99),
			ConfirmPollPeriod:   config.MustNewDuration(42 * time.Millisecond),
			OCR2CachePollPeriod: config.MustNewDuration(100 * time.Second),
			OCR2CacheTTL:        config.MustNewDuration(15 * time.Minute),
			BalancePollPeriod:   config.MustNewDuration(time.Hour),
			RetentionPeriod:     config.MustNewDuration(0),
			ReapInterval:        config.MustNewDuration(time.Minute),
		},
		Nodes: NodeConfigs{
			{
				Name:        ptr("node"),
				URL:         config.MustParseURL("https://example.com/tron"),
				SolidityURL: config.MustParseURL("http://example.com/solidity"),
			},
		},
	}
	configtest.AssertFullMarshal(t, full, fullTOML)
}

func ptr[T any](v T) *T { return &v }
