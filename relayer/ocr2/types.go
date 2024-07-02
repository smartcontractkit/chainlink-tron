package ocr2

import (
	"math/big"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"
)

// Tron zero address - https://developers.tron.network/docs/faq#3-what-is-the-destruction-address-of-tron
const (
	TRON_ZERO_ADDR_B58 = "T9yD14Nj9j7xAB4dbGeiX9h8unkKHxuWwb"
	TRON_ZERO_ADDR_HEX = "410000000000000000000000000000000000000000"
)

type ContractConfigDetails struct {
	Block  uint64
	Digest types.ConfigDigest
}

type ContractConfig struct {
	Config      types.ContractConfig
	ConfigBlock uint64
}

type TransmissionDetails struct {
	Digest          types.ConfigDigest
	Epoch           uint32
	Round           uint8
	LatestAnswer    *big.Int
	LatestTimestamp time.Time
}

type BillingDetails struct {
	ObservationPaymentGJuels  uint32
	TransmissionPaymentGJuels uint32
}

type RoundData struct {
	RoundID   uint32
	Answer    *big.Int
	StartedAt time.Time
	UpdatedAt time.Time
}

type NewTransmissionEvent struct {
	RoundId         uint32 //nolint:revive
	LatestAnswer    *big.Int
	Transmitter     *address.Address
	LatestTimestamp time.Time
	Observers       []uint8
	ObservationsLen uint32
	Observations    []*big.Int
	JuelsPerFeeCoin *big.Int
	GasPrice        *big.Int
	ConfigDigest    types.ConfigDigest
	Epoch           uint32
	Round           uint8
	Reimbursement   *big.Int
}
