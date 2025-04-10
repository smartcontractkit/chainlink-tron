package ocr2

import (
	"context"
	"encoding/hex"
	"fmt"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/pkg/errors"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/services"
	"github.com/smartcontractkit/chainlink-tron/relayer/txm"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/chains/evmutil"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

type ContractTransmitter interface {
	services.Service
	ocrtypes.ContractTransmitter
}

var _ ContractTransmitter = (*contractTransmitter)(nil)

type transmitterOps struct {
	excludeSigs      bool
	ethereumKeystore bool
	isCCIPExec       bool
}

type contractTransmitter struct {
	transmissionsCache TransmissionsCache
	contractAddress    address.Address
	senderAddress      address.Address
	txm                *txm.TronTxm
	lggr               logger.Logger
	transmitterOptions *transmitterOps
}

func NewOCRContractTransmitter(
	ctx context.Context,
	transmissionsCache TransmissionsCache,
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
		transmitterOptions: &transmitterOps{
			excludeSigs:      false,
			ethereumKeystore: false,
			isCCIPExec:       false,
		},
	}
}

func (oc *contractTransmitter) WithExcludeSignatures() *contractTransmitter {
	oc.transmitterOptions.excludeSigs = true
	return oc
}

func (oc *contractTransmitter) WithEthereumKeystore() *contractTransmitter {
	oc.transmitterOptions.ethereumKeystore = true
	return oc
}

func (oc *contractTransmitter) WithCCIPExec() *contractTransmitter {
	oc.transmitterOptions.isCCIPExec = true
	return oc
}

// Transmit sends the report to the on-chain smart contract's Transmit method.
func (oc *contractTransmitter) Transmit(ctx context.Context, reportCtx ocrtypes.ReportContext, report ocrtypes.Report, signatures []ocrtypes.AttributedOnchainSignature) error {
	rawReportCtx := evmutil.RawReportContext(reportCtx)

	var rs [][32]byte
	var ss [][32]byte
	var vs [32]byte
	if len(signatures) > 32 {
		return errors.New("too many signatures, maximum is 32")
	}
	for i, as := range signatures {
		r, s, v, err := evmutil.SplitSignature(as.Signature)
		if err != nil {
			panic("eventTransmit(ev): error in SplitSignature")
		}

		if !oc.transmitterOptions.excludeSigs {
			rs = append(rs, r)
			ss = append(ss, s)
			vs[i] = v
		}
	}

	oc.lggr.Debugw("Transmitting report", "report", hex.EncodeToString(report), "rawReportCtx", rawReportCtx, "contractAddress", oc.contractAddress)

	// build params
	params := []any{
		"bytes32[3]", rawReportCtx,
		"bytes", []byte(report),
		"bytes32[]", rs,
		"bytes32[]", ss,
		"bytes32", vs,
	}

	request := &txm.TronTxmRequest{
		FromAddress:     oc.senderAddress,
		ContractAddress: oc.contractAddress,
		Method:          "transmit(bytes32[3],bytes,bytes32[],bytes32[],bytes32)",
		Params:          params,
	}
	return oc.txm.Enqueue(request)
}

func (oc *contractTransmitter) LatestConfigDigestAndEpoch(
	ctx context.Context,
) (
	ocrtypes.ConfigDigest,
	uint32,
	error,
) {
	configDigest, epoch, _, _, _, err := oc.transmissionsCache.LatestTransmissionDetails(ctx)
	if err != nil {
		return ocrtypes.ConfigDigest{}, 0, fmt.Errorf("couldn't fetch latest transmission details: %w", err)
	}
	return configDigest, epoch, nil
}

// FromAccount returns the account from which the transmitter invokes the contract
func (oc *contractTransmitter) FromAccount(ctx context.Context) (ocrtypes.Account, error) {
	if oc.transmitterOptions.ethereumKeystore {
		return ocrtypes.Account(oc.senderAddress.EthAddress().String()), nil
	}

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
