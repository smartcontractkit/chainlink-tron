package ocr2

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"

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
// This is different from the EVM config digester because EVM expects the chain id to fit in a
// uint64:
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
	contractAddress  common.Address
	configDigestArgs abi.Arguments
}

var _ types.OffchainConfigDigester = &TRONOffchainConfigDigester{}

func NewOffchainConfigDigester(lggr logger.Logger, chainID *big.Int, contractAddress common.Address) (*TRONOffchainConfigDigester, error) {
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

func (d *TRONOffchainConfigDigester) ConfigDigest(cc types.ContractConfig) (types.ConfigDigest, error) {
	d.lggr.Error("DEBUG: called ConfigDigest")

	signers := []common.Address{}
	for i, signer := range cc.Signers {
		if len(signer) != 20 {
			return types.ConfigDigest{}, fmt.Errorf("%v-th evm signer should be a 20 byte address, but got %x", i, signer)
		}
		a := common.BytesToAddress(signer)
		signers = append(signers, a)
	}
	transmitters := []common.Address{}
	for i, transmitter := range cc.Transmitters {
		if !strings.HasPrefix(string(transmitter), "0x") || len(transmitter) != 42 || !common.IsHexAddress(string(transmitter)) {
			return types.ConfigDigest{}, fmt.Errorf("%v-th evm transmitter should be a 42 character Ethereum address string, but got '%v'", i, transmitter)
		}
		a := common.HexToAddress(string(transmitter))
		transmitters = append(transmitters, a)
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

func (d *TRONOffchainConfigDigester) ConfigDigestPrefix() (types.ConfigDigestPrefix, error) {
	return types.ConfigDigestPrefixEVM, nil
}

func (d *TRONOffchainConfigDigester) configDigestFromConfigData(
	chainId *big.Int,
	contractAddress common.Address,
	configCount uint64,
	signers []common.Address,
	transmitters []common.Address,
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
