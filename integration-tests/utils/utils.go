package utils

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"math/big"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/fbsobreira/gotron-sdk/pkg/keystore"
	"github.com/fbsobreira/gotron-sdk/pkg/proto/core"
	"github.com/pborman/uuid"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-common/pkg/loop"
	relayer_txm "github.com/smartcontractkit/chainlink-internal-integrations/tron/relayer/txm"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median"
	"github.com/smartcontractkit/libocr/offchainreporting2/reportingplugin/median/evmreportcodec"
	"github.com/stretchr/testify/require"
)

// Enum of TestEnv types
type TestEnv string

type Account struct {
	Address    string
	PublicKey  *ecdsa.PublicKey
	PrivateKey *ecdsa.PrivateKey
	Keystore   *TestKeystore
}

const (
	TestnetShasta TestEnv = "shasta"
	TestnetNile   TestEnv = "nile"
	Devnet        TestEnv = "devnet"
)

func GenerateRandomAccount() (string, *ecdsa.PrivateKey) {
	accountKey := createKey(rand.Reader)
	randomAddress := accountKey.Address.String()
	privateKey := accountKey.PrivateKey

	return randomAddress, privateKey
}

func WaitForTransactionConfirmation(t *testing.T, logger logger.Logger, txm *relayer_txm.TronTxm, waitTime time.Duration) {
	for {
		queueLen, unconfirmedLen := txm.InflightCount()
		logger.Debugw("Inflight count", "queued", queueLen, "unconfirmed", unconfirmedLen)
		if queueLen == 0 && unconfirmedLen == 0 {
			break
		}
		time.Sleep(waitTime)
	}
}

type CompiledArtifact struct {
	Bytecode string    `json:"bytecode"`
	ABI      []ABIItem `json:"abi"`
}

type ABIItem struct {
	Inputs          []ABIInput  `json:"inputs"`
	Name            string      `json:"name"`
	Outputs         []ABIOutput `json:"outputs"`
	StateMutability string      `json:"stateMutability"`
	Type            string      `json:"type"`
}

type ABIInput struct {
	InternalType string `json:"internalType"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

type ABIOutput struct {
	InternalType string `json:"internalType"`
	Name         string `json:"name"`
	Type         string `json:"type"`
}

func WaitForTransactionInfo(t *testing.T, txm *relayer_txm.TronTxm, txHash string, waitSecs int) *core.TransactionInfo {
	for i := 1; i <= waitSecs; i++ {
		txInfo, err := txm.GetClient().GetTransactionInfoByID(txHash)
		if err != nil {
			time.Sleep(time.Second)
			continue
		}
		return txInfo
	}

	require.FailNow(t, fmt.Sprintf("failed to wait for transaction: %s", txHash))

	return nil
}

type TestKeystore struct {
	Keys map[string]*ecdsa.PrivateKey
}

var _ loop.Keystore = &TestKeystore{}

func NewTestKeystore(address string, privateKey *ecdsa.PrivateKey) *TestKeystore {
	// TODO: we don't actually need a map if we only have a single key pair.
	keys := map[string]*ecdsa.PrivateKey{}
	keys[address] = privateKey
	return &TestKeystore{Keys: keys}
}

func (tk *TestKeystore) Sign(ctx context.Context, id string, hash []byte) ([]byte, error) {
	privateKey, ok := tk.Keys[id]
	if !ok {
		return nil, fmt.Errorf("no such key")
	}

	// used to check if the account exists.
	if hash == nil {
		return nil, nil
	}

	return crypto.Sign(hash, privateKey)
}

// SignReport signs the report after computing keccak256(abi.encode(keccak256(report), reportContext)).
func (tk *TestKeystore) SignReport(ctx context.Context, id string, report []byte, repctx ReportContext) ([]byte, error) {
	rawReportContext := RawReportContext(repctx)
	sigData := crypto.Keccak256(report)
	sigData = append(sigData, rawReportContext[0][:]...)
	sigData = append(sigData, rawReportContext[1][:]...)
	sigData = append(sigData, rawReportContext[2][:]...)
	return tk.Sign(ctx, id, crypto.Keccak256(sigData))
}

func (tk *TestKeystore) Accounts(ctx context.Context) ([]string, error) {
	accounts := make([]string, 0, len(tk.Keys))
	for id := range tk.Keys {
		accounts = append(accounts, id)
	}
	return accounts, nil
}

// this is copied from keystore.NewKeyFromDirectICAP, which keeps trying to
// recreate the key if it doesn't start with a 0 prefix and can take significantly longer.
// the function we need is keystore.newKey which is unfortunately private.
// ref: https://github.com/fbsobreira/gotron-sdk/blob/1e824406fe8ce02f2fec4c96629d122560a3598f/pkg/keystore/key.go#L146
func createKey(rand io.Reader) *keystore.Key {
	randBytes := make([]byte, 64)
	_, err := rand.Read(randBytes)
	if err != nil {
		panic("key generation: could not read from random source: " + err.Error())
	}
	reader := bytes.NewReader(randBytes)
	privateKeyECDSA, err := ecdsa.GenerateKey(crypto.S256(), reader)
	if err != nil {
		panic("key generation: ecdsa.GenerateKey failed: " + err.Error())
	}
	key := newKeyFromECDSA(privateKeyECDSA)

	return key
}

func newKeyFromECDSA(privateKeyECDSA *ecdsa.PrivateKey) *keystore.Key {
	id := uuid.NewRandom()
	key := &keystore.Key{
		ID:         id,
		Address:    address.PubkeyToAddress(privateKeyECDSA.PublicKey),
		PrivateKey: privateKeyECDSA,
	}
	return key
}

// Finds the closest git repo root, assuming that a directory with a .git directory is a git repo.
func findGitRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		gitDir := filepath.Join(currentDir, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return currentDir, nil
		}

		parentDir := filepath.Dir(currentDir)
		if parentDir == currentDir {
			return "", fmt.Errorf("no Git repository found")
		}

		currentDir = parentDir
	}
}

func StartTronNodeWithGenesisAccount(genesisAddress string) error {
	gitRoot, err := findGitRoot()
	if err != nil {
		return fmt.Errorf("failed to find Git root: %v", err)
	}

	scriptPath := filepath.Join(gitRoot, "./tron/scripts/java-tron.sh")
	cmd := exec.Command(scriptPath, genesisAddress)

	output, err := cmd.CombinedOutput()

	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			fmt.Printf("Failed to start java-tron, dumping output:\n%s\n", string(output))
			return fmt.Errorf("Failed to start java-tron, bad exit code: %v", exitError.ExitCode())
		}
		return fmt.Errorf("Failed to start java-tron: %+v", err)
	}

	return nil
}

// Setup genesis account for tests
func SetupTestGenesisAccount(t *testing.T) (genesisAddress string, genesisPrivateKey *ecdsa.PrivateKey, privateKeyHex string) {
	privateKeyHex = "TODO"

	if privateKeyHex == "" {
		genesisAddress, genesisPrivateKey = GenerateRandomAccount()
		privateKeyHex = hex.EncodeToString(crypto.FromECDSA(genesisPrivateKey))
	} else {
		privateKey, err := crypto.HexToECDSA(privateKeyHex)
		require.NoError(t, err)
		genesisAddress = address.PubkeyToAddress(privateKey.PublicKey).String()
		genesisPrivateKey = privateKey
	}

	return genesisAddress, genesisPrivateKey, privateKeyHex
}

// CreateRandomAccounts generates a specified number of random accounts for testing purposes.
func CreateRandomAccounts(t *testing.T, count int) []Account {
	accounts := make([]Account, count) // Pre-allocate slice with the required length
	for i := 0; i < count; i++ {
		address, privateKey := GenerateRandomAccount()
		keystore := NewTestKeystore(address, privateKey)
		accounts[i] = Account{
			Address:    address,
			PublicKey:  &privateKey.PublicKey,
			PrivateKey: privateKey,
			Keystore:   keystore,
		}
	}

	return accounts
}

// GetRSVFromSignature extracts r, s, and v values from the given signature.
// r = first 32 bytes of signature
// s = second 32 bytes of signature
// v = final 1 byte of signature
func GetRSVFromSignature(signature []byte) (r, s string, v byte, err error) {
	if len(signature) != 65 {
		return "", "", byte(0), errors.New("invalid signature length, expected 65 bytes")
	}

	r = ToHexString(signature[:32])
	s = ToHexString(signature[32:64])
	v = signature[64]

	return r, s, v, nil
}

// ToHexString converts a byte slice to a hex string prefixed with "0x".
func ToHexString(b []byte) string {
	return fmt.Sprintf("0x%s", hex.EncodeToString(b))
}

func ConvertSliceOf32BytesToHexStrings(data [][32]byte) []string {
	var hexStrings []string
	for _, item := range data {
		hexStrings = append(hexStrings, "0x"+hex.EncodeToString(item[:]))
	}
	return hexStrings
}

// Convert32ByteToHexString converts a [32]byte array to a hex string with 0x prefix.
func Convert32BytesToHexString(data [32]byte) string {
	return "0x" + hex.EncodeToString(data[:])
}

// Pad or trim rawVs to ensure it is exactly 32 bytes long
func PadToBytes32(rawVs []byte) []byte {
	if len(rawVs) > 32 {
		return rawVs[:32]
	}
	padded := make([]byte, 32)
	copy(padded, rawVs)
	return padded
}

// Digest of the configuration for a OCR2 protocol instance. The first two
// bytes indicate which config digester (typically specific to a targeted
// blockchain) was used to compute a ConfigDigest. This value is used as a
// domain separator between different protocol instances and is thus security
// critical. It should be the output of a cryptographic hash function over all
// relevant configuration fields as well as e.g. the address of the target
// contract/state accounts/...
type ConfigDigest [32]byte

// ReportTimestamp is the logical timestamp of a report.
type ReportTimestamp struct {
	ConfigDigest ConfigDigest
	Epoch        uint32
	Round        uint8
}

type ReportContext struct {
	ReportTimestamp
	// A hash over some data that is exchanged during execution of the offchain
	// protocol. The data itself is not needed onchain, but we still want to
	// include it in the signature that goes onchain.
	ExtraHash [32]byte
}

func RawReportContext(repctx ReportContext) [3][32]byte {
	rawRepctx := [3][32]byte{}
	copy(rawRepctx[0][:], repctx.ConfigDigest[:])
	binary.BigEndian.PutUint32(rawRepctx[1][32-5:32-1], repctx.Epoch)
	rawRepctx[1][31] = repctx.Round
	rawRepctx[2] = repctx.ExtraHash
	return rawRepctx
}

// Convert RawReportContext to hex string
func RawReportContextToHexString(rawRepctx [3][32]byte) [3]string {
	rawRepctxStr := [3]string{}
	for i, raw := range rawRepctx {
		rawRepctxStr[i] = "0x" + hex.EncodeToString(raw[:])
	}
	return rawRepctxStr
}

func SplitSignature(sig []byte) (r, s [32]byte, v byte, err error) {
	if len(sig) != 65 {
		return r, s, v, fmt.Errorf("SplitSignature: wrong size")
	}
	r = common.BytesToHash(sig[:32])
	s = common.BytesToHash(sig[32:64])
	v = sig[64]
	return r, s, v, nil
}

type ParsedAttributedObservation struct {
	Timestamp        uint32
	Value            *big.Int
	JuelsPerFeeCoin  *big.Int
	GasPriceSubunits *big.Int
	Observer         uint8
}

func GenerateOCR2Report() (hexReport string, medianObservation string, err error) {
	juelsPerFeeCoin := big.NewInt(1e18)

	// Oracle and their observations are generated here https://github.com/smartcontractkit/offchain-reporting/blob/master/lib/offchainreporting2plus/internal/ocr2/contract_tests/helpers_test.go#L97
	// And turned into struct here https://github.com/smartcontractkit/offchain-reporting/blob/7e6c241e684bd322516251d6b70f026ffecb6953/lib/offchainreporting2plus/internal/ocr2/contract_tests/helpers_test.go#L40
	paos := []median.ParsedAttributedObservation{
		{1750755864, big.NewInt(1271127070705242445), juelsPerFeeCoin, big.NewInt(0), 0},
		{901430628, big.NewInt(7142860741125246750), juelsPerFeeCoin, big.NewInt(0), 1},
		{3441600637, big.NewInt(6632808366083118734), juelsPerFeeCoin, big.NewInt(0), 2},
		{2535110149, big.NewInt(8911919877521849818), juelsPerFeeCoin, big.NewInt(0), 3},
		{3892996001, big.NewInt(786026922747303397), juelsPerFeeCoin, big.NewInt(0), 4},
		{632275502, big.NewInt(2731549183433775004), juelsPerFeeCoin, big.NewInt(0), 5},
		{3117570168, big.NewInt(7911589968318461182), juelsPerFeeCoin, big.NewInt(0), 6},
		{2894514909, big.NewInt(5776549002916223716), juelsPerFeeCoin, big.NewInt(0), 7},
		{810936509, big.NewInt(6200928152459444193), juelsPerFeeCoin, big.NewInt(0), 8},
		{1745543156, big.NewInt(6701302325764932302), juelsPerFeeCoin, big.NewInt(0), 9},
		{1480268637, big.NewInt(8154764892406459019), juelsPerFeeCoin, big.NewInt(0), 10},
		{4251749876, big.NewInt(2846500446905292057), juelsPerFeeCoin, big.NewInt(0), 11},
		{2841869438, big.NewInt(6297123943045257721), juelsPerFeeCoin, big.NewInt(0), 12},
		{2778158242, big.NewInt(4898235019263516856), juelsPerFeeCoin, big.NewInt(0), 13},
		{358910537, big.NewInt(538793199214391335), juelsPerFeeCoin, big.NewInt(0), 14},
		{1929768584, big.NewInt(1848167929414573296), juelsPerFeeCoin, big.NewInt(0), 15},
		{2097386607, big.NewInt(6045326344142570107), juelsPerFeeCoin, big.NewInt(0), 16},
		{1796893012, big.NewInt(6191080222126842088), juelsPerFeeCoin, big.NewInt(0), 17},
		{2325194541, big.NewInt(5164923635739237355), juelsPerFeeCoin, big.NewInt(0), 18},
		{806699139, big.NewInt(8639664443292681347), juelsPerFeeCoin, big.NewInt(0), 19},
		{1561217097, big.NewInt(1933969485392146004), juelsPerFeeCoin, big.NewInt(0), 20},
		{2577347004, big.NewInt(6988926203054418701), juelsPerFeeCoin, big.NewInt(0), 21},
		{637874287, big.NewInt(5165075787665716655), juelsPerFeeCoin, big.NewInt(0), 22},
		{2560416105, big.NewInt(6193326628629273182), juelsPerFeeCoin, big.NewInt(0), 23},
		{1636608432, big.NewInt(8221375515390474978), juelsPerFeeCoin, big.NewInt(0), 24},
		{1962450390, big.NewInt(6513272651572558066), juelsPerFeeCoin, big.NewInt(0), 25},
		{2760531991, big.NewInt(7814566626301179977), juelsPerFeeCoin, big.NewInt(0), 26},
		{3432267792, big.NewInt(4173452982976083880), juelsPerFeeCoin, big.NewInt(0), 27},
		{2808783005, big.NewInt(6740236431543500352), juelsPerFeeCoin, big.NewInt(0), 28},
		{2258741909, big.NewInt(4575545089208729518), juelsPerFeeCoin, big.NewInt(0), 29},
		{2051531540, big.NewInt(4567101290862106515), juelsPerFeeCoin, big.NewInt(0), 30},
	}

	// Extract the observations from paos
	observations := make([]*big.Int, len(paos))
	for i, pao := range paos {
		observations[i] = pao.Value
	}

	// Calculate the median observation
	// Sort the observations by value
	sort.Slice(paos, func(i, j int) bool {
		return paos[i].Value.Cmp(paos[j].Value) < 0
	})

	// Calculate the median observation
	medianIndex := len(paos) / 2
	median := paos[medianIndex].Value
	// Convert the median to hex format
	medianHex := fmt.Sprintf("0x%x", median)

	reportCodec := evmreportcodec.ReportCodec{}
	report, err := reportCodec.BuildReport(paos)
	if err != nil {
		return "", "", err
	}
	hexReport = "0x" + fmt.Sprintf("%x", report)

	return hexReport, medianHex, nil
}

// Create keccak256 hex string hash of function signature
func FunctionSignatureHash(functionSignature string) string {
	hash := crypto.Keccak256([]byte(functionSignature))
	return "0x" + hex.EncodeToString(hash)
}

// Returns base58 encoded Tron address from Ethereum address
func EthereumToTronAddressBase58(ethAddress common.Address) string {
	prefix := "41" + ethAddress.Hex()[2:]
	addr := address.HexToAddress(prefix)
	return addr.String()
}

// Converts a Tron base58 encoded address to an Ethereum address.
func TronAddressBase58ToEthereum(tronAddress string) (common.Address, error) {
	tronHexAddress, err := address.Base58ToAddress(tronAddress)
	if err != nil {
		return common.Address{}, fmt.Errorf("failed to convert Tron base58 address to hex: %v", err)
	}

	if len(tronHexAddress.Hex()) < 4 {
		return common.Address{}, fmt.Errorf("invalid Tron hex address: %s", tronHexAddress.Hex())
	}

	ethAddressHex := "0x" + tronHexAddress.Hex()[4:]

	ethAddress := common.HexToAddress(ethAddressHex)
	return ethAddress, nil
}
