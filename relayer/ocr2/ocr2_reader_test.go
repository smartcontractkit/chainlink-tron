package ocr2_test

import (
	"bytes"
	"context"
	"strconv"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/mocks"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/ocr2"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/reader"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
)

func TestOCR2Reader(t *testing.T) {
	testLogger, _ := logger.TestObserved(t, zapcore.DebugLevel)

	grpcClient := mocks.NewGrpcClient(t)
	ocr2AggregatorAbi, _ := contract.JSONtoABI(testutils.TRON_OCR2_AGGREGATOR_ABI)
	grpcClient.On("GetContractABI", mock.Anything).Maybe().Return(ocr2AggregatorAbi, nil)

	readerClient := reader.NewReader(grpcClient, testLogger)
	ocr2Reader := ocr2.NewOCR2Reader(readerClient, testLogger)

	t.Run("LatestConfigDetails", func(t *testing.T) {
		configCount := 1
		blockNumber := 12345
		configDigest := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
		constContractRes, err := abi.GetPaddedParam([]any{
			"uint32", strconv.FormatUint(uint64(configCount), 10),
			"uint32", strconv.FormatUint(uint64(blockNumber), 10),
			"bytes32", configDigest,
		})
		require.NoError(t, err)
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Unset()
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&api.TransactionExtention{
			Result:         &api.Return{Result: true},
			ConstantResult: [][]byte{constContractRes},
		}, nil)

		res, err := ocr2Reader.LatestConfigDetails(context.TODO(), nil)
		require.NoError(t, err)
		require.Equal(t, uint64(blockNumber), res.Block)
		require.Equal(t, configDigest, res.Digest.String())
	})

	t.Run("LatestTransmissionDetails", func(t *testing.T) {
		configDigest := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
		epoch := 1
		round := 4
		latestAnswer := "123456789"
		latestTimestamp := 87654
		constContractRes, err := abi.GetPaddedParam([]any{
			"bytes32", configDigest, // configDigest
			"uint32", strconv.FormatUint(uint64(epoch), 10), // epoch
			"uint8", strconv.FormatUint(uint64(round), 10), // round
			"int192", latestAnswer, // latestAnswer
			"uint64", strconv.FormatUint(uint64(latestTimestamp), 10), // latestTimestamp
		})
		require.NoError(t, err)
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Unset()
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&api.TransactionExtention{
			Result:         &api.Return{Result: true},
			ConstantResult: [][]byte{constContractRes},
		}, nil)

		res, err := ocr2Reader.LatestTransmissionDetails(context.TODO(), nil)
		require.NoError(t, err)
		require.Equal(t, configDigest, res.Digest.String())
		require.Equal(t, uint32(epoch), res.Epoch)
		require.Equal(t, uint8(round), res.Round)
		require.Equal(t, latestAnswer, res.LatestAnswer.String())
		require.Equal(t, int64(latestTimestamp), res.LatestTimestamp.Unix())
	})

	t.Run("LatestRoundData", func(t *testing.T) {
		roundId := 1
		answer := "123456789"
		startedAt := 87654
		updatedAt := 87666
		answeredInRound := 1
		constContractRes, err := abi.GetPaddedParam([]any{
			"uint80", strconv.FormatUint(uint64(roundId), 10),
			"int256", answer,
			"uint256", strconv.FormatUint(uint64(startedAt), 10),
			"uint256", strconv.FormatUint(uint64(updatedAt), 10),
			"uint80", strconv.FormatUint(uint64(answeredInRound), 10),
		})
		require.NoError(t, err)
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Unset()
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&api.TransactionExtention{
			Result:         &api.Return{Result: true},
			ConstantResult: [][]byte{constContractRes},
		}, nil)

		res, err := ocr2Reader.LatestRoundData(context.TODO(), nil)
		require.NoError(t, err)
		require.Equal(t, uint32(roundId), res.RoundID)
		require.Equal(t, answer, res.Answer.String())
		require.Equal(t, int64(startedAt), res.StartedAt.Unix())
		require.Equal(t, int64(updatedAt), res.UpdatedAt.Unix())
	})

	t.Run("LinkAvailableForPayment", func(t *testing.T) {
		availableBalance := "123456789"
		constContractRes, err := abi.GetPaddedParam([]any{
			"int256", availableBalance,
		})
		require.NoError(t, err)
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Unset()
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&api.TransactionExtention{
			Result:         &api.Return{Result: true},
			ConstantResult: [][]byte{constContractRes},
		}, nil)

		res, err := ocr2Reader.LinkAvailableForPayment(context.TODO(), nil)
		require.NoError(t, err)
		require.Equal(t, availableBalance, res.String())
	})

	t.Run("BillingDetails", func(t *testing.T) {
		maximumGasPriceGwei := 123
		reasonableGasPriceGwei := 456
		observationPaymentGjuels := 567
		transmissionPaymentGjuels := 789
		accountingGas := 111
		constContractRes, err := abi.GetPaddedParam([]any{
			"uint32", strconv.FormatUint(uint64(maximumGasPriceGwei), 10),
			"uint32", strconv.FormatUint(uint64(reasonableGasPriceGwei), 10),
			"uint32", strconv.FormatUint(uint64(observationPaymentGjuels), 10),
			"uint32", strconv.FormatUint(uint64(transmissionPaymentGjuels), 10),
			"uint32", strconv.FormatUint(uint64(accountingGas), 10),
		})
		require.NoError(t, err)
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Unset()
		grpcClient.On("TriggerConstantContract", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(&api.TransactionExtention{
			Result:         &api.Return{Result: true},
			ConstantResult: [][]byte{constContractRes},
		}, nil)

		res, err := ocr2Reader.BillingDetails(context.TODO(), nil)
		require.NoError(t, err)
		require.Equal(t, uint32(observationPaymentGjuels), res.ObservationPaymentGJuels)
		require.Equal(t, uint32(transmissionPaymentGjuels), res.TransmissionPaymentGJuels)
	})

	t.Run("ConfigFromEventAt", func(t *testing.T) {
		prevConfigBlockNumber := 12344
		configDigest := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
		configCount := 1
		signers := []interface{}{sdk.TRON_ZERO_ADDR_B58}
		transmitters := []interface{}{sdk.TRON_ZERO_ADDR_B58}
		f := 3
		onchainConfig := []byte{8, 9, 10, 11}
		offchainConfigVersion := 2
		offchainConfig := []byte{12, 13, 14, 15}
		encodedData, err := abi.GetPaddedParam([]any{
			"uint32", strconv.FormatUint(uint64(prevConfigBlockNumber), 10),
			"bytes32", configDigest,
			"uint64", strconv.FormatUint(uint64(configCount), 10),
			"address[]", signers,
			"address[]", transmitters,
			"uint8", strconv.FormatUint(uint64(f), 10),
			"bytes", onchainConfig,
			"uint64", strconv.FormatUint(uint64(offchainConfigVersion), 10),
			"bytes", offchainConfig,
		})
		require.NoError(t, err)
		grpcClient.On("GetBlockByNum", mock.Anything).Return(&api.BlockExtention{
			BlockHeader: &core.BlockHeader{
				RawData: &core.BlockHeaderRaw{
					Number: 12345,
				},
			},
			Transactions: []*api.TransactionExtention{
				{
					Logs: []*core.TransactionInfo_Log{
						{
							Address: []byte{0, 1, 2, 3},
							Topics:  [][]byte{relayer.GetEventTopicHash("ConfigSet(uint32,bytes32,uint64,address[],address[],uint8,bytes,uint64,bytes)")},
							Data:    encodedData,
						},
					},
				},
			},
		}, nil)

		res, err := ocr2Reader.ConfigFromEventAt(context.TODO(), []byte{0, 1, 2, 3}, 12345)
		require.NoError(t, err)
		require.Equal(t, uint64(12345), res.ConfigBlock)
		require.Equal(t, configDigest, res.Config.ConfigDigest.Hex())
		require.Equal(t, uint64(configCount), res.Config.ConfigCount)
		require.Equal(t, []types.OnchainPublicKey{bytes.Repeat([]byte{0}, 20)}, res.Config.Signers)
		require.Equal(t, []types.Account{"0x0000000000000000000000000000000000000000"}, res.Config.Transmitters)
		require.Equal(t, uint8(f), res.Config.F)
		require.Equal(t, onchainConfig, res.Config.OnchainConfig)
		require.Equal(t, uint64(offchainConfigVersion), res.Config.OffchainConfigVersion)
		require.Equal(t, offchainConfig, res.Config.OffchainConfig)
	})
}
