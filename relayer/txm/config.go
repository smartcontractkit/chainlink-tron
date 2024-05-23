package txm

type TronTxmConfig struct {
	RPCAddress        string
	RPCInsecure       bool
	BroadcastChanSize uint
	ConfirmPollSecs   uint
}
