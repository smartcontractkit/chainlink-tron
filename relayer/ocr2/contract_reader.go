package ocr2

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
)

// ContractReader implements both the median.MedianContract and ocrtypes.ContractConfigTracker interfaces
type ContractReader interface {
	types.ContractConfigTracker
	median.MedianContract
}

var _ ContractReader = (*contractReader)(nil)

type contractReader struct {
	address address.Address
	reader  OCR2Reader
	lggr    logger.Logger
}

func NewContractReader(address address.Address, reader OCR2Reader, lggr logger.Logger) ContractReader {
	return &contractReader{
		address: address,
		reader:  reader,
		lggr:    lggr,
	}
}

func (c *contractReader) Notify() <-chan struct{} {
	return nil
}

func (c *contractReader) LatestConfigDetails(ctx context.Context) (uint64, types.ConfigDigest, error) {
	resp, err := c.reader.LatestConfigDetails(ctx, c.address)
	if err != nil {
		return 0, types.ConfigDigest{}, fmt.Errorf("couldn't get latest config details: %w", err)
	}

	changedInBlock := resp.Block
	configDigest := resp.Digest

	return changedInBlock, configDigest, nil
}

func (c *contractReader) LatestConfig(ctx context.Context, changedInBlock uint64) (types.ContractConfig, error) {
	resp, err := c.reader.ConfigFromEventAt(ctx, c.address, changedInBlock)
	if err != nil {
		return types.ContractConfig{}, fmt.Errorf("couldn't get latest config: %w", err)
	}

	return resp.Config, nil
}

func (c *contractReader) LatestBlockHeight(ctx context.Context) (uint64, error) {
	return c.reader.BaseReader().LatestBlockHeight()
}

func (c *contractReader) LatestTransmissionDetails(
	ctx context.Context,
) (
	types.ConfigDigest,
	uint32,
	uint8,
	*big.Int,
	time.Time,
	error,
) {
	transmissionDetails, err := c.reader.LatestTransmissionDetails(ctx, c.address)
	if err != nil {
		return types.ConfigDigest{}, 0, 0, nil, time.Time{}, fmt.Errorf("couldn't get transmission details: %w", err)
	}

	configDigest := transmissionDetails.Digest
	epoch := transmissionDetails.Epoch
	round := transmissionDetails.Round
	latestAnswer := transmissionDetails.LatestAnswer
	latestTimestamp := transmissionDetails.LatestTimestamp

	return configDigest, epoch, round, latestAnswer, latestTimestamp, nil
}

func (c *contractReader) LatestRoundRequested(
	ctx context.Context,
	lookback time.Duration,
) (
	types.ConfigDigest,
	uint32,
	uint8,
	error,
) {
	requestedRound, err := c.reader.LatestRoundRequested(ctx, c.address, lookback)
	if err != nil {
		err = fmt.Errorf("couldn't get latestRoundRequested details: %w", err)
	}

	configDigest := requestedRound.Digest
	epoch := requestedRound.Epoch
	round := requestedRound.Round

	return configDigest, epoch, round, nil
}

func (c *contractReader) LatestBillingDetails(ctx context.Context) (BillingDetails, error) {
	bd, err := c.reader.BillingDetails(ctx, c.address)
	if err != nil {
		return BillingDetails{}, fmt.Errorf("couldn't get billing details: %w", err)
	}

	return bd, nil
}
