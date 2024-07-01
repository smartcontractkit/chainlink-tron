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

func (c *contractReader) LatestConfigDetails(ctx context.Context) (changedInBlock uint64, configDigest types.ConfigDigest, err error) {
	resp, err := c.reader.LatestConfigDetails(ctx, c.address)
	if err != nil {
		return changedInBlock, configDigest, fmt.Errorf("couldn't get latest config details: %w", err)
	}

	changedInBlock = resp.Block
	configDigest = resp.Digest

	return
}

func (c *contractReader) LatestConfig(ctx context.Context, changedInBlock uint64) (config types.ContractConfig, err error) {
	resp, err := c.reader.ConfigFromEventAt(ctx, c.address, changedInBlock)
	if err != nil {
		return config, fmt.Errorf("couldn't get latest config: %w", err)
	}

	config = resp.Config

	return
}

func (c *contractReader) LatestBlockHeight(ctx context.Context) (blockHeight uint64, err error) {
	return c.reader.BaseReader().LatestBlockHeight()
}

func (c *contractReader) LatestTransmissionDetails(
	ctx context.Context,
) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	latestAnswer *big.Int,
	latestTimestamp time.Time,
	err error,
) {
	transmissionDetails, err := c.reader.LatestTransmissionDetails(ctx, c.address)
	if err != nil {
		err = fmt.Errorf("couldn't get transmission details: %w", err)
	}

	configDigest = transmissionDetails.Digest
	epoch = transmissionDetails.Epoch
	round = transmissionDetails.Round
	latestAnswer = transmissionDetails.LatestAnswer
	latestTimestamp = transmissionDetails.LatestTimestamp

	return
}

func (c *contractReader) LatestRoundRequested(
	ctx context.Context,
	lookback time.Duration,
) (
	configDigest types.ConfigDigest,
	epoch uint32,
	round uint8,
	err error,
) {
	transmissionDetails, err := c.reader.LatestTransmissionDetails(ctx, c.address)
	if err != nil {
		err = fmt.Errorf("couldn't get transmission details: %w", err)
	}

	configDigest = transmissionDetails.Digest
	epoch = transmissionDetails.Epoch
	round = transmissionDetails.Round

	return
}

func (c *contractReader) LatestBillingDetails(ctx context.Context) (bd BillingDetails, err error) {
	bd, err = c.reader.BillingDetails(ctx, c.address)
	if err != nil {
		err = fmt.Errorf("couldn't get billing details: %w", err)
	}

	return
}
