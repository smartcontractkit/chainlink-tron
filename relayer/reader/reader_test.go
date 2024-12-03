package reader_test

import (
	"encoding/hex"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/http/fullnode"
	"github.com/fbsobreira/gotron-sdk/pkg/http/soliditynode"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zapcore"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"

	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/mocks"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/reader"
)

var mockAbi = &common.JSONABI{
	Entrys: []common.Entry{
		{
			Name: "foo",
			Type: "function",
			Inputs: []common.EntryInput{
				{
					Name: "a",
					Type: "uint64",
				},
				{
					Name: "b",
					Type: "uint64",
				},
			},
			Outputs: []common.EntryOutput{
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

var mockConstantContractResponse = &soliditynode.TriggerConstantContractResponse{
	Result: soliditynode.ReturnEnergyEstimate{Result: true},
	// a packed (u64, u32) result as expected
	ConstantResult: []string{"000000000000000000000000000000000000000000000000000000000000007b00000000000000000000000000000000000000000000000000000000000001c8"},
	Transaction: &common.ExecutedTransaction{
		Transaction: common.Transaction{
			RawData: common.RawData{
				Timestamp:    123,
				Expiration:   456,
				RefBlockHash: "abc",
				FeeLimit:     789,
			},
		},
	},
	EnergyUsed: 1000,
}

var mockContractAddress = []byte{0, 1, 2, 3}

var mockBlock = &soliditynode.Block{
	BlockHeader: &soliditynode.BlockHeader{
		RawData: &soliditynode.BlockHeaderRaw{
			Number: 1,
		},
	},
	Transactions: []common.ExecutedTransaction{
		{
			Transaction: common.Transaction{
				RawData: common.RawData{
					Contract: []common.Contract{{Parameter: common.Parameter{
						TypeUrl: "type.googleapis.com/protocol.TriggerSmartContract",
						Value: common.ParameterValue{
							ContractAddress: hex.EncodeToString(mockContractAddress),
						},
					}}},
					Timestamp:    123,
					Expiration:   456,
					RefBlockHash: "abc",
					FeeLimit:     789,
				},
			},
		},
	},
}

func TestReader(t *testing.T) {
	// setup
	testLogger, _ := logger.TestObserved(t, zapcore.DebugLevel)
	fullNodeClient := mocks.NewFullNodeClient(t)

	t.Run("LatestBlockHeight", func(t *testing.T) {
		fullNodeClient.On(
			"GetNowBlock",
		).Return(mockBlock, nil).Once()
		reader := reader.NewReader(fullNodeClient, testLogger)

		blockHeight, err := reader.LatestBlockHeight()
		require.NoError(t, err)
		require.Equal(t, uint64(1), blockHeight)
	})

	t.Run("CallContract_NoParams", func(t *testing.T) {
		fullNodeClient.On(
			"GetContract",
			mock.Anything, // address
		).Return(&fullnode.GetContractResponse{
			ABI: mockAbi,
		}, nil).Once()
		fullNodeClient.On(
			"TriggerConstantContract",
			mock.Anything, // from
			mock.Anything, // contract
			mock.Anything, // method
			mock.Anything, // params
		).Return(mockConstantContractResponse, nil).Once()
		reader := reader.NewReader(fullNodeClient, testLogger)

		res, err := reader.CallContract(address.ZeroAddress, "foo", nil)
		require.NoError(t, err)
		require.Equal(t, uint64(123), res["a"])
		require.Equal(t, uint64(456), res["b"])
	})

	t.Run("CallContract_CachesABI", func(t *testing.T) {
		fullNodeClient.On(
			"GetContract",
			mock.Anything, // address
		).Return(&fullnode.GetContractResponse{
			ABI: mockAbi,
		}, nil).Once()
		fullNodeClient.On(
			"TriggerConstantContract",
			mock.Anything, // from
			mock.Anything, // contract
			mock.Anything, // method
			mock.Anything, // params
		).Return(mockConstantContractResponse, nil).Twice()
		reader := reader.NewReader(fullNodeClient, testLogger)

		_, err := reader.CallContract(address.ZeroAddress, "foo", nil)
		require.NoError(t, err)

		// should not call GetContract again
		_, err = reader.CallContract(address.ZeroAddress, "foo", nil)
		require.NoError(t, err)
	})

	t.Run("GetEventLogsFromBlock", func(t *testing.T) {
		mockAbi.Entrys = append(mockAbi.Entrys, common.Entry{
			Name: "event",
			Inputs: []common.EntryInput{
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
		fullNodeClient.On(
			"GetContract",
			mock.Anything, // address
		).Return(&fullnode.GetContractResponse{
			ABI: mockAbi,
		}, nil).Once()
		fullNodeClient.On(
			"GetBlockByNum",
			mock.Anything, // blockNum
		).Return(mockBlock, nil).Once()
		encodedData, err := abi.GetPaddedParam([]any{
			"uint64", "123",
			"uint64", "456",
			"uint32", "789",
		})
		require.NoError(t, err)
		fullNodeClient.On("GetTransactionInfoById", mock.Anything).Return(&soliditynode.TransactionInfo{
			Log: []soliditynode.Log{
				{
					Topics: []string{relayer.GetEventTopicHash("event(uint64,uint64,uint32)")},
					Data:   hex.EncodeToString(encodedData),
				},
			},
		}, nil)
		reader := reader.NewReader(fullNodeClient, testLogger)

		events, err := reader.GetEventsFromBlock(mockContractAddress, "event", 1)
		require.NoError(t, err)
		require.Len(t, events, 1)
		require.Equal(t, uint64(123), events[0]["a"])
		require.Equal(t, uint64(456), events[0]["b"])
		require.Equal(t, uint32(789), events[0]["c"])
	})
}
