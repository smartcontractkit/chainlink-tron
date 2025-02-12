package ocr2

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"

	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/types"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/utils"
)

var _ Tracker = (*transmissionsCache)(nil)
var _ median.MedianContract = (*transmissionsCache)(nil)

type transmissionsCache struct {
	transmissionDetails             TransmissionDetails
	consecutiveSkippedTransmissions int
	skippedTransmissionStartTime    time.Time
	tdLock                          sync.RWMutex
	tdLastCheckedAt                 time.Time

	stop, done chan struct{}

	reader median.MedianContract
	cfg    Config
	lggr   logger.Logger
}

func NewTransmissionsCache(cfg Config, reader median.MedianContract, lggr logger.Logger) *transmissionsCache {
	return &transmissionsCache{
		cfg:    cfg,
		reader: reader,
		lggr:   lggr,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
		transmissionDetails: TransmissionDetails{
			LatestAnswer: big.NewInt(0), // should always return at least 0 and not nil
		},
	}
}

func (c *transmissionsCache) updateTransmission(ctx context.Context) error {
	digest, epoch, round, answer, timestamp, err := c.reader.LatestTransmissionDetails(ctx)
	if err != nil {
		return fmt.Errorf("couldn't fetch latest transmission details: %w", err)
	}

	c.tdLock.Lock()
	defer c.tdLock.Unlock()
	c.tdLastCheckedAt = time.Now()

	td := TransmissionDetails{
		Digest:          digest,
		Epoch:           epoch,
		Round:           round,
		LatestAnswer:    answer,
		LatestTimestamp: timestamp,
	}

	// If timestamp from latest transmission details is zero, skip the cache update as the transmit
	// transaction has yet to be included in a block, although the fullnode api still returns the chain
	// state as if it has been executed. We effectively treat such a case as if we have not yet seen
	// the newest transmission, and instead will wait until the transmit transaction has been included
	// in a block which usually happens within a few seconds. Note: updating the transmission cache with
	// a zero timestamp would cause issues in OCR2, triggering the deltaC timeout.
	if timestamp.Unix() == 0 {
		if c.consecutiveSkippedTransmissions == 0 {
			c.skippedTransmissionStartTime = time.Now() // start tracking time of first skipped transmission
		}
		secondsSinceFirstSkipped := time.Since(c.skippedTransmissionStartTime).Seconds()

		c.consecutiveSkippedTransmissions++ // increment counter for logging purposes

		loggerKeyValues := []interface{}{
			"consecutiveSkippedTransmissions", c.consecutiveSkippedTransmissions,
			"secondsSinceFirstSkipped", secondsSinceFirstSkipped,
			"newTransmission", td,
		}
		if c.consecutiveSkippedTransmissions > 1 {
			// warn if we have skipped multiple transmissions in a row
			c.lggr.Warnw("transmission cache not updated consecutively: latestTimestamp is 0", loggerKeyValues...)
		} else {
			// otherwise just log as debug - single transmission skips are common
			c.lggr.Debugw("transmission cache not updated: latestTimestamp is 0", loggerKeyValues...)
		}

		return nil
	}

	// timestamp is non-zero, so reset skipped transmission tracking
	c.skippedTransmissionStartTime = time.Time{}
	c.consecutiveSkippedTransmissions = 0

	secondsSinceLastCacheUpdate := timestamp.Sub(c.transmissionDetails.LatestTimestamp).Seconds()
	c.transmissionDetails = td

	c.lggr.Debugw("transmission cache update", "secondsSinceLastCacheUpdate", secondsSinceLastCacheUpdate, "details", c.transmissionDetails)

	return nil
}

func (c *transmissionsCache) Start() error {
	ctx, cancel := utils.ContextFromChan(c.stop)
	defer cancel()
	if err := c.updateTransmission(ctx); err != nil {
		c.lggr.Warnf("failed to populate initial transmission details: %w", err)
	}
	go c.poll()
	return nil
}

func (c *transmissionsCache) Close() error {
	close(c.stop)
	return nil
}

func (c *transmissionsCache) poll() {
	defer close(c.done)
	tick := time.After(0)
	for {
		select {
		case <-c.stop:
			return
		case <-tick:
			ctx, cancel := utils.ContextFromChan(c.stop)

			if err := c.updateTransmission(ctx); err != nil {
				c.lggr.Errorf("Failed to update transmission: %w", err)
			}
			cancel()

			tick = time.After(utils.WithJitter(c.cfg.OCR2CachePollPeriod()))
		}
	}
}

func (c *transmissionsCache) LatestTransmissionDetails(
	ctx context.Context,
) (
	types.ConfigDigest,
	uint32,
	uint8,
	*big.Int,
	time.Time,
	error,
) {
	c.tdLock.RLock()
	defer c.tdLock.RUnlock()
	configDigest := c.transmissionDetails.Digest
	epoch := c.transmissionDetails.Epoch
	round := c.transmissionDetails.Round
	latestAnswer := c.transmissionDetails.LatestAnswer
	latestTimestamp := c.transmissionDetails.LatestTimestamp
	err := c.assertTransmissionsNotStale()
	return configDigest, epoch, round, latestAnswer, latestTimestamp, err
}

func (c *transmissionsCache) LatestRoundRequested(
	ctx context.Context,
	lookback time.Duration,
) (
	types.ConfigDigest,
	uint32,
	uint8,
	error,
) {
	c.tdLock.RLock()
	defer c.tdLock.RUnlock()
	configDigest := c.transmissionDetails.Digest
	epoch := c.transmissionDetails.Epoch
	round := c.transmissionDetails.Round
	err := c.assertTransmissionsNotStale()
	return configDigest, epoch, round, err
}

func (c *transmissionsCache) assertTransmissionsNotStale() error {
	if c.tdLastCheckedAt.IsZero() {
		return errors.New("transmissions cache not yet initialized")
	}

	if since := time.Since(c.tdLastCheckedAt); since > c.cfg.OCR2CacheTTL() {
		return fmt.Errorf("transmissions cache expired: checked last %s ago", since)
	}

	return nil
}
