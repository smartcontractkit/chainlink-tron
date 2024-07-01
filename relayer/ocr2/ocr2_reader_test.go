package ocr2

import (
	"context"
	"strconv"
	"testing"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/contract"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func setup(t *testing.T, client *testutils.MockClient) (OCR2Reader, *observer.ObservedLogs) {
	testLogger, observedlogs := logger.TestObserved(t, zapcore.DebugLevel)

	// set client get abi resp
	ocr2AggregatorAbi, err := contract.JSONtoABI(testutils.TRON_OCR2_AGGREGATOR_ABI)
	require.NoError(t, err)
	client.SetGetContractAbiResp(ocr2AggregatorAbi, nil)

	reader := relayer.NewReader(client, testLogger)
	ocr2Reader := NewOCR2Reader(reader, testLogger)

	return ocr2Reader, observedlogs
}

func TestOCR2Reader_LatestConfigDetails(t *testing.T) {
	configCount := 1
	blockNumber := 12345
	configDigest := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	constContractRes, _ := abi.GetPaddedParam([]abi.Param{
		{"uint32": strconv.FormatUint(uint64(configCount), 10)},
		{"uint32": strconv.FormatUint(uint64(blockNumber), 10)},
		{"bytes32": configDigest},
	})
	grpcClient := testutils.NewMockClient()
	grpcClient.SetTriggerConstantContractResp([][]byte{constContractRes}, nil)
	ocr2Reader, _ := setup(t, grpcClient)
	res, err := ocr2Reader.LatestConfigDetails(context.TODO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, uint64(blockNumber), res.Block)
	require.Equal(t, configDigest, res.Digest.String())
}

func TestOCR2Reader_LatestTransmissionDetails(t *testing.T) {
	configDigest := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	epoch := 1
	round := 4
	latestAnswer := "123456789"
	latestTimestamp := 87654
	constContractRes, _ := abi.GetPaddedParam([]abi.Param{
		{"bytes32": configDigest},                                   // configDigest
		{"uint32": strconv.FormatUint(uint64(epoch), 10)},           // epoch
		{"uint8": strconv.FormatUint(uint64(round), 10)},            // round
		{"int192": latestAnswer},                                    // latestAnswer
		{"uint64": strconv.FormatUint(uint64(latestTimestamp), 10)}, // latestTimestamp
	})
	grpcClient := testutils.NewMockClient()
	grpcClient.SetTriggerConstantContractResp([][]byte{constContractRes}, nil)
	ocr2Reader, _ := setup(t, grpcClient)
	res, err := ocr2Reader.LatestTransmissionDetails(context.TODO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, configDigest, res.Digest.String())
	require.Equal(t, uint32(epoch), res.Epoch)
	require.Equal(t, uint8(round), res.Round)
	require.Equal(t, latestAnswer, res.LatestAnswer.String())
	require.Equal(t, time.Unix(int64(latestTimestamp), 0), res.LatestTimestamp)
}

func TestOCR2Reader_LatestRoundData(t *testing.T) {
	roundId := 1
	answer := "123456789"
	startedAt := 87654
	updatedAt := 87666
	answeredInRound := 1
	constContractRes, _ := abi.GetPaddedParam([]abi.Param{
		{"uint80": strconv.FormatUint(uint64(roundId), 10)},
		{"int256": answer},
		{"uint256": strconv.FormatUint(uint64(startedAt), 10)},
		{"uint256": strconv.FormatUint(uint64(updatedAt), 10)},
		{"uint80": strconv.FormatUint(uint64(answeredInRound), 10)},
	})
	grpcClient := testutils.NewMockClient()
	grpcClient.SetTriggerConstantContractResp([][]byte{constContractRes}, nil)
	ocr2Reader, _ := setup(t, grpcClient)
	res, err := ocr2Reader.LatestRoundData(context.TODO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, uint32(roundId), res.RoundID)
	require.Equal(t, answer, res.Answer.String())
	require.Equal(t, time.Unix(int64(startedAt), 0), res.StartedAt)
	require.Equal(t, time.Unix(int64(updatedAt), 0), res.UpdatedAt)
}

func TestOCR2Reader_LinkAvailableForPayment(t *testing.T) {
	availableBalance := "123456789"
	constContractRes, _ := abi.GetPaddedParam([]abi.Param{
		{"int256": availableBalance},
	})
	grpcClient := testutils.NewMockClient()
	grpcClient.SetTriggerConstantContractResp([][]byte{constContractRes}, nil)
	ocr2Reader, _ := setup(t, grpcClient)
	res, err := ocr2Reader.LinkAvailableForPayment(context.TODO(), nil)
	if err != nil {
		t.Fatal(err)
	}
	require.Equal(t, availableBalance, res.String())
}

// todo: test ConfigFromEventAt
// todo: test NewTransmissionsFromEventsAt
