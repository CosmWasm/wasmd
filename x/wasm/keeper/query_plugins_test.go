package keeper

import (
	"encoding/hex"
	"encoding/json"
	"testing"

	"github.com/cosmos/cosmos-sdk/store"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/golang/protobuf/proto"
	dbm "github.com/tendermint/tm-db"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	channeltypes "github.com/cosmos/ibc-go/v3/modules/core/04-channel/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestIBCQuerier(t *testing.T) {
	myExampleChannels := []channeltypes.IdentifiedChannel{
		// this is returned
		{
			State:    channeltypes.OPEN,
			Ordering: channeltypes.ORDERED,
			Counterparty: channeltypes.Counterparty{
				PortId:    "counterPartyPortID",
				ChannelId: "counterPartyChannelID",
			},
			ConnectionHops: []string{"one"},
			Version:        "v1",
			PortId:         "myPortID",
			ChannelId:      "myChannelID",
		},
		// this is filtered out
		{
			State:    channeltypes.INIT,
			Ordering: channeltypes.UNORDERED,
			Counterparty: channeltypes.Counterparty{
				PortId: "foobar",
			},
			ConnectionHops: []string{"one"},
			Version:        "initversion",
			PortId:         "initPortID",
			ChannelId:      "initChannelID",
		},
		// this is returned
		{
			State:    channeltypes.OPEN,
			Ordering: channeltypes.UNORDERED,
			Counterparty: channeltypes.Counterparty{
				PortId:    "otherCounterPartyPortID",
				ChannelId: "otherCounterPartyChannelID",
			},
			ConnectionHops: []string{"other", "second"},
			Version:        "otherVersion",
			PortId:         "otherPortID",
			ChannelId:      "otherChannelID",
		},
		// this is filtered out
		{
			State:    channeltypes.CLOSED,
			Ordering: channeltypes.ORDERED,
			Counterparty: channeltypes.Counterparty{
				PortId:    "super",
				ChannelId: "duper",
			},
			ConnectionHops: []string{"no-more"},
			Version:        "closedVersion",
			PortId:         "closedPortID",
			ChannelId:      "closedChannelID",
		},
	}
	specs := map[string]struct {
		srcQuery      *wasmvmtypes.IBCQuery
		wasmKeeper    *mockWasmQueryKeeper
		channelKeeper *wasmtesting.MockChannelKeeper
		expJsonResult string
		expErr        *sdkerrors.Error
	}{
		"query port id": {
			srcQuery: &wasmvmtypes.IBCQuery{
				PortID: &wasmvmtypes.PortIDQuery{},
			},
			wasmKeeper: &mockWasmQueryKeeper{
				GetContractInfoFn: func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					return &types.ContractInfo{IBCPortID: "myIBCPortID"}
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{},
			expJsonResult: `{"port_id":"myIBCPortID"}`,
		},
		"query list channels - all": {
			srcQuery: &wasmvmtypes.IBCQuery{
				ListChannels: &wasmvmtypes.ListChannelsQuery{},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				IterateChannelsFn: wasmtesting.MockChannelKeeperIterator(myExampleChannels),
			},
			expJsonResult: `{
  "channels": [
    {
      "endpoint": {
        "port_id": "myPortID",
        "channel_id": "myChannelID"
      },
      "counterparty_endpoint": {
        "port_id": "counterPartyPortID",
        "channel_id": "counterPartyChannelID"
      },
      "order": "ORDER_ORDERED",
      "version": "v1",
      "connection_id": "one"
    },
    {
      "endpoint": {
        "port_id": "otherPortID",
        "channel_id": "otherChannelID"
      },
      "counterparty_endpoint": {
        "port_id": "otherCounterPartyPortID",
        "channel_id": "otherCounterPartyChannelID"
      },
      "order": "ORDER_UNORDERED",
      "version": "otherVersion",
      "connection_id": "other"
    }
  ]
}`,
		},
		"query list channels - filtered": {
			srcQuery: &wasmvmtypes.IBCQuery{
				ListChannels: &wasmvmtypes.ListChannelsQuery{
					PortID: "otherPortID",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				IterateChannelsFn: wasmtesting.MockChannelKeeperIterator(myExampleChannels),
			},
			expJsonResult: `{
  "channels": [
    {
      "endpoint": {
        "port_id": "otherPortID",
        "channel_id": "otherChannelID"
      },
      "counterparty_endpoint": {
        "port_id": "otherCounterPartyPortID",
        "channel_id": "otherCounterPartyChannelID"
      },
      "order": "ORDER_UNORDERED",
      "version": "otherVersion",
      "connection_id": "other"
    }
  ]
}`,
		},
		"query list channels - filtered empty": {
			srcQuery: &wasmvmtypes.IBCQuery{
				ListChannels: &wasmvmtypes.ListChannelsQuery{
					PortID: "none-existing",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				IterateChannelsFn: wasmtesting.MockChannelKeeperIterator(myExampleChannels),
			},
			expJsonResult: `{"channels": []}`,
		},
		"query channel": {
			srcQuery: &wasmvmtypes.IBCQuery{
				Channel: &wasmvmtypes.ChannelQuery{
					PortID:    "myQueryPortID",
					ChannelID: "myQueryChannelID",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
					return channeltypes.Channel{
						State:    channeltypes.OPEN,
						Ordering: channeltypes.UNORDERED,
						Counterparty: channeltypes.Counterparty{
							PortId:    "counterPartyPortID",
							ChannelId: "otherCounterPartyChannelID",
						},
						ConnectionHops: []string{"one"},
						Version:        "version",
					}, true
				},
			},
			expJsonResult: `{
  "channel": {
    "endpoint": {
      "port_id": "myQueryPortID",
      "channel_id": "myQueryChannelID"
    },
    "counterparty_endpoint": {
      "port_id": "counterPartyPortID",
      "channel_id": "otherCounterPartyChannelID"
    },
    "order": "ORDER_UNORDERED",
    "version": "version",
    "connection_id": "one"
  }
}`,
		},
		"query channel - without port set": {
			srcQuery: &wasmvmtypes.IBCQuery{
				Channel: &wasmvmtypes.ChannelQuery{
					ChannelID: "myQueryChannelID",
				},
			},
			wasmKeeper: &mockWasmQueryKeeper{
				GetContractInfoFn: func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					return &types.ContractInfo{IBCPortID: "myLoadedPortID"}
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
					return channeltypes.Channel{
						State:    channeltypes.OPEN,
						Ordering: channeltypes.UNORDERED,
						Counterparty: channeltypes.Counterparty{
							PortId:    "counterPartyPortID",
							ChannelId: "otherCounterPartyChannelID",
						},
						ConnectionHops: []string{"one"},
						Version:        "version",
					}, true
				},
			},
			expJsonResult: `{
  "channel": {
    "endpoint": {
      "port_id": "myLoadedPortID",
      "channel_id": "myQueryChannelID"
    },
    "counterparty_endpoint": {
      "port_id": "counterPartyPortID",
      "channel_id": "otherCounterPartyChannelID"
    },
    "order": "ORDER_UNORDERED",
    "version": "version",
    "connection_id": "one"
  }
}`,
		},
		"query channel in init state": {
			srcQuery: &wasmvmtypes.IBCQuery{
				Channel: &wasmvmtypes.ChannelQuery{
					PortID:    "myQueryPortID",
					ChannelID: "myQueryChannelID",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
					return channeltypes.Channel{
						State:    channeltypes.INIT,
						Ordering: channeltypes.UNORDERED,
						Counterparty: channeltypes.Counterparty{
							PortId: "foobar",
						},
						ConnectionHops: []string{"one"},
						Version:        "initversion",
					}, true
				},
			},
			expJsonResult: "{}",
		},
		"query channel in closed state": {
			srcQuery: &wasmvmtypes.IBCQuery{
				Channel: &wasmvmtypes.ChannelQuery{
					PortID:    "myQueryPortID",
					ChannelID: "myQueryChannelID",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
					return channeltypes.Channel{
						State:    channeltypes.CLOSED,
						Ordering: channeltypes.ORDERED,
						Counterparty: channeltypes.Counterparty{
							PortId:    "super",
							ChannelId: "duper",
						},
						ConnectionHops: []string{"no-more"},
						Version:        "closedVersion",
					}, true
				},
			},
			expJsonResult: "{}",
		},
		"query channel - empty result": {
			srcQuery: &wasmvmtypes.IBCQuery{
				Channel: &wasmvmtypes.ChannelQuery{
					PortID:    "myQueryPortID",
					ChannelID: "myQueryChannelID",
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{
				GetChannelFn: func(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool) {
					return channeltypes.Channel{}, false
				},
			},
			expJsonResult: "{}",
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			h := IBCQuerier(spec.wasmKeeper, spec.channelKeeper)
			gotResult, gotErr := h(sdk.Context{}, RandomAccountAddress(t), spec.srcQuery)
			require.True(t, spec.expErr.Is(gotErr), "exp %v but got %#+v", spec.expErr, gotErr)
			if spec.expErr != nil {
				return
			}
			assert.JSONEq(t, spec.expJsonResult, string(gotResult), string(gotResult))
		})
	}
}

func TestBankQuerierBalance(t *testing.T) {
	mock := bankKeeperMock{GetBalanceFn: func(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
		return sdk.NewCoin(denom, sdk.NewInt(1))
	}}

	ctx := sdk.Context{}
	q := BankQuerier(mock)
	gotBz, gotErr := q(ctx, &wasmvmtypes.BankQuery{
		Balance: &wasmvmtypes.BalanceQuery{
			Address: RandomBech32AccountAddress(t),
			Denom:   "ALX",
		},
	})
	require.NoError(t, gotErr)
	var got wasmvmtypes.BalanceResponse
	require.NoError(t, json.Unmarshal(gotBz, &got))
	exp := wasmvmtypes.BalanceResponse{
		Amount: wasmvmtypes.Coin{
			Denom:  "ALX",
			Amount: "1",
		},
	}
	assert.Equal(t, exp, got)
}

func TestContractInfoWasmQuerier(t *testing.T) {
	myValidContractAddr := RandomBech32AccountAddress(t)
	myCreatorAddr := RandomBech32AccountAddress(t)
	myAdminAddr := RandomBech32AccountAddress(t)
	var ctx sdk.Context

	specs := map[string]struct {
		req    *wasmvmtypes.WasmQuery
		mock   mockWasmQueryKeeper
		expRes wasmvmtypes.ContractInfoResponse
		expErr bool
	}{
		"all good": {
			req: &wasmvmtypes.WasmQuery{
				ContractInfo: &wasmvmtypes.ContractInfoQuery{ContractAddr: myValidContractAddr},
			},
			mock: mockWasmQueryKeeper{
				GetContractInfoFn: func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					val := types.ContractInfoFixture(func(i *types.ContractInfo) {
						i.Admin, i.Creator, i.IBCPortID = myAdminAddr, myCreatorAddr, "myIBCPort"
					})
					return &val
				},
				IsPinnedCodeFn: func(ctx sdk.Context, codeID uint64) bool { return true },
			},
			expRes: wasmvmtypes.ContractInfoResponse{
				CodeID:  1,
				Creator: myCreatorAddr,
				Admin:   myAdminAddr,
				Pinned:  true,
				IBCPort: "myIBCPort",
			},
		},
		"invalid addr": {
			req: &wasmvmtypes.WasmQuery{
				ContractInfo: &wasmvmtypes.ContractInfoQuery{ContractAddr: "not a valid addr"},
			},
			expErr: true,
		},
		"unknown addr": {
			req: &wasmvmtypes.WasmQuery{
				ContractInfo: &wasmvmtypes.ContractInfoQuery{ContractAddr: myValidContractAddr},
			},
			mock: mockWasmQueryKeeper{GetContractInfoFn: func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
				return nil
			}},
			expErr: true,
		},
		"not pinned": {
			req: &wasmvmtypes.WasmQuery{
				ContractInfo: &wasmvmtypes.ContractInfoQuery{ContractAddr: myValidContractAddr},
			},
			mock: mockWasmQueryKeeper{
				GetContractInfoFn: func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					val := types.ContractInfoFixture(func(i *types.ContractInfo) {
						i.Admin, i.Creator = myAdminAddr, myCreatorAddr
					})
					return &val
				},
				IsPinnedCodeFn: func(ctx sdk.Context, codeID uint64) bool { return false },
			},
			expRes: wasmvmtypes.ContractInfoResponse{
				CodeID:  1,
				Creator: myCreatorAddr,
				Admin:   myAdminAddr,
				Pinned:  false,
			},
		},
		"without admin": {
			req: &wasmvmtypes.WasmQuery{
				ContractInfo: &wasmvmtypes.ContractInfoQuery{ContractAddr: myValidContractAddr},
			},
			mock: mockWasmQueryKeeper{
				GetContractInfoFn: func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					val := types.ContractInfoFixture(func(i *types.ContractInfo) {
						i.Creator = myCreatorAddr
					})
					return &val
				},
				IsPinnedCodeFn: func(ctx sdk.Context, codeID uint64) bool { return true },
			},
			expRes: wasmvmtypes.ContractInfoResponse{
				CodeID:  1,
				Creator: myCreatorAddr,
				Pinned:  true,
			},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			q := WasmQuerier(spec.mock)
			gotBz, gotErr := q(ctx, spec.req)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			var gotRes wasmvmtypes.ContractInfoResponse
			require.NoError(t, json.Unmarshal(gotBz, &gotRes))
			assert.Equal(t, spec.expRes, gotRes)
		})
	}
}

func TestQueryErrors(t *testing.T) {
	specs := map[string]struct {
		src    error
		expErr error
	}{
		"no error": {},
		"no such contract": {
			src:    &types.ErrNoSuchContract{Addr: "contract-addr"},
			expErr: wasmvmtypes.NoSuchContract{Addr: "contract-addr"},
		},
		"no such contract - wrapped": {
			src:    sdkerrors.Wrap(&types.ErrNoSuchContract{Addr: "contract-addr"}, "my additional data"),
			expErr: wasmvmtypes.NoSuchContract{Addr: "contract-addr"},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			mock := WasmVMQueryHandlerFn(func(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error) {
				return nil, spec.src
			})
			ctx := sdk.Context{}.WithGasMeter(sdk.NewInfiniteGasMeter()).WithMultiStore(store.NewCommitMultiStore(dbm.NewMemDB()))
			q := NewQueryHandler(ctx, mock, sdk.AccAddress{}, NewDefaultWasmGasRegister())
			_, gotErr := q.Query(wasmvmtypes.QueryRequest{}, 1)
			assert.Equal(t, spec.expErr, gotErr)
		})
	}
}

type mockWasmQueryKeeper struct {
	GetContractInfoFn func(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo
	QueryRawFn        func(ctx sdk.Context, contractAddress sdk.AccAddress, key []byte) []byte
	QuerySmartFn      func(ctx sdk.Context, contractAddr sdk.AccAddress, req types.RawContractMessage) ([]byte, error)
	IsPinnedCodeFn    func(ctx sdk.Context, codeID uint64) bool
}

func (m mockWasmQueryKeeper) GetContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
	if m.GetContractInfoFn == nil {
		panic("not expected to be called")
	}
	return m.GetContractInfoFn(ctx, contractAddress)
}

func (m mockWasmQueryKeeper) QueryRaw(ctx sdk.Context, contractAddress sdk.AccAddress, key []byte) []byte {
	if m.QueryRawFn == nil {
		panic("not expected to be called")
	}
	return m.QueryRawFn(ctx, contractAddress, key)
}

func (m mockWasmQueryKeeper) QuerySmart(ctx sdk.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error) {
	if m.QuerySmartFn == nil {
		panic("not expected to be called")
	}
	return m.QuerySmartFn(ctx, contractAddr, req)
}

func (m mockWasmQueryKeeper) IsPinnedCode(ctx sdk.Context, codeID uint64) bool {
	if m.IsPinnedCodeFn == nil {
		panic("not expected to be called")
	}
	return m.IsPinnedCodeFn(ctx, codeID)
}

type bankKeeperMock struct {
	GetSupplyFn      func(ctx sdk.Context, denom string) sdk.Coin
	GetBalanceFn     func(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalancesFn func(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins
}

func (m bankKeeperMock) GetSupply(ctx sdk.Context, denom string) sdk.Coin {
	if m.GetSupplyFn == nil {
		panic("not expected to be called")
	}
	return m.GetSupplyFn(ctx, denom)
}

func (m bankKeeperMock) GetBalance(ctx sdk.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	if m.GetBalanceFn == nil {
		panic("not expected to be called")
	}
	return m.GetBalanceFn(ctx, addr, denom)
}

func (m bankKeeperMock) GetAllBalances(ctx sdk.Context, addr sdk.AccAddress) sdk.Coins {
	if m.GetAllBalancesFn == nil {
		panic("not expected to be called")
	}
	return m.GetAllBalancesFn(ctx, addr)
}

/**
 * Origin Response
 * balances:<denom:"bar" amount:"30" > pagination:<next_key:"foo" >
 * "0a090a036261721202333012050a03666f6f"
 *
 * New Version Response
 * The binary built from the proto response with additional field address
 * balances:<denom:"bar" amount:"30" > pagination:<next_key:"foo" > address:"cosmos1j6j5tsquq2jlw2af7l3xekyaq7zg4l8jsufu78"
 * "0a090a036261721202333012050a03666f6f1a2d636f736d6f73316a366a357473717571326a6c77326166376c3378656b796171377a67346c386a737566753738"
// Updated proto
message QueryAllBalancesResponse {
	// balances is the balances of all the coins.
	repeated cosmos.base.v1beta1.Coin balances = 1
	[(gogoproto.nullable) = false, (gogoproto.castrepeated) = "github.com/cosmos/cosmos-sdk/types.Coins"];
	// pagination defines the pagination in the response.
	cosmos.base.query.v1beta1.PageResponse pagination = 2;
	// address is the address to query all balances for.
	string address = 3;
}
// Origin proto
message QueryAllBalancesResponse {
	// balances is the balances of all the coins.
	repeated cosmos.base.v1beta1.Coin balances = 1
	[(gogoproto.nullable) = false, (gogoproto.castrepeated) = "github.com/cosmos/cosmos-sdk/types.Coins"];
	// pagination defines the pagination in the response.
	cosmos.base.query.v1beta1.PageResponse pagination = 2;
}
*/

func TestDeterministic_AllBalances(t *testing.T) {
	originVersionBz, err := hex.DecodeString("0a090a036261721202333012050a03666f6f")
	require.NoError(t, err)

	newVersionBz, err := hex.DecodeString("0a090a036261721202333012050a03666f6f1a2d636f736d6f73316a366a357473717571326a6c77326166376c3378656b796171377a67346c386a737566753738")
	require.NoError(t, err)

	StargateWhitelist.Store("/cosmos.bank.v1beta1.Query/AllBalances", &banktypes.QueryAllBalancesResponse{})
	binding, ok := StargateWhitelist.Load("/cosmos.bank.v1beta1.Query/AllBalances")
	require.True(t, ok)

	// new version response should be changed into origin version response
	normalizedBz, err := NormalizeReponse(binding, newVersionBz)
	require.NoError(t, err)

	require.Equal(t, originVersionBz, normalizedBz)
	require.NotEqual(t, newVersionBz, normalizedBz)

	// raw build also make same result
	expectedResponse := banktypes.QueryAllBalancesResponse{
		Balances: sdk.NewCoins(sdk.NewCoin("bar", sdk.NewInt(30))),
		Pagination: &query.PageResponse{
			NextKey: []byte("foo"),
		},
	}
	expectedResponseBz, err := proto.Marshal(&expectedResponse)
	require.NoError(t, err)
	require.Equal(t, expectedResponseBz, normalizedBz)

	// should be cleared
	data := binding.(*banktypes.QueryAllBalancesResponse)
	require.Empty(t, data.Balances)
	require.Empty(t, data.Pagination)
}

/**
 *
 * Origin Response
 * 0a530a202f636f736d6f732e617574682e763162657461312e426173654163636f756e74122f0a2d636f736d6f7331346c3268686a6e676c3939367772703935673867646a6871653038326375367a7732706c686b
 *
 * Updated Response
 * 0a530a202f636f736d6f732e617574682e763162657461312e426173654163636f756e74122f0a2d636f736d6f7331646a783375676866736d6b6135386676673076616a6e6533766c72776b7a6a346e6377747271122d636f736d6f7331646a783375676866736d6b6135386676673076616a6e6533766c72776b7a6a346e6377747271
// Origin proto
message QueryAccountResponse {
  // account defines the account of the corresponding address.
  google.protobuf.Any account = 1 [(cosmos_proto.accepts_interface) = "AccountI"];
}
// Updated proto
message QueryAccountResponse {
  // account defines the account of the corresponding address.
  google.protobuf.Any account = 1 [(cosmos_proto.accepts_interface) = "AccountI"];
  // address is the address to query for.
	string address = 2;
}
*/

func TestDeterministic_Account(t *testing.T) {
	originVersionBz, err := hex.DecodeString("0a530a202f636f736d6f732e617574682e763162657461312e426173654163636f756e74122f0a2d636f736d6f733166387578756c746e3873717a687a6e72737a3371373778776171756867727367366a79766679")
	require.NoError(t, err)

	newVersionBz, err := hex.DecodeString("0a530a202f636f736d6f732e617574682e763162657461312e426173654163636f756e74122f0a2d636f736d6f733166387578756c746e3873717a687a6e72737a3371373778776171756867727367366a79766679122d636f736d6f733166387578756c746e3873717a687a6e72737a3371373778776171756867727367366a79766679")
	require.NoError(t, err)

	StargateWhitelist.Store("/cosmos.auth.v1beta1.Query/Account", &authtypes.QueryAccountResponse{})

	binding, ok := StargateWhitelist.Load("/cosmos.auth.v1beta1.Query/Account")
	require.True(t, ok)

	// new version response should be changed into origin version response
	normalizedBz, err := NormalizeReponse(binding, newVersionBz)
	require.NoError(t, err)

	require.Equal(t, originVersionBz, normalizedBz)
	require.NotEqual(t, newVersionBz, normalizedBz)

	// should be cleared
	data := binding.(*authtypes.QueryAccountResponse)
	require.Empty(t, data.Account)
}
