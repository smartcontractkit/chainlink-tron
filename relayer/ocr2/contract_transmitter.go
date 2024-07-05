package ocr2

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/chains/evmutil"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

type ContractTransmitter interface {
	services.Service
	ocrtypes.ContractTransmitter
}

var _ ContractTransmitter = (*contractTransmitter)(nil)

type contractTransmitter struct {
	transmissionsCache *transmissionsCache
	contractAddress    address.Address
	senderAddress      address.Address
	txm                *txm.TronTxm
	lggr               logger.Logger
}

func NewOCRContractTransmitter(
	ctx context.Context,
	transmissionsCache *transmissionsCache,
	contractAddress address.Address,
	senderAddress address.Address,
	txm *txm.TronTxm,
	lggr logger.Logger,
) *contractTransmitter {
	return &contractTransmitter{
		contractAddress:    contractAddress,
		txm:                txm,
		senderAddress:      senderAddress,
		transmissionsCache: transmissionsCache,
		lggr:               logger.Named(lggr, "OCRContractTransmitter"),
	}
}

// Transmit sends the report to the on-chain smart contract's Transmit method.
func (oc *contractTransmitter) Transmit(ctx context.Context, reportCtx ocrtypes.ReportContext, report ocrtypes.Report, signatures []ocrtypes.AttributedOnchainSignature) error {
	var reportCtxStr string
	rawReportCtx := evmutil.RawReportContext(reportCtx)
	for _, r := range rawReportCtx {
		reportCtxStr = reportCtxStr + hex.EncodeToString(r[:])
	}

	var rs [][]byte
	var ss [][]byte
	var vs [32]byte
	if len(signatures) > 32 {
		return errors.New("too many signatures, maximum is 32")
	}
	for i, as := range signatures {
		r, s, v, err := evmutil.SplitSignature(as.Signature)
		if err != nil {
			panic("eventTransmit(ev): error in SplitSignature")
		}
		rs = append(rs, r[:])
		ss = append(ss, s[:])
		vs[i] = v
	}

	oc.lggr.Debugw("Transmitting report", "report", hex.EncodeToString(report), "rawReportCtx", rawReportCtx, "contractAddress", oc.contractAddress)

	// build params
	reportStr := "0x" + reportCtxStr + hex.EncodeToString(report)
	rsStr := relayer.ByteArrayToStr(rs)
	ssStr := relayer.ByteArrayToStr(ss)
	vsStr := "0x" + hex.EncodeToString(vs[:])
	params := []any{
		"bytes", reportStr,
		"bytes32[]", rsStr,
		"bytes32[]", ssStr,
		"bytes32", vsStr,
	}

	return oc.txm.Enqueue(oc.senderAddress.String(), oc.contractAddress.String(), "transmit", params...)
}

func (oc *contractTransmitter) LatestConfigDigestAndEpoch(
	ctx context.Context,
) (
	configDigest ocrtypes.ConfigDigest,
	epoch uint32,
	err error,
) {
	configDigest, epoch, _, _, _, err = oc.transmissionsCache.LatestTransmissionDetails(ctx)
	if err != nil {
		err = fmt.Errorf("couldn't fetch latest transmission details: %w", err)
	}
	return
}

// FromAccount returns the account from which the transmitter invokes the contract
func (oc *contractTransmitter) FromAccount() (ocrtypes.Account, error) {
	return ocrtypes.Account(oc.senderAddress.String()), nil
}

func (oc *contractTransmitter) Start(ctx context.Context) error { return nil }
func (oc *contractTransmitter) Close() error                    { return nil }

// Has no state/lifecycle so it's always healthy and ready
func (oc *contractTransmitter) Ready() error { return nil }
func (oc *contractTransmitter) HealthReport() map[string]error {
	return map[string]error{oc.Name(): nil}
}
func (oc *contractTransmitter) Name() string { return oc.lggr.Name() }
