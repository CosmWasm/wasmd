package keeper_test

import (
	"encoding/binary"
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/CosmWasm/wasmd/x/wasm"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authkeeper "github.com/cosmos/cosmos-sdk/x/auth/keeper"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/cosmos/cosmos-sdk/x/ibc/04-channel/types"
	host "github.com/cosmos/cosmos-sdk/x/ibc/24-host"
	"github.com/stretchr/testify/require"
	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/crypto"
	"github.com/tendermint/tendermint/crypto/ed25519"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/app/integration"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/internal/keeper"
)

func TestBindingPortOnInstantiate(t *testing.T) {
	app := CreateTestApp(t)
	ctx := app.BaseApp.NewContext(false, header(10))

	accKeeper, keeper, bankKeeper := app.AccountKeeper, app.WasmKeeper, app.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	contractID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "", nil)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()

	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	// create with no balance is legal
	addr, err := keeper.Instantiate(ctx, contractID, creator, nil, initMsgBz, "demo contract 1", nil)
	require.NoError(t, err)
	require.Equal(t, "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5", addr.String())

	// ensure we bound the port
	owner, _, err := app.IBCKeeper.PortKeeper.LookupModuleByPort(ctx, keeper.GetContractInfo(ctx, addr).IBCPortID)
	require.NoError(t, err)
	require.Equal(t, "wasm", owner)

	// create a second contract should give yet another portID (and different address)
	addr, err = keeper.Instantiate(ctx, contractID, creator, nil, initMsgBz, "demo contract 2", nil)
	require.NoError(t, err)
	require.NotEqual(t, "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5", addr.String())

	portID2 := wasmkeeper.PortIDForContract(addr)
	owner, _, err = app.IBCKeeper.PortKeeper.LookupModuleByPort(ctx, portID2)
	require.NoError(t, err)
	require.Equal(t, "wasm", owner)

}

func TestRelay(t *testing.T) {
	// a chain with a contract instantiated and a channel

	// when our test-contract sends the MsgWasmIBCCall for an outgoing ibc call
	// then a WasmIBCContractPacketData is sent to the channel
	// and a WasmIBCContractPacketAcknowledgement stored

	// when the relayer picks up WasmIBCContractPacketData and returns an ack to
	// our chain
	// then the ack is passed to the contract with the origin WasmIBCContractPacketData

	wasmApp := CreateTestApp(t)
	ctx := wasmApp.BaseApp.NewContext(false, header(10))
	contractAddr := withAContractInstance(t, ctx, wasmApp)
	portID := wasmApp.WasmKeeper.GetContractInfo(ctx, contractAddr).IBCPortID

	mockChannelKeeper := NewMockChannelKeeper(wasmApp.WasmKeeper.ChannelKeeper)
	wasmApp.WasmKeeper.ChannelKeeper = mockChannelKeeper
	withCapabilities(t, ctx, channelID, portID, wasmApp)

	myContract := MockContract(t, contractAddr, wasmApp)
	myContract.DoIBCCall(ctx, channelID, portID)
	// then
	require.Len(t, mockChannelKeeper.received, 1)

	// new scenario:
	mockChannelKeeper.Reset()
	// when we receive an incoming ibc packet

	packet := types.Packet{
		Sequence:           1,
		SourcePort:         counterpartyPortID,
		SourceChannel:      counterpartyChannelID,
		DestinationPort:    portID,
		DestinationChannel: channelID,
		Data:               []byte(`{"my":"data""}`),
		TimeoutHeight:      100,
		TimeoutTimestamp:   0,
	}

	ibcHandler := wasm.NewIBCHandler(wasmApp.WasmKeeper)
	_, ack, err := ibcHandler.OnRecvPacket(ctx, packet)
	require.NoError(t, err)
	require.NotEmpty(t, ack)
	//
	res, err := ibcHandler.OnAcknowledgementPacket(ctx, packet, ack)
	require.NoError(t, err)
	_ = res
}

func withCapabilities(t *testing.T, ctx sdk.Context, channelID, portID string, wasmApp *app.WasmApp) {
	capName := host.ChannelCapabilityPath(portID, channelID)
	cap, err := wasmApp.ScopedIBCKeeper.NewCapability(ctx, capName)
	require.NoError(t, err)
	err = wasmApp.ScopedWasmKeeper.ClaimCapability(ctx, cap, capName)
	require.NoError(t, err)
}

func withAContractInstance(t *testing.T, ctx sdk.Context, app *app.WasmApp) sdk.AccAddress {
	t.Helper()
	accKeeper, keeper, bankKeeper := app.AccountKeeper, app.WasmKeeper, app.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	contractID, err := keeper.Create(ctx, creator, wasmCode, "https://github.com/CosmWasm/wasmd/blob/master/x/wasm/testdata/escrow.wasm", "", nil)
	require.NoError(t, err)

	_, _, bob := keyPubAddr()
	_, _, fred := keyPubAddr()

	initMsg := InitMsg{
		Verifier:    fred,
		Beneficiary: bob,
	}
	initMsgBz, err := json.Marshal(initMsg)
	require.NoError(t, err)

	// create with no balance is legal
	addr, err := keeper.Instantiate(ctx, contractID, creator, nil, initMsgBz, "demo contract 1", nil)
	require.NoError(t, err)

	return addr
}

// This should replace CreateTestInput when possible (likely after CosmWasm 1.0 is merged into this branch)
func CreateTestApp(t *testing.T) *app.WasmApp {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	t.Cleanup(func() { os.RemoveAll(tempDir) })

	return integration.Setup(false, tempDir)
}

func header(height int64) abci.Header {
	return abci.Header{
		Height: height,
		Time:   time.Date(2020, time.April, 22, 12, 0, 0, 0, time.UTC).Add(time.Second * time.Duration(height)),
	}
}

// copied from keeper_test.go as we are a different package...

type InitMsg struct {
	Verifier    sdk.AccAddress `json:"verifier"`
	Beneficiary sdk.AccAddress `json:"beneficiary"`
}

func createFakeFundedAccount(t *testing.T, ctx sdk.Context, am authkeeper.AccountKeeper, bank bankkeeper.Keeper, coins sdk.Coins) sdk.AccAddress {
	_, _, addr := keyPubAddr()
	acc := am.NewAccountWithAddress(ctx, addr)
	am.SetAccount(ctx, acc)
	require.NoError(t, bank.SetBalances(ctx, addr, coins))
	return addr
}

var keyCounter uint64 = 0

// we need to make this deterministic (same every test run), as encoded address size and thus gas cost,
// depends on the actual bytes (due to ugly CanonicalAddress encoding)
func keyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	keyCounter++
	seed := make([]byte, 8)
	binary.BigEndian.PutUint64(seed, keyCounter)

	key := ed25519.GenPrivKeyFromSecret(seed)
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}
