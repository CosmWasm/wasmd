package keeper

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/CosmWasm/go-cosmwasm"
	cosmwasmv1 "github.com/CosmWasm/go-cosmwasm/types"
	cosmwasmv2 "github.com/CosmWasm/wasmd/x/wasm/internal/keeper/cosmwasm"
	"github.com/cosmos/cosmos-sdk/store/prefix"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/stretchr/testify/require"
)

func TestMinter(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, keepers := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper
	totalSupply := types.NewSupply(sdk.NewCoins(sdk.NewInt64Coin("denom", 400000000)))
	bankKeeper.SetSupply(ctx, totalSupply)

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	creator := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, deposit)

	wasmCode, err := ioutil.ReadFile("./testdata/contract.wasm")
	require.NoError(t, err)

	// create any dummy contract to mock later
	codeID, err := keeper.Create(ctx, creator, wasmCode, "", "any/builder:tag", nil)
	require.NoError(t, err)
	// with random addresses
	initMsgBz := []byte(`{"verifier": "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5", "beneficiary":"cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5"}`)
	contractAddr, err := keeper.Instantiate(ctx, codeID, creator, nil, initMsgBz, "demo contract 3", deposit)
	require.NoError(t, err)
	require.Equal(t, "cosmos18vd8fpwxzck93qlwghaj6arh4p7c5n89uzcee5", contractAddr.String())

	MockContracts[contractAddr.String()] = &minterContract{t: t, contractAddr: contractAddr}
	fred := createFakeFundedAccount(t, ctx, accKeeper, bankKeeper, sdk.NewCoins(sdk.NewInt64Coin("denom", 5000)))

	// call execute to return a mint message
	_, err = keeper.Execute(ctx, contractAddr, fred, []byte(`mint`), nil)
	require.NoError(t, err)

	const contractsDenom = "wasm/5EFEC84ADDE58024AEF3A7BB9CD5945F91D180D0A4AC5AA2534BEF8F6194CD79"
	t.Logf("+++ Contract owns: %s", bankKeeper.GetAllBalances(ctx, contractAddr).String())
	require.Equal(t, sdk.Coin{Denom: contractsDenom, Amount: sdk.NewInt(10000000)}, bankKeeper.GetBalance(ctx, contractAddr, contractsDenom))

	// now call execute to return a burn message
	_, err = keeper.Execute(ctx, contractAddr, fred, []byte(`burn`), nil)
	require.NoError(t, err)
	t.Logf("+++ Contract owns: %s", bankKeeper.GetAllBalances(ctx, contractAddr).String())
	require.Equal(t, sdk.Coin{Denom: contractsDenom, Amount: sdk.NewInt(1)}, bankKeeper.GetBalance(ctx, contractAddr, contractsDenom))
}

type minterContract struct {
	t            *testing.T
	contractAddr sdk.AccAddress
}

func (m *minterContract) Execute(hash []byte, params cosmwasmv1.Env, msg []byte, store prefix.Store, api cosmwasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.HandleResponse, uint64, error) {
	msgs := map[string]*cosmwasmv2.BankMsg{
		"mint": {
			Mint: &cosmwasmv2.MintMsg{Coin: cosmwasmv1.Coin{Denom: "alx", Amount: "10000000"}},
		},
		"burn": {
			Burn: &cosmwasmv2.BurnMsg{Coin: cosmwasmv1.Coin{Denom: "alx", Amount: "9999999"}},
		},
	}
	return &cosmwasmv2.HandleResponse{
		Messages: []cosmwasmv2.CosmosMsg{
			{Bank: msgs[string(msg)]},
		},
	}, 0, nil
}

func (m minterContract) OnIBCPacketReceive(hash []byte, params cosmwasmv2.Env, packet cosmwasmv2.IBCPacket, store prefix.Store, api cosmwasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketReceiveResponse, uint64, error) {
	panic("implement me")
}

func (m minterContract) OnIBCPacketAcknowledgement(hash []byte, params cosmwasmv2.Env, packetAck cosmwasmv2.IBCAcknowledgement, store prefix.Store, api cosmwasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketAcknowledgementResponse, uint64, error) {
	panic("implement me")
}

func (m minterContract) OnIBCPacketTimeout(hash []byte, params cosmwasmv2.Env, packet cosmwasmv2.IBCPacket, store prefix.Store, api cosmwasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCPacketTimeoutResponse, uint64, error) {
	panic("implement me")
}

func (m minterContract) OnIBCChannelOpen(hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, store prefix.Store, api cosmwasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelOpenResponse, uint64, error) {
	panic("implement me")
}

func (m minterContract) OnIBCChannelConnect(hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, store prefix.Store, api cosmwasm.GoAPI, querier QueryHandler, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelConnectResponse, uint64, error) {
	panic("implement me")
}

func (m minterContract) OnIBCChannelClose(ctx sdk.Context, hash []byte, params cosmwasmv2.Env, channel cosmwasmv2.IBCChannel, meter sdk.GasMeter, gas uint64) (*cosmwasmv2.IBCChannelCloseResponse, uint64, error) {
	panic("implement me")
}
