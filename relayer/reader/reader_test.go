package reader_test

import (
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/api"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/mocks"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/reader"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/sdk"
)

var mockTxExtention = api.TransactionExtention{
	Transaction: &core.Transaction{
		RawData: &core.TransactionRaw{
			Timestamp:    123,
			Expiration:   456,
			RefBlockHash: []byte("abc"),
			FeeLimit:     789,
		},
	},
	Txid:           []byte("txid"),
	ConstantResult: [][]byte{{0x01}},
	Result:         &api.Return{Result: true},
	EnergyUsed:     1000,
	Logs: []*core.TransactionInfo_Log{
		{
			Address: []byte{0, 1, 2, 3},
			Topics:  [][]byte{{0x02}},
			Data:    []byte("data"),
		},
	},
}
var mockBlockExtention = api.BlockExtention{
	BlockHeader: &core.BlockHeader{
		RawData: &core.BlockHeaderRaw{
			Number: 1,
		},
	},
	Transactions: []*api.TransactionExtention{
		&mockTxExtention,
	},
}

var mockAbi = core.SmartContract_ABI{
	Entrys: []*core.SmartContract_ABI_Entry{
		{
			Name: "foo",
			Type: core.SmartContract_ABI_Entry_Function,
			Inputs: []*core.SmartContract_ABI_Entry_Param{
				{
					Name: "a",
					Type: "uint64",
				},
				{
					Name: "b",
					Type: "uint64",
				},
			},
			Outputs: []*core.SmartContract_ABI_Entry_Param{
				{
					Name: "a",
					Type: "uint64",
				},
				{
					Name: "b",
					Type: "uint64",
				},
			},
		},
	},
}

func TestReader(t *testing.T) {
	// setup
	testLogger, _ := logger.TestObserved(t, zapcore.DebugLevel)
	grpcClient := mocks.NewGrpcClient(t)

	t.Run("LatestBlockHeight", func(t *testing.T) {
		grpcClient.On(
			"GetNowBlock",
			mock.Anything, // ctx
		).Return(&mockBlockExtention, nil).Once()
		reader := reader.NewReader(grpcClient, testLogger)

		blockHeight, err := reader.LatestBlockHeight()
		require.NoError(t, err)
		require.Equal(t, uint64(1), blockHeight)
	})

	t.Run("CallContract_NoParams", func(t *testing.T) {
		grpcClient.On(
			"GetContractABI",
			mock.Anything, // address
		).Return(&mockAbi, nil).Once()
		constContractRes, _ := abi.GetPaddedParam([]any{"uint64", "123", "uint64", "456"})
		mockTxExtention.ConstantResult = [][]byte{constContractRes}
		grpcClient.On(
			"TriggerConstantContract",
			mock.Anything, // ctx
			mock.Anything, // contract
			mock.Anything, // method
			mock.Anything, // json
		).Return(&mockTxExtention, nil).Once()
		reader := reader.NewReader(grpcClient, testLogger)

		res, err := reader.CallContract(address.HexToAddress(sdk.TRON_ZERO_ADDR_HEX), "foo", nil)
		require.NoError(t, err)
		require.Equal(t, uint64(123), res["a"])
		require.Equal(t, uint64(456), res["b"])
	})

	t.Run("CallContract_CachesABI", func(t *testing.T) {
		grpcClient.On(
			"GetContractABI",
			mock.Anything, // address
		).Return(&mockAbi, nil).Once()
		constContractRes, _ := abi.GetPaddedParam([]any{"uint64", "123", "uint64", "456"})
		mockTxExtention.ConstantResult = [][]byte{constContractRes}
		grpcClient.On(
			"TriggerConstantContract",
			mock.Anything, // ctx
			mock.Anything, // contract
			mock.Anything, // method
			mock.Anything, // json
		).Return(&mockTxExtention, nil).Twice()
		reader := reader.NewReader(grpcClient, testLogger)

		_, err := reader.CallContract(address.HexToAddress(sdk.TRON_ZERO_ADDR_HEX), "foo", nil)
		require.NoError(t, err)

		// should not call GetContractABI again
		_, err = reader.CallContract(address.HexToAddress(sdk.TRON_ZERO_ADDR_HEX), "foo", nil)
		require.NoError(t, err)
	})

	t.Run("GetEventLogsFromBlock", func(t *testing.T) {
		mockAbi.Entrys = append(mockAbi.Entrys, &core.SmartContract_ABI_Entry{
			Name: "event",
			Inputs: []*core.SmartContract_ABI_Entry_Param{
				{
					Name: "a",
					Type: "uint64",
				},
				{
					Name: "b",
					Type: "uint64",
				},
				{
					Name: "c",
					Type: "uint32",
				},
			},
		})
		grpcClient.On(
			"GetContractABI",
			mock.Anything, // address
		).Return(&mockAbi, nil).Once()
		encodedData, _ := abi.GetPaddedParam([]any{
			"uint64", "123",
			"uint64", "456",
			"uint32", "789",
		})
		mockTxExtention.Logs = []*core.TransactionInfo_Log{
			{
				Address: []byte{0, 1, 2, 3},
				Topics:  [][]byte{relayer.GetEventTopicHash("event(uint64,uint64,uint32)")},
				Data:    encodedData,
			},
		}
		grpcClient.On(
			"GetBlockByNum",
			mock.Anything, // ctx
		).Return(&mockBlockExtention, nil).Once()
		reader := reader.NewReader(grpcClient, testLogger)

		events, err := reader.GetEventsFromBlock(address.Address{0, 1, 2, 3}, "event", 1)
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.Equal(t, uint64(123), events[0]["a"])
		require.Equal(t, uint64(456), events[0]["b"])
		require.Equal(t, uint32(789), events[0]["c"])
	})
}
