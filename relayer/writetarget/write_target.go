package writetarget

import (
	"context"

	"github.com/smartcontractkit/chainlink-tron/relayer/plugin"

	"github.com/smartcontractkit/chainlink-common/pkg/capabilities"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-framework/capabilities/writetarget"
)

func NewWriteTarget(ctx context.Context, relayer *plugin.TronRelayer, lggr logger.Logger) (capabilities.ExecutableCapability, error) {
	writetarget.NewWriteTarget(writetarget.WriteTargetOpts{})
	return nil, nil
}
