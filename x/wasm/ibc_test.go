package wasm_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// define constants used for testing
const (
	testClientIDA = "testclientIDA"
	testClientIDB = "testClientIDb"

	testConnection = "testconnectionatob"
	testPort1      = "bank"
	testPort2      = "testportid"
	testChannel1   = "firstchannel"
	testChannel2   = "secondchannel"
)

// define variables used for testing
var (
	testAddr1, _ = sdk.AccAddressFromBech32("cosmos1scqhwpgsmr6vmztaa7suurfl52my6nd2kmrudl")
	testAddr2, _ = sdk.AccAddressFromBech32("cosmos1scqhwpgsmr6vmztaa7suurfl52my6nd2kmrujl")

	testCoins, _ = sdk.ParseCoins("100atom")
	prefixCoins  = sdk.NewCoins(sdk.NewCoin("bank/firstchannel/atom", sdk.NewInt(100)))
	prefixCoins2 = sdk.NewCoins(sdk.NewCoin("testportid/secondchannel/atom", sdk.NewInt(100)))
)

type IBCTestSuite struct {
	suite.Suite

	chainA *TestChain
	chainB *TestChain

	cleanupA func()
	cleanupB func()
}

func (suite *IBCTestSuite) SetupTest() {
	suite.chainA, suite.cleanupA = NewTestChain(testClientIDA)
	suite.chainB, suite.cleanupB = NewTestChain(testClientIDB)
}

func (suite *IBCTestSuite) TearDownTest() {
	suite.cleanupA()
	suite.cleanupB()
}

func (suite *IBCTestSuite) TestBindPorts() {
	suite.T().Logf("To be implemented")
}

func TestIBCTestSuite(t *testing.T) {
	suite.Run(t, new(IBCTestSuite))
}
