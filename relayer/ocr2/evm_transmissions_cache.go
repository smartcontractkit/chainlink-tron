package ocr2

import (
	"context"
	"database/sql"
	"errors"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-evm/pkg/client"
	"github.com/smartcontractkit/chainlink-evm/pkg/logpoller"
	"github.com/smartcontractkit/chainlink-evm/pkg/utils"
	ocrtypes "github.com/smartcontractkit/libocr/offchainreporting2/types"
)

// Provides a simple interface to cache the latest config digest and epoch from the OCR2 contract
type evmTransmissionsCache struct {
	contractAddress     common.Address
	client              client.Client
	contractABI         abi.ABI
	lp                  logpoller.LogPoller
	transmittedEventSig common.Hash
}

type evmContractReader interface {
	CallContract(ctx context.Context, call ethereum.CallMsg, blockNumber *big.Int) ([]byte, error)
}

func NewEVMTransmissionsCache(ctx context.Context, lggr logger.Logger, contractAddress common.Address, client client.Client, contractABI abi.ABI, lp logpoller.LogPoller, transmittedEventSig common.Hash) *evmTransmissionsCache {
	return &evmTransmissionsCache{
		contractAddress:     contractAddress,
		client:              client,
		contractABI:         contractABI,
		lp:                  lp,
		transmittedEventSig: transmittedEventSig,
	}
}

func (c *evmTransmissionsCache) LatestTransmissionDetails(ctx context.Context) (configDigest ocrtypes.ConfigDigest, epoch uint32, round uint8, latestAnswer *big.Int, latestTimestamp time.Time, err error) {
	// Uses the EVM Client to call the latestConfigDigestAndEpoch function on the contract, reuses the same logic thats in the EVM ContractTransmitter
	configDigest, epoch, err = getLatestConfigDigestAndEpoch(ctx, c.lp, c.contractAddress, c.contractABI, c.transmittedEventSig, c.client)
	return configDigest, epoch, 0, latestAnswer, latestTimestamp, nil
}

func (c *evmTransmissionsCache) LatestRoundRequested(ctx context.Context, lookback time.Duration) (ocrtypes.ConfigDigest, uint32, uint8, error) {
	return ocrtypes.ConfigDigest{}, 0, 0, nil
}

func callContract(ctx context.Context, addr common.Address, contractABI abi.ABI, method string, args []interface{}, caller evmContractReader) ([]interface{}, error) {
	input, err := contractABI.Pack(method, args...)
	if err != nil {
		return nil, err
	}
	output, err := caller.CallContract(ctx, ethereum.CallMsg{To: &addr, Data: input}, nil)
	if err != nil {
		return nil, err
	}
	return contractABI.Unpack(method, output)
}

func parseTransmitted(log []byte) ([32]byte, uint32, error) {
	var args abi.Arguments = []abi.Argument{
		{
			Name: "configDigest",
			Type: utils.MustAbiType("bytes32", nil),
		},
		{
			Name: "epoch",
			Type: utils.MustAbiType("uint32", nil),
		},
	}
	transmitted, err := args.Unpack(log)
	if err != nil {
		return [32]byte{}, 0, err
	}
	if len(transmitted) < 2 {
		return [32]byte{}, 0, errors.New("transmitted event log has too few arguments")
	}
	configDigest := *abi.ConvertType(transmitted[0], new([32]byte)).(*[32]byte)
	epoch := *abi.ConvertType(transmitted[1], new(uint32)).(*uint32)
	return configDigest, epoch, err
}

// Retrieves the latest config digest and epoch from the OCR2 contract.
func getLatestConfigDigestAndEpoch(ctx context.Context, lp logpoller.LogPoller, contractAddress common.Address, contractABI abi.ABI, eventSig common.Hash, caller evmContractReader) (ocrtypes.ConfigDigest, uint32, error) {
	latestConfigDigestAndEpoch, err := callContract(ctx, contractAddress, contractABI, "latestConfigDigestAndEpoch", nil, caller)
	if err != nil {
		return ocrtypes.ConfigDigest{}, 0, err
	}
	// Panic on these conversions erroring, would mean a broken contract.
	scanLogs := *abi.ConvertType(latestConfigDigestAndEpoch[0], new(bool)).(*bool)
	configDigest := *abi.ConvertType(latestConfigDigestAndEpoch[1], new([32]byte)).(*[32]byte)
	epoch := *abi.ConvertType(latestConfigDigestAndEpoch[2], new(uint32)).(*uint32)
	if !scanLogs {
		return configDigest, epoch, nil
	}

	// Otherwise, we have to scan for the logs.
	latest, err := lp.LatestLogByEventSigWithConfs(ctx, eventSig, contractAddress, 1)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			// No transmissions yet
			return configDigest, 0, nil
		}
		return ocrtypes.ConfigDigest{}, 0, err
	}

	return parseTransmitted(latest.Data)
}
