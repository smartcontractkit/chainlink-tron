package ocr2

import (
	"context"
	"fmt"
	"math/big"
	"time"

	tronaddress "github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	relayer "github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
)

//go:generate mockery --name OCR2Reader --output ./mocks/
type OCR2Reader interface { //nolint:revive
	LatestConfigDetails(context.Context, tronaddress.Address) (ContractConfigDetails, error)
	LatestTransmissionDetails(context.Context, tronaddress.Address) (TransmissionDetails, error)
	LatestRoundData(context.Context, tronaddress.Address) (RoundData, error)
	LinkAvailableForPayment(context.Context, tronaddress.Address) (*big.Int, error)
	ConfigFromEventAt(context.Context, tronaddress.Address, uint64) (ContractConfig, error)
	// NewTransmissionsFromEventsAt(context.Context, tronaddress.Address, uint64) ([]NewTransmissionEvent, error) // is this needed?
	BillingDetails(context.Context, tronaddress.Address) (BillingDetails, error)

	BaseReader() relayer.Reader
}

var _ OCR2Reader = (*OCR2ReaderClient)(nil)

type OCR2ReaderClient struct {
	r    relayer.Reader
	lggr logger.Logger
}

func NewOCR2Reader(reader relayer.Reader, lggr logger.Logger) *OCR2ReaderClient {
	return &OCR2ReaderClient{
		r:    reader,
		lggr: lggr,
	}
}

func (c *OCR2ReaderClient) BaseReader() relayer.Reader {
	return c.r
}

func (c *OCR2ReaderClient) BillingDetails(ctx context.Context, address tronaddress.Address) (bd BillingDetails, err error) {
	res, err := c.r.CallContract(address, "billing", nil)
	if err != nil {
		return bd, fmt.Errorf("failed to call contract: %w", err)
	}

	op, ok := res["observationPaymentGjuels"].(uint64)
	if !ok {
		return bd, fmt.Errorf("cannot convert observationPaymentGjuels %+v to uint64", res["observationPaymentGjuels"])
	}
	tp, ok := res["transmissionPaymentGjuels"].(uint64)
	if !ok {
		return bd, fmt.Errorf("cannot convert transmissionPaymentGjuels %+v to uint64", res["transmissionPaymentGjuels"])
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
	fmt.Println(res)

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
	fmt.Println(res)

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
	eventLogs, err := c.r.GetEventLogsFromBlock(address, "ConfigSet", blockNum)
	if err != nil {
		return cc, fmt.Errorf("failed to fetch ConfigSet event logs: %w", err)
	}
	if len(eventLogs) == 0 {
		return cc, fmt.Errorf("expected to find at least one ConfigSet event in block %d for address %s but found %d", blockNum, address, len(eventLogs))
	}

	// todo: parse logs into ContractConfig

	return
}

// NewTransmissionsFromEventsAt finds events of type new_transmission emitted by the contract address in a given block number.
func (c *OCR2ReaderClient) NewTransmissionsFromEventsAt(ctx context.Context, address tronaddress.Address, blockNum uint64) (events []NewTransmissionEvent, err error) {
	eventLogs, err := c.r.GetEventLogsFromBlock(address, "NewTransmission", blockNum)
	if err != nil {
		return events, fmt.Errorf("failed to fetch NewTransmission event logs: %w", err)
	}
	if len(eventLogs) == 0 {
		return events, fmt.Errorf("expected to find at least one NewTransmission event in block %d for address %s but found %d", blockNum, address, len(eventLogs))
	}

	// todo: parse logs into NewTransmissionEvent

	return
}
