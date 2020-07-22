package wasm_test

import (
	"fmt"
	"testing"

	"github.com/CosmWasm/wasmd/x/wasm"
	connectiontypes "github.com/cosmos/cosmos-sdk/x/ibc/03-connection/types"
	channeltypes "github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	commitmenttypes "github.com/cosmos/cosmos-sdk/x/ibc/23-commitment/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
	"github.com/stretchr/testify/suite"
	abci "github.com/tendermint/tendermint/abci/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// define constants used for testing
const (
	testClientIDA = "testclientIDA"
	testClientIDB = "testClientIDb"

	testConnection = "testconnectionatob"
	testPort1      = "ibc-wasm"
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

func (suite *IBCTestSuite) TestReceivePacket() {
	suite.chainA.CreateClient(suite.chainB)
	suite.chainA.createConnection(testConnection, testConnection, testClientIDB, testClientIDA, connectiontypes.OPEN)
	suite.chainA.createChannel(testPort1, testChannel1, testPort2, testChannel2, channeltypes.OPEN, channeltypes.ORDERED, testConnection)
	suite.chainA.App.IBCKeeper.ChannelKeeper.SetNextSequenceSend(suite.chainA.GetContext(), testPort1, testChannel1, 1)

	capName := host.ChannelCapabilityPath(testPort1, testChannel1)
	cap, err := suite.chainA.App.ScopedIBCKeeper.NewCapability(suite.chainA.GetContext(), capName)
	suite.Require().NoError(err)
	err = suite.chainA.App.ScopedTransferKeeper.ClaimCapability(suite.chainA.GetContext(), cap, capName)
	suite.Require().NoError(err)

	var (
		addr1 sdk.AccAddress = make([]byte, sdk.AddrLen)
	)
	incomingWasmPacket := wasm.WasmIBCContractPacketData{
		Sender: addr1,
		Msg:    []byte("{}"),
	}
	payload, err := incomingWasmPacket.Marshal()
	suite.Require().NoError(err)
	packet := channeltypes.NewPacket(payload, 1, testPort1, testChannel1, testPort2, testChannel2, 100, 0)
	_ = channeltypes.NewMsgPacket(packet, []byte{}, 0, addr1)
	//tx := authtypes.NewStdTx([]sdk.Msg{msg},nil, }
	//_, r, err := suite.chainA.App.Deliver(&tx)
	//suite.Require().NoError(err)
	//suite.T().Log(r.Log)
}

func queryProof(chain *TestChain, key string) ([]byte, int64) {
	res := chain.App.Query(abci.RequestQuery{
		Path:  fmt.Sprintf("store/%s/key", host.StoreKey),
		Data:  []byte(key),
		Prove: true,
	})

	height := res.Height
	merkleProof := commitmenttypes.MerkleProof{
		Proof: res.Proof,
	}

	proof, err := chain.App.Codec().MarshalBinaryBare(&merkleProof)
	if err != nil {
		panic(err)
	}
	return proof, height
}

//func NextBlock(chain *TestChain) {
//	// set the last header to the current header
//	chain.LastHeader = chain.CreateTMClientHeader()
//
//	// increment the current header
//	chain.CurrentHeader = abci.Header{
//		Height:  chain.App.LastBlockHeight() + 1,
//		AppHash: chain.App.LastCommitID().Hash,
//		// NOTE: the time is increased by the coordinator to maintain time synchrony amongst
//		// chains.
//		Time: chain.CurrentHeader.Time,
//	}
//
//	chain.App.BeginBlock(abci.RequestBeginBlock{Header: chain.CurrentHeader})
//
//}

func TestIBCTestSuite(t *testing.T) {
	suite.Run(t, new(IBCTestSuite))
}
