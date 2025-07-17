package txm

import "time"

type TronTxmConfig struct {
	BroadcastChanSize uint
	ConfirmPollSecs   uint
	EnergyMultiplier  float64
	FixedEnergyValue  int64
	FinalityDepth     uint64
	RetentionPeriod   time.Duration
	ReapInterval      time.Duration
}
