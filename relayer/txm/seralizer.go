package txm

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/fbsobreira/gotron-sdk/pkg/abi"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/http/common"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

type Serializer struct {
	TransactionType  core.Transaction_Contract_ContractType
	FromAddress      address.Address
	ContractAddress  address.Address
	Method           string
	Params           []any
	CallValueSun     int64
	FeeLimitSun      int64
	RefBlockBytes    []byte
	RefBlockHash     []byte
	ExpirationMillis int64
	TimestampMillis  int64
}

const DefaultExpirationMillis = 30_000 // 30 seconds

func (p *Serializer) BuildTransaction() (*common.Transaction, error) {
	if p.TransactionType != core.Transaction_Contract_TriggerSmartContract {
		return nil, fmt.Errorf("invalid transaction type: %d", p.TransactionType)
	}

	if len(p.RefBlockBytes) != 2 && len(p.RefBlockHash) != 8 {
		return nil, fmt.Errorf("invalid ref block bytes or hash")
	}

	callData, err := p.buildCallData()
	if err != nil {
		return nil, fmt.Errorf("failed to build call data: %+w", err)
	}

	smartContractCall := &core.TriggerSmartContract{
		OwnerAddress:    p.FromAddress.Bytes(),
		ContractAddress: p.ContractAddress.Bytes(),
		Data:            callData,
		CallValue:       p.CallValueSun,
	}

	callPayload, err := anypb.New(smartContractCall)
	if err != nil {
		return nil, fmt.Errorf("failed to create call payload: %+w", err)
	}

	contract := &core.Transaction_Contract{
		Parameter: callPayload,
		Type:      core.Transaction_Contract_TriggerSmartContract,
	}

	now := time.Now().UnixMilli()
	timestamp := p.TimestampMillis
	if timestamp == 0 {
		timestamp = now
	}

	expiration := p.ExpirationMillis
	if expiration == 0 {
		expiration = now + DefaultExpirationMillis
	}

	rawData := &core.TransactionRaw{
		Contract:      []*core.Transaction_Contract{contract},
		FeeLimit:      p.FeeLimitSun,
		Expiration:    expiration,
		Timestamp:     timestamp,
		RefBlockBytes: p.RefBlockBytes,
		RefBlockHash:  p.RefBlockHash,
	}

	rawBytes, err := proto.Marshal(rawData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal raw data: %+w", err)
	}

	hash := sha256.Sum256(rawBytes)
	txIDHex := hex.EncodeToString(hash[:])
	rawDataHex := hex.EncodeToString(rawBytes)

	commonRawData := common.RawData{
		Contract: []common.Contract{
			{
				Parameter: common.Parameter{
					Value: common.ParameterValue{
						OwnerAddress:    p.FromAddress.String(),
						ContractAddress: p.ContractAddress.String(),
						Data:            hex.EncodeToString(smartContractCall.Data),
						Amount:          p.CallValueSun,
					},
					TypeUrl: "type.googleapis.com/protocol.TriggerSmartContract",
				},
				Type: "TriggerSmartContract",
			},
		},
		RefBlockBytes: hex.EncodeToString(p.RefBlockBytes),
		RefBlockHash:  hex.EncodeToString(p.RefBlockHash),
		Expiration:    expiration,
		FeeLimit:      p.FeeLimitSun,
		Timestamp:     timestamp,
	}

	return &common.Transaction{
		Visible:    true,
		TxID:       txIDHex,
		RawData:    commonRawData,
		RawDataHex: rawDataHex,
	}, nil
}

func (p *Serializer) buildCallData() ([]byte, error) {
	parsed, err := abi.Pack(p.Method, p.Params)
	if err != nil {
		return nil, fmt.Errorf("failed to pack params: %+w", err)
	}
	return parsed, nil
}
