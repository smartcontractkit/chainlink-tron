package ocr2

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/reader"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

//go:generate mockery --name OCR2Reader --output ../mocks/
type OCR2Reader interface { //nolint:revive
	LatestConfigDetails(context.Context, tronaddress.Address) (ContractConfigDetails, error)
	LatestTransmissionDetails(context.Context, tronaddress.Address) (TransmissionDetails, error)
	LatestRoundData(context.Context, tronaddress.Address) (RoundData, error)
	LinkAvailableForPayment(context.Context, tronaddress.Address) (*big.Int, error)
	ConfigFromEventAt(context.Context, tronaddress.Address, uint64) (ContractConfig, error)
	// NewTransmissionsFromEventsAt(context.Context, tronaddress.Address, uint64) ([]NewTransmissionEvent, error) // is this needed?
	BillingDetails(context.Context, tronaddress.Address) (BillingDetails, error)

	BaseReader() reader.Reader
}

var _ OCR2Reader = (*OCR2ReaderClient)(nil)

type OCR2ReaderClient struct {
	r    reader.Reader
	lggr logger.Logger
}

func NewOCR2Reader(reader reader.Reader, lggr logger.Logger) *OCR2ReaderClient {
	return &OCR2ReaderClient{
		r:    reader,
		lggr: lggr,
	}
}

func (c *OCR2ReaderClient) BaseReader() reader.Reader {
	return c.r
}

func (c *OCR2ReaderClient) BillingDetails(ctx context.Context, address tronaddress.Address) (bd BillingDetails, err error) {
	res, err := c.r.CallContract(address, "getBilling", nil)
	if err != nil {
		return bd, fmt.Errorf("failed to call contract: %w", err)
	}

	op, ok := res["observationPaymentGjuels"].(uint32)
	if !ok {
		return bd, fmt.Errorf("expected observationPaymentGjuels %+v to be of type uint32, got %T", res["observationPaymentGjuels"], res["observationPaymentGjuels"])
	}
	tp, ok := res["transmissionPaymentGjuels"].(uint32)
	if !ok {
		return bd, fmt.Errorf("expected transmissionPaymentGjuels %+v to be of type uint32, got %T", res["transmissionPaymentGjuels"], res["transmissionPaymentGjuels"])
	}

	return BillingDetails{
		ObservationPaymentGJuels:  op,
		TransmissionPaymentGJuels: tp,
	}, nil
}

func (c *OCR2ReaderClient) LatestConfigDetails(ctx context.Context, address tronaddress.Address) (ccd ContractConfigDetails, err error) {
	res, err := c.r.CallContract(address, "latestConfigDetails", nil)
	if err != nil {
		return ccd, fmt.Errorf("couldn't call the contract: %w", err)
	}

	blockNumUint32, ok := res["blockNumber"].(uint32)
	if !ok {
		return ccd, fmt.Errorf("expected blockNumber %+v to be of type uint32, got %T", res["blockNumber"], res["blockNumber"])
	}
	digest, ok := res["configDigest"].([32]byte)
	if !ok {
		return ccd, fmt.Errorf("expected configDigest %+v to be of type [32]byte, got %T", res["configDigest"], res["configDigest"])
	}

	return ContractConfigDetails{
		Block:  uint64(blockNumUint32),
		Digest: digest,
	}, nil
}

func (c *OCR2ReaderClient) LatestTransmissionDetails(ctx context.Context, address tronaddress.Address) (td TransmissionDetails, err error) {
	res, err := c.r.CallContract(address, "latestTransmissionDetails", nil)
	if err != nil {
		return td, fmt.Errorf("couldn't call the contract: %w", err)
	}

	configDigest, ok := res["configDigest"].([32]byte)
	if !ok {
		return td, fmt.Errorf("expected configDigest %+v to be of type [32]byte, got %T", res["configDigest"], res["configDigest"])
	}
	epoch, ok := res["epoch"].(uint32)
	if !ok {
		return td, fmt.Errorf("expected epoch %+v to be of type uint32, got %T", res["epoch"], res["epoch"])
	}
	round, ok := res["round"].(uint8)
	if !ok {
		return td, fmt.Errorf("expected round %+v to be of type uint8, got %T", res["round"], res["round"])
	}
	latestAnswer, ok := res["latestAnswer_"].(*big.Int)
	if !ok {
		return td, fmt.Errorf("expected latestAnswer %+v to be of type *big.Int, got %T", res["latestAnswer_"], res["latestAnswer_"])
	}
	latestTimestamp := res["latestTimestamp_"].(uint64)
	if !ok {
		return td, fmt.Errorf("expected latestTimestamp %+v to be of type uint64, got %T", res["latestTimestamp_"], res["latestTimestamp_"])
	}

	td = TransmissionDetails{
		Digest:          configDigest,
		Epoch:           epoch,
		Round:           round,
		LatestAnswer:    latestAnswer,
		LatestTimestamp: time.Unix(int64(latestTimestamp), 0),
	}

	return td, nil
}

func (c *OCR2ReaderClient) LatestRoundData(ctx context.Context, address tronaddress.Address) (round RoundData, err error) {
	res, err := c.r.CallContract(address, "latestRoundData", nil)
	if err != nil {
		return round, fmt.Errorf("couldn't call the contract: %w", err)
	}

	roundID, ok := res["roundId"].(*big.Int)
	if !ok {
		return round, fmt.Errorf("expected roundId %+v to be of type *big.Int, got %T", res["roundId"], res["roundId"])
	}
	answer, ok := res["answer"].(*big.Int)
	if !ok {
		return round, fmt.Errorf("expected answer %+v to be of type *big.Int, got %T", res["answer"], res["answer"])
	}
	startedAt, ok := res["startedAt"].(*big.Int)
	if !ok {
		return round, fmt.Errorf("expected startedAt %+v to be of type *big.Int, got %T", res["startedAt"], res["startedAt"])
	}
	updatedAt, ok := res["updatedAt"].(*big.Int)
	if !ok {
		return round, fmt.Errorf("expected updatedAt %+v to be of type *big.Int, got %T", res["updatedAt"], res["updatedAt"])
	}

	round = RoundData{
		RoundID:   uint32(roundID.Uint64()),
		Answer:    answer,
		StartedAt: time.Unix(startedAt.Int64(), 0),
		UpdatedAt: time.Unix(updatedAt.Int64(), 0),
	}
	return round, nil
}

func (c *OCR2ReaderClient) LinkAvailableForPayment(ctx context.Context, address tronaddress.Address) (*big.Int, error) {
	res, err := c.r.CallContract(address, "linkAvailableForPayment", nil)
	if err != nil {
		return nil, fmt.Errorf("couldn't call the contract: %w", err)
	}

	availableBalance, ok := res["availableBalance"].(*big.Int)
	if !ok {
		return nil, fmt.Errorf("expected availableBalance %+v to be of type big.Int, got %T", res["availableBalance"], res["availableBalance"])
	}

	return availableBalance, nil
}

func (c *OCR2ReaderClient) ConfigFromEventAt(ctx context.Context, address tronaddress.Address, blockNum uint64) (cc ContractConfig, err error) {
	events, err := c.r.GetEventsFromBlock(address, "ConfigSet", blockNum)
	if err != nil {
		return cc, fmt.Errorf("failed to fetch ConfigSet event logs: %w", err)
	}
	if len(events) != 1 {
		return cc, fmt.Errorf("expected to find at exactly one ConfigSet event in block %d for address %s but found %d", blockNum, address, len(events))
	}

	cfg := events[0]

	configDigest, ok := cfg["configDigest"].([32]byte)
	if !ok {
		return cc, fmt.Errorf("expected configDigest %+v to be of type bytes32, got %T", cfg["configDigest"], cfg["configDigest"])
	}
	configCount, ok := cfg["configCount"].(uint64)
	if !ok {
		return cc, fmt.Errorf("expected configCount %+v to be of type uint64, got %T", cfg["configCount"], cfg["configCount"])
	}
	signers, ok := cfg["signers"].([]common.Address)
	if !ok {
		return cc, fmt.Errorf("expected signers %+v to be of type []common.Address, got %T", cfg["signers"], cfg["signers"])
	}
	transmitters, ok := cfg["transmitters"].([]common.Address)
	if !ok {
		return cc, fmt.Errorf("expected transmitters %+v to be of type []common.Address, got %T", cfg["transmitters"], cfg["transmitters"])
	}
	f, ok := cfg["f"].(uint8)
	if !ok {
		return cc, fmt.Errorf("expected f %+v to be of type uint8, got %T", cfg["f"], cfg["f"])
	}
	onchainConfig, ok := cfg["onchainConfig"].([]byte)
	if !ok {
		return cc, fmt.Errorf("expected onchainConfig %+v to be of type []byte, got %T", cfg["onchainConfig"], cfg["onchainConfig"])
	}
	offchainConfigVersion, ok := cfg["offchainConfigVersion"].(uint64)
	if !ok {
		return cc, fmt.Errorf("expected offchainConfigVersion %+v to be of type uint64, got %T", cfg["offchainConfigVersion"], cfg["offchainConfigVersion"])
	}
	offchainConfig, ok := cfg["offchainConfig"].([]byte)
	if !ok {
		return cc, fmt.Errorf("expected offchainConfig %+v to be of type []byte, got %T", cfg["offchainConfig"], cfg["offchainConfig"])
	}

	// EVM format hex addresses, so that the OCR2 key in the DON can match against it, and because ecrecover returns this value
	// and can then look it up in the signers map in the aggregator contract
	var parsedSigners []types.OnchainPublicKey
	for _, s := range signers {
		parsedSigners = append(parsedSigners, types.OnchainPublicKey(s.Bytes()))
	}

	// TRON format addresses, must match the FromAccount value in ContractTransmitter
	var parsedTransmitters []types.Account
	for _, t := range transmitters {
		parsedTransmitters = append(parsedTransmitters, types.Account(relayer.EVMToTronAddress(t).String()))
	}

	cc = ContractConfig{
		Config: types.ContractConfig{
			ConfigDigest:          types.ConfigDigest(configDigest),
			ConfigCount:           configCount,
			Signers:               parsedSigners,
			Transmitters:          parsedTransmitters,
			F:                     f,
			OnchainConfig:         onchainConfig,
			OffchainConfigVersion: offchainConfigVersion,
			OffchainConfig:        offchainConfig,
		},
		ConfigBlock: blockNum,
	}

	return
}
