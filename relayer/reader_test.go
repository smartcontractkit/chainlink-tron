package relayer

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/testutils"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"
	"go.uber.org/zap/zaptest/observer"
)

func setup(t *testing.T, client *testutils.MockClient) (*ReaderClient, *observer.ObservedLogs) {
	testLogger, observedlogs := logger.TestObserved(t, zapcore.DebugLevel)
	reader := NewReader(client, testLogger)

	return reader, observedlogs
}

func TestReader_LatestBlockHeight(t *testing.T) {
	grpcClient := testutils.NewMockClient()
	readerClient, _ := setup(t, grpcClient)
	blockHeight, err := readerClient.LatestBlockHeight()
	require.NoError(t, err)
	require.Equal(t, uint64(1), blockHeight)
}

func TestReader_CallContract_NoParams(t *testing.T) {
	constContractRes, _ := abi.GetPaddedParam([]abi.Param{{"uint64": "123"}, {"uint64": "456"}})
	grpcClient := testutils.NewMockClient()
	grpcClient.SetTriggerConstantContractResp([][]byte{constContractRes}, nil)
	readerClient, _ := setup(t, grpcClient)

	addr := address.HexToAddress(TRON_ZERO_ADDR_HEX)
	res, err := readerClient.CallContract(addr, "foo", []map[string]string{})
	require.NoError(t, err)
	require.Equal(t, uint64(123), res["a"])
	require.Equal(t, uint64(456), res["b"])
}

func TestReader_GetEventLogsFromBlock(t *testing.T) {
	addr := address.Address{0, 1, 2, 3}
	topicHash := GetEventTopicHash("foo(uint64,uint64)")
	grpcClient := testutils.NewMockClient()
	grpcClient.SetGetBlockByNumResp(&api.TransactionExtention{
		Logs: []*core.TransactionInfo_Log{
			{
				Address: addr.Bytes(),
				Topics:  [][]byte{topicHash},
			},
		},
	}, nil)
	readerClient, _ := setup(t, grpcClient)

	eventLogs, err := readerClient.GetEventLogsFromBlock(addr, "foo", 1)
	require.NoError(t, err)
	require.Len(t, eventLogs, 1)
	require.Equal(t, addr.Bytes(), eventLogs[0].Address)
	require.Equal(t, topicHash, eventLogs[0].Topics[0])
}
