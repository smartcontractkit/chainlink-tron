//go:build integration

// This file contains all the non data feeds related contract validation tests i.e peripheral contracts
package e2e_tests

import (
	"context"
	"crypto/ecdsa"
	"math/big"
	"strings"
	"testing"

	"github.com/fbsobreira/gotron-sdk/pkg/address"
	"github.com/smartcontractkit/chainlink-common/pkg/logger"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/contract"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/gauntlet"
	"github.com/smartcontractkit/chainlink-internal-integrations/tron/integration-tests/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type IntegrationTestSuite struct {
	suite.Suite
	ctx               context.Context
	logger            logger.Logger
	genesisAddress    string
	genesisPrivateKey *ecdsa.PrivateKey
	config            gauntlet.Config
	provider          gauntlet.Provider
	keystore          *utils.TestKeystore
	opcodeTest        *gauntlet.ContractTest
	linkTest          *gauntlet.ContractTest
}

// This allows us to parralelize the tests in the future
func TestPeripheryContracts(t *testing.T) {
	suite.Run(t, new(IntegrationTestSuite))
}

func (s *IntegrationTestSuite) setupTronInstance() {
	s.ctx = context.Background()
	s.logger = logger.Test(s.T())

	genesisAddress, genesisPrivateKey, genesisPrivateKeyHex := utils.SetupTestGenesisAccount(s.T())
	s.genesisAddress = genesisAddress
	s.genesisPrivateKey = genesisPrivateKey
	s.keystore = utils.NewTestKeystore(s.genesisAddress, s.genesisPrivateKey)
	s.logger.Debugw("Using genesis account", "address", s.genesisAddress)

	err := utils.StartTronNodeWithGenesisAccount(s.genesisAddress)
	require.NoError(s.T(), err)
	s.logger.Debugw("Started TRON node")

	s.config = gauntlet.NewDeploymentLocalTestConfig(genesisPrivateKeyHex)
	s.provider = gauntlet.NewProvider(s.ctx, &s.config)

	s.opcodeTest = gauntlet.NewContractTest("TestEvmOpcode", &s.config, s.provider, address.Address{})
	s.linkTest = gauntlet.NewContractTest("TestLinkToken", &s.config, s.provider, address.Address{})
}

func (s *IntegrationTestSuite) teardownTronInstance() {
	err := s.opcodeTest.Teardown(s.T(), s.logger)
	require.NoError(s.T(), err)

	err = s.linkTest.Teardown(s.T(), s.logger)
	require.NoError(s.T(), err)
}

// EVM Opcode contract tests
func (s *IntegrationTestSuite) TestEvmOpcode() {
	s.setupTronInstance()
	defer s.teardownTronInstance()

	s.logger.Debugw("Starting TestDeployAndInvokeOpcode")
	err := s.opcodeTest.Setup(s.ctx, contract.OpCodes)
	require.NoError(s.T(), err)

	s.opcodeTest.InvokeOperation(s.ctx, s.opcodeTest.ContractAddress().String(), contract.OpCodes, "test()")
}

func (s *IntegrationTestSuite) TestLinkToken() {
	s.setupTronInstance()
	defer s.teardownTronInstance()

	s.logger.Debugw("Starting TestDeployAndInvokeLink")
	err := s.linkTest.Setup(s.ctx, contract.LinkToken)
	require.NoError(s.T(), err)

	grantMintAndBurnRoles(s.T(), s.ctx, s.linkTest, s.genesisAddress, s.logger)

	mintAmount := big.NewInt(100000000000000)
	mintTokens(s.T(), s.ctx, s.linkTest, mintAmount, s.genesisAddress, s.logger)

	// Get balance after minting
	balance, err := getBalance(s.T(), s.ctx, s.linkTest, s.genesisAddress, s.logger)
	require.NoError(s.T(), err)
	require.Equal(s.T(), mintAmount, balance)

	// Approve token spending
	recipientAddress, _ := utils.GenerateRandomAccount()
	transferAmount := new(big.Int).Div(mintAmount, big.NewInt(2)) // half of minted amount
	approveTokenSpending(s.T(), s.ctx, s.linkTest, recipientAddress, transferAmount, s.logger)

	// Transfer tokens
	transferTokens(s.T(), s.ctx, s.linkTest, recipientAddress, transferAmount, s.logger)

	// Check balance of recipient address
	contractBalance, err := getBalance(s.T(), s.ctx, s.linkTest, recipientAddress, s.logger)
	require.NoError(s.T(), err)
	require.Equal(s.T(), transferAmount, contractBalance)

	// Increase Allowance for burning
	increaseAllowance(s.T(), s.ctx, s.linkTest, s.genesisAddress, transferAmount, s.logger)

	// Burn remaining tokens - half of minted amount
	burnTokens(s.T(), s.ctx, s.linkTest, s.genesisAddress, transferAmount, s.logger)

	// Check balance of contract address
	contractBalance, err = getBalance(s.T(), s.ctx, s.linkTest, s.genesisAddress, s.logger)
	require.NoError(s.T(), err)
	require.Equal(s.T(), big.NewInt(0), contractBalance)
}

/** Helper functions for LinkToken contract tests **/

func grantMintAndBurnRoles(t *testing.T, ctx context.Context, test *gauntlet.ContractTest, address string, logger logger.Logger) {
	args := []interface{}{
		address,
	}

	err := test.InvokeOperation(ctx, test.ContractAddress().String(), contract.LinkToken, GrantMintAndBurnRolesFunction, args...)
	require.NoError(t, err)
	logger.Debugw("Granted Mint and Burn Roles")
}

func mintTokens(t *testing.T, ctx context.Context, test *gauntlet.ContractTest, amount *big.Int, address string, logger logger.Logger) {
	args := []interface{}{
		address,
		amount,
	}

	err := test.InvokeOperation(ctx, test.ContractAddress().String(), contract.LinkToken, MintFunction, args...)
	require.NoError(t, err)
	logger.Debugw("Minted tokens")
}

func approveTokenSpending(t *testing.T, ctx context.Context, test *gauntlet.ContractTest, spender string, amount *big.Int, logger logger.Logger) {
	args := []interface{}{
		spender,
		amount,
	}

	err := test.InvokeOperation(ctx, test.ContractAddress().String(), contract.LinkToken, ApproveFunction, args...)
	require.NoError(t, err)
	logger.Debugw("Approved token spending")
}

func transferTokens(t *testing.T, ctx context.Context, test *gauntlet.ContractTest, recipient string, amount *big.Int, logger logger.Logger) {
	args := []interface{}{
		recipient,
		amount,
	}

	err := test.InvokeOperation(ctx, test.ContractAddress().String(), contract.LinkToken, TransferFunction, args...)
	require.NoError(t, err)
	logger.Debugw("Transferred tokens")
}

func burnTokens(t *testing.T, ctx context.Context, test *gauntlet.ContractTest, burntAddress string, amount *big.Int, logger logger.Logger) {
	args := []interface{}{
		burntAddress,
		amount,
	}

	err := test.InvokeOperation(ctx, test.ContractAddress().String(), contract.LinkToken, BurnFunction, args...)
	require.NoError(t, err)
	logger.Debugw("Burned remaining tokens")
}

func increaseAllowance(t *testing.T, ctx context.Context, test *gauntlet.ContractTest, burntAddress string, amount *big.Int, logger logger.Logger) {
	args := []interface{}{
		burntAddress,
		amount,
	}

	err := test.InvokeOperation(ctx, test.ContractAddress().String(), contract.LinkToken, IncreaseAllowanceFunction, args...)
	require.NoError(t, err)
	logger.Debugw("Increased Allowance")
}

func getBalance(t *testing.T, ctx context.Context, test *gauntlet.ContractTest, address string, logger logger.Logger) (*big.Int, error) {
	args := []interface{}{
		address,
	}

	balanceJSON, err := test.QueryContract(ctx, test.ContractAddress().String(), contract.LinkToken, BalanceOfFunction, "", args...)
	require.NoError(t, err)

	balanceHexString := balanceJSON.Get("hex").String()

	balanceHexString = strings.Replace(balanceHexString, `"`, "", -1)
	balanceHexString = strings.TrimPrefix(balanceHexString, "0x")

	balance := new(big.Int)
	_, ok := balance.SetString(balanceHexString, 16)
	require.True(t, ok)

	logger.Debugw("Checked balance", "address", address, "balance", balance)
	return balance, nil
}

const (
	LinkTokenContractName         = "LinkToken"
	GrantMintAndBurnRolesFunction = "grantMintAndBurnRoles(address)"
	MintFunction                  = "mint(address,uint256)"
	BalanceOfFunction             = "balanceOf(address)"
	ApproveFunction               = "approve(address,uint256)"
	TransferFunction              = "transfer(address,uint256)"
	BurnFunction                  = "burnFrom(address,uint256)"
	IncreaseAllowanceFunction     = "increaseAllowance(address,uint256)"
)
