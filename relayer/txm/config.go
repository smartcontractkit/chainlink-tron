package txm

type TronTxmConfig struct {
	BroadcastChanSize uint
	ConfirmPollSecs   uint
	EnergyMultiplier  float64
	FixedEnergyValue  int64
}
