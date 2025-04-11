package txm

import (
	"context"

	"github.com/smartcontractkit/chainlink-common/pkg/types"
)

type TronTxmConfig struct {
	BroadcastChanSize uint
	ConfirmPollSecs   uint
	EnergyMultiplier  float64
	StatusChecker     CCIPTransactionStatusChecker
}

// CCIPTransactionStatusChecker is an interface that defines the method for checking the status of a transaction.
// CheckMessageStatus checks the status of a transaction for a given message ID.
// It returns a list of transaction statuses, the retry counter, and an error if any occurred during the process.
type CCIPTransactionStatusChecker interface {
	CheckMessageStatus(ctx context.Context, msgID string) (transactionStatuses []types.TransactionStatus, retryCounter int, err error)
}
