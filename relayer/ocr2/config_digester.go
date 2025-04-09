package ocr2

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"

	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/libocr/offchainreporting2plus/types"
)

const functionABI = `[{
    "inputs": [
        {"name": "chainId", "type": "uint256"},
        {"name": "contractAddress", "type": "address"},
        {"name": "configCount", "type": "uint64"},
        {"name": "signers", "type": "address[]"},
        {"name": "transmitters", "type": "address[]"},
        {"name": "f", "type": "uint8"},
        {"name": "onchainConfig", "type": "bytes"},
        {"name": "offchainConfigVersion", "type": "uint64"},
        {"name": "offchainConfig", "type": "bytes"}
    ],
    "name": "_configDigestFromConfigData",
    "outputs": [{"name": "", "type": "bytes32"}],
    "type": "function"
}]`

// TRON offchain config digester.
//
// This is different from the EVM config digester because we store TRON format addresses as transmitters,
// and because EVM expects the chain id to fit in a uint64:
// https://github.com/smartcontractkit/libocr/blob/063ceef8c42eeadbe94221e55b8892690d36099a/offchainreporting2plus/chains/evmutil/config_digest.go#L28
//
// although the calculation onchain using `block.chainid` is a uint256:
// https://github.com/smartcontractkit/libocr/blob/063ceef8c42eeadbe94221e55b8892690d36099a/contract2/OCR2Abstract.sol#L72
//
// `block.chainid` in the TVM, depending on config, can be the entire genesis block id, or the last 4 bytes. for the former,
// it does not fit in a uint64.
type TRONOffchainConfigDigester struct {
	lggr             logger.Logger
	chainID          *big.Int
	contractAddress  ethcommon.Address
	configDigestArgs abi.Arguments
}

var _ types.OffchainConfigDigester = &TRONOffchainConfigDigester{}

func NewOffchainConfigDigester(lggr logger.Logger, chainID *big.Int, contractAddress ethcommon.Address) (*TRONOffchainConfigDigester, error) {
	parsedAbi, err := abi.JSON(strings.NewReader(functionABI))
	if err != nil {
		return nil, err
	}
	configDigestArgs := parsedAbi.Methods["_configDigestFromConfigData"].Inputs
	return &TRONOffchainConfigDigester{
		lggr,
		chainID,
		contractAddress,
		configDigestArgs,
	}, nil
}

func (d *TRONOffchainConfigDigester) ConfigDigest(ctx context.Context, cc types.ContractConfig) (types.ConfigDigest, error) {
	signers := []ethcommon.Address{}
	for i, signer := range cc.Signers {
		if len(signer) != ethcommon.AddressLength {
			return types.ConfigDigest{}, fmt.Errorf("%v-th signer should be a 20 byte hex address, but got %x", i, signer)
		}
		a := ethcommon.BytesToAddress(signer)
		signers = append(signers, a)
	}
	transmitters := []ethcommon.Address{}
	// These should be saved as EVM hex addresses on-chain, however this logic supports any valid Tron address format in case of a migration.
	for i, transmitter := range cc.Transmitters {
		address, err := address.StringToAddress(string(transmitter))
		if err != nil {
			return types.ConfigDigest{}, fmt.Errorf("%v-th transmitter should be a valid Tron address string, but got '%v'", i, transmitter)
		}
		transmitters = append(transmitters, address.EthAddress())
	}

	calculatedDigest, err := d.configDigestFromConfigData(
		d.chainID,
		d.contractAddress,
		cc.ConfigCount,
		signers,
		transmitters,
		cc.F,
		cc.OnchainConfig,
		cc.OffchainConfigVersion,
		cc.OffchainConfig,
	)

	if err != nil {
		return types.ConfigDigest{}, err
	}

	return calculatedDigest, nil
}

func (d *TRONOffchainConfigDigester) ConfigDigestPrefix(ctx context.Context) (types.ConfigDigestPrefix, error) {
	return types.ConfigDigestPrefixEVM, nil
}

func (d *TRONOffchainConfigDigester) configDigestFromConfigData(
	chainId *big.Int,
	contractAddress ethcommon.Address,
	configCount uint64,
	signers []ethcommon.Address,
	transmitters []ethcommon.Address,
	f uint8,
	onchainConfig []byte,
	offchainConfigVersion uint64,
	offchainConfig []byte,
) (types.ConfigDigest, error) {
	packed, err := d.configDigestArgs.Pack(
		chainId,
		contractAddress,
		configCount,
		signers,
		transmitters,
		f,
		onchainConfig,
		offchainConfigVersion,
		offchainConfig)
	if err != nil {
		return [32]byte{}, err
	}

	rawHash := crypto.Keccak256(packed)

	configDigest := types.ConfigDigest{}
	if n := copy(configDigest[:], rawHash); n != len(configDigest) {
		// assertion
		panic("copy too little data")
	}
	configDigest[0] = 0
	configDigest[1] = 1
	return configDigest, nil
}
