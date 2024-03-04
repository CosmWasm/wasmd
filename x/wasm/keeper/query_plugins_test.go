package keeper_test

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"
	channeltypes "github.com/cosmos/ibc-go/v8/modules/core/04-channel/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestIBCQuerier(t *testing.T) {
	specs := map[string]struct {
		srcQuery      *wasmvmtypes.IBCQuery
		wasmKeeper    *mockWasmQueryKeeper
		channelKeeper *wasmtesting.MockChannelKeeper
		expJSONResult string
		expErr        *errorsmod.Error
	}{
		"query port id": {
			srcQuery: &wasmvmtypes.IBCQuery{
				PortID: &wasmvmtypes.PortIDQuery{},
			},
			wasmKeeper: &mockWasmQueryKeeper{
				GetContractInfoFn: func(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					return &types.ContractInfo{IBCPortID: "myIBCPortID"}
				},
			},
			channelKeeper: &wasmtesting.MockChannelKeeper{},
			expJSONResult: `{"port_id":"myIBCPortID"}`,
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
			expJSONResult: `{
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
				GetContractInfoFn: func(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
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
			expJSONResult: `{
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
			expJSONResult: "{}",
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
			expJSONResult: "{}",
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
			expJSONResult: "{}",
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			h := keeper.IBCQuerier(spec.wasmKeeper, spec.channelKeeper)
			gotResult, gotErr := h(sdk.Context{}, keeper.RandomAccountAddress(t), spec.srcQuery)
			require.True(t, spec.expErr.Is(gotErr), "exp %v but got %#+v", spec.expErr, gotErr)
			if spec.expErr != nil {
				return
			}
			assert.JSONEq(t, spec.expJSONResult, string(gotResult), string(gotResult))
		})
	}
}

func TestBankQuerierBalance(t *testing.T) {
	mock := bankKeeperMock{GetBalanceFn: func(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
		return sdk.NewCoin(denom, sdkmath.NewInt(1))
	}}

	ctx := sdk.Context{}
	q := keeper.BankQuerier(mock)
	gotBz, gotErr := q(ctx, &wasmvmtypes.BankQuery{
		Balance: &wasmvmtypes.BalanceQuery{
			Address: keeper.RandomBech32AccountAddress(t),
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

func TestBankQuerierMetadata(t *testing.T) {
	metadata := banktypes.Metadata{
		Name: "Test Token",
		Base: "utest",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "utest",
				Exponent: 0,
			},
		},
	}

	mock := bankKeeperMock{GetDenomMetadataFn: func(ctx context.Context, denom string) (banktypes.Metadata, bool) {
		if denom == "utest" {
			return metadata, true
		} else {
			return banktypes.Metadata{}, false
		}
	}}

	ctx := sdk.Context{}
	q := keeper.BankQuerier(mock)
	gotBz, gotErr := q(ctx, &wasmvmtypes.BankQuery{
		DenomMetadata: &wasmvmtypes.DenomMetadataQuery{
			Denom: "utest",
		},
	})
	require.NoError(t, gotErr)
	var got wasmvmtypes.DenomMetadataResponse
	require.NoError(t, json.Unmarshal(gotBz, &got))
	exp := wasmvmtypes.DenomMetadata{
		Name: "Test Token",
		Base: "utest",
		DenomUnits: []wasmvmtypes.DenomUnit{
			{
				Denom:    "utest",
				Exponent: 0,
			},
		},
	}
	assert.Equal(t, exp, got.Metadata)

	_, gotErr2 := q(ctx, &wasmvmtypes.BankQuery{
		DenomMetadata: &wasmvmtypes.DenomMetadataQuery{
			Denom: "uatom",
		},
	})
	require.Error(t, gotErr2)
	assert.Contains(t, gotErr2.Error(), "uatom: not found")
}

func TestBankQuerierAllMetadata(t *testing.T) {
	metadata := []banktypes.Metadata{
		{
			Name: "Test Token",
			Base: "utest",
			DenomUnits: []*banktypes.DenomUnit{
				{
					Denom:    "utest",
					Exponent: 0,
				},
			},
		},
	}

	mock := bankKeeperMock{GetDenomsMetadataFn: func(ctx context.Context, req *banktypes.QueryDenomsMetadataRequest) (*banktypes.QueryDenomsMetadataResponse, error) {
		return &banktypes.QueryDenomsMetadataResponse{
			Metadatas:  metadata,
			Pagination: &query.PageResponse{},
		}, nil
	}}

	ctx := sdk.Context{}
	q := keeper.BankQuerier(mock)
	gotBz, gotErr := q(ctx, &wasmvmtypes.BankQuery{
		AllDenomMetadata: &wasmvmtypes.AllDenomMetadataQuery{},
	})
	require.NoError(t, gotErr)
	var got wasmvmtypes.AllDenomMetadataResponse
	require.NoError(t, json.Unmarshal(gotBz, &got))
	exp := wasmvmtypes.AllDenomMetadataResponse{
		Metadata: []wasmvmtypes.DenomMetadata{
			{
				Name: "Test Token",
				Base: "utest",
				DenomUnits: []wasmvmtypes.DenomUnit{
					{
						Denom:    "utest",
						Exponent: 0,
					},
				},
			},
		},
	}
	assert.Equal(t, exp, got)
}

func TestBankQuerierAllMetadataPagination(t *testing.T) {
	var capturedPagination *query.PageRequest
	mock := bankKeeperMock{GetDenomsMetadataFn: func(ctx context.Context, req *banktypes.QueryDenomsMetadataRequest) (*banktypes.QueryDenomsMetadataResponse, error) {
		capturedPagination = req.Pagination
		return &banktypes.QueryDenomsMetadataResponse{
			Metadatas: []banktypes.Metadata{},
			Pagination: &query.PageResponse{
				NextKey: nil,
			},
		}, nil
	}}

	ctx := sdk.Context{}
	q := keeper.BankQuerier(mock)
	_, gotErr := q(ctx, &wasmvmtypes.BankQuery{
		AllDenomMetadata: &wasmvmtypes.AllDenomMetadataQuery{
			Pagination: &wasmvmtypes.PageRequest{
				Key:   []byte("key"),
				Limit: 10,
			},
		},
	})
	require.NoError(t, gotErr)
	exp := &query.PageRequest{
		Key:   []byte("key"),
		Limit: 10,
	}
	assert.Equal(t, exp, capturedPagination)
}

func TestContractInfoWasmQuerier(t *testing.T) {
	myValidContractAddr := keeper.RandomBech32AccountAddress(t)
	myCreatorAddr := keeper.RandomBech32AccountAddress(t)
	myAdminAddr := keeper.RandomBech32AccountAddress(t)
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
				GetContractInfoFn: func(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					val := types.ContractInfoFixture(func(i *types.ContractInfo) {
						i.Admin, i.Creator, i.IBCPortID = myAdminAddr, myCreatorAddr, "myIBCPort"
					})
					return &val
				},
				IsPinnedCodeFn: func(ctx context.Context, codeID uint64) bool { return true },
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
			mock: mockWasmQueryKeeper{GetContractInfoFn: func(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
				return nil
			}},
			expErr: true,
		},
		"not pinned": {
			req: &wasmvmtypes.WasmQuery{
				ContractInfo: &wasmvmtypes.ContractInfoQuery{ContractAddr: myValidContractAddr},
			},
			mock: mockWasmQueryKeeper{
				GetContractInfoFn: func(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					val := types.ContractInfoFixture(func(i *types.ContractInfo) {
						i.Admin, i.Creator = myAdminAddr, myCreatorAddr
					})
					return &val
				},
				IsPinnedCodeFn: func(ctx context.Context, codeID uint64) bool { return false },
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
				GetContractInfoFn: func(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
					val := types.ContractInfoFixture(func(i *types.ContractInfo) {
						i.Creator = myCreatorAddr
					})
					return &val
				},
				IsPinnedCodeFn: func(ctx context.Context, codeID uint64) bool { return true },
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
			q := keeper.WasmQuerier(spec.mock)
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

func TestCodeInfoWasmQuerier(t *testing.T) {
	myCreatorAddr := keeper.RandomBech32AccountAddress(t)
	var ctx sdk.Context

	myRawChecksum := []byte("myHash78901234567890123456789012")
	specs := map[string]struct {
		req    *wasmvmtypes.WasmQuery
		mock   mockWasmQueryKeeper
		expRes wasmvmtypes.CodeInfoResponse
		expErr bool
	}{
		"all good": {
			req: &wasmvmtypes.WasmQuery{
				CodeInfo: &wasmvmtypes.CodeInfoQuery{CodeID: 1},
			},
			mock: mockWasmQueryKeeper{
				GetCodeInfoFn: func(ctx context.Context, codeID uint64) *types.CodeInfo {
					return &types.CodeInfo{
						CodeHash: myRawChecksum,
						Creator:  myCreatorAddr,
						InstantiateConfig: types.AccessConfig{
							Permission: types.AccessTypeNobody,
							Addresses:  []string{myCreatorAddr},
						},
					}
				},
			},
			expRes: wasmvmtypes.CodeInfoResponse{
				CodeID:   1,
				Creator:  myCreatorAddr,
				Checksum: myRawChecksum,
			},
		},
		"empty code id": {
			req: &wasmvmtypes.WasmQuery{
				CodeInfo: &wasmvmtypes.CodeInfoQuery{},
			},
			expErr: true,
		},
		"unknown code id": {
			req: &wasmvmtypes.WasmQuery{
				CodeInfo: &wasmvmtypes.CodeInfoQuery{CodeID: 1},
			},
			mock: mockWasmQueryKeeper{
				GetCodeInfoFn: func(ctx context.Context, codeID uint64) *types.CodeInfo {
					return nil
				},
			},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			q := keeper.WasmQuerier(spec.mock)
			gotBz, gotErr := q(ctx, spec.req)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			var gotRes wasmvmtypes.CodeInfoResponse
			require.NoError(t, json.Unmarshal(gotBz, &gotRes), string(gotBz))
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
			src:    types.ErrNoSuchContractFn("contract-addr"),
			expErr: wasmvmtypes.NoSuchContract{Addr: "contract-addr"},
		},
		"no such contract - wrapped": {
			src:    errorsmod.Wrap(types.ErrNoSuchContractFn("contract-addr"), "my additional data"),
			expErr: wasmvmtypes.NoSuchContract{Addr: "contract-addr"},
		},
		"no such code": {
			src:    types.ErrNoSuchCodeFn(123),
			expErr: wasmvmtypes.NoSuchCode{CodeID: 123},
		},
		"no such code - wrapped": {
			src:    errorsmod.Wrap(types.ErrNoSuchCodeFn(123), "my additional data"),
			expErr: wasmvmtypes.NoSuchCode{CodeID: 123},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			mock := keeper.WasmVMQueryHandlerFn(func(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error) {
				return nil, spec.src
			})
			ms := store.NewCommitMultiStore(dbm.NewMemDB(), log.NewTestLogger(t), storemetrics.NewNoOpMetrics())
			ctx := sdk.Context{}.WithGasMeter(storetypes.NewInfiniteGasMeter()).WithMultiStore(ms).WithLogger(log.NewTestLogger(t))
			q := keeper.NewQueryHandler(ctx, mock, sdk.AccAddress{}, types.NewDefaultWasmGasRegister())
			_, gotErr := q.Query(wasmvmtypes.QueryRequest{}, 1)
			assert.Equal(t, spec.expErr, gotErr)
		})
	}
}

func TestAcceptListStargateQuerier(t *testing.T) {
	wasmApp := app.SetupWithEmptyStore(t)
	ctx := wasmApp.NewUncachedContext(false, cmtproto.Header{ChainID: "foo", Height: 1, Time: time.Now()})
	err := wasmApp.StakingKeeper.SetParams(ctx, stakingtypes.DefaultParams())
	require.NoError(t, err)

	addrs := app.AddTestAddrsIncremental(wasmApp, ctx, 2, sdkmath.NewInt(1_000_000))
	accepted := keeper.AcceptedStargateQueries{
		"/cosmos.auth.v1beta1.Query/Account": &authtypes.QueryAccountResponse{},
		"/no/route/to/this":                  &authtypes.QueryAccountResponse{},
	}

	marshal := func(pb proto.Message) []byte {
		b, err := proto.Marshal(pb)
		require.NoError(t, err)
		return b
	}

	specs := map[string]struct {
		req     *wasmvmtypes.StargateQuery
		expErr  bool
		expResp string
	}{
		"in accept list - success result": {
			req: &wasmvmtypes.StargateQuery{
				Path: "/cosmos.auth.v1beta1.Query/Account",
				Data: marshal(&authtypes.QueryAccountRequest{Address: addrs[0].String()}),
			},
			expResp: fmt.Sprintf(`{"account":{"@type":"/cosmos.auth.v1beta1.BaseAccount","address":%q,"pub_key":null,"account_number":"1","sequence":"0"}}`, addrs[0].String()),
		},
		"in accept list - error result": {
			req: &wasmvmtypes.StargateQuery{
				Path: "/cosmos.auth.v1beta1.Query/Account",
				Data: marshal(&authtypes.QueryAccountRequest{Address: sdk.AccAddress(ed25519.GenPrivKey().PubKey().Address()).String()}),
			},
			expErr: true,
		},
		"not in accept list": {
			req: &wasmvmtypes.StargateQuery{
				Path: "/cosmos.bank.v1beta1.Query/AllBalances",
				Data: marshal(&banktypes.QueryAllBalancesRequest{Address: addrs[0].String()}),
			},
			expErr: true,
		},
		"unknown route": {
			req: &wasmvmtypes.StargateQuery{
				Path: "/no/route/to/this",
				Data: marshal(&banktypes.QueryAllBalancesRequest{Address: addrs[0].String()}),
			},
			expErr: true,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			q := keeper.AcceptListStargateQuerier(accepted, wasmApp.GRPCQueryRouter(), wasmApp.AppCodec())
			gotBz, gotErr := q(ctx, spec.req)
			if spec.expErr {
				require.Error(t, gotErr)
				return
			}
			require.NoError(t, gotErr)
			assert.JSONEq(t, spec.expResp, string(gotBz), string(gotBz))
		})
	}
}

type mockWasmQueryKeeper struct {
	GetContractInfoFn func(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo
	QueryRawFn        func(ctx context.Context, contractAddress sdk.AccAddress, key []byte) []byte
	QuerySmartFn      func(ctx context.Context, contractAddr sdk.AccAddress, req types.RawContractMessage) ([]byte, error)
	IsPinnedCodeFn    func(ctx context.Context, codeID uint64) bool
	GetCodeInfoFn     func(ctx context.Context, codeID uint64) *types.CodeInfo
}

func (m mockWasmQueryKeeper) GetContractInfo(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo {
	if m.GetContractInfoFn == nil {
		panic("not expected to be called")
	}
	return m.GetContractInfoFn(ctx, contractAddress)
}

func (m mockWasmQueryKeeper) QueryRaw(ctx context.Context, contractAddress sdk.AccAddress, key []byte) []byte {
	if m.QueryRawFn == nil {
		panic("not expected to be called")
	}
	return m.QueryRawFn(ctx, contractAddress, key)
}

func (m mockWasmQueryKeeper) QuerySmart(ctx context.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error) {
	if m.QuerySmartFn == nil {
		panic("not expected to be called")
	}
	return m.QuerySmartFn(ctx, contractAddr, req)
}

func (m mockWasmQueryKeeper) IsPinnedCode(ctx context.Context, codeID uint64) bool {
	if m.IsPinnedCodeFn == nil {
		panic("not expected to be called")
	}
	return m.IsPinnedCodeFn(ctx, codeID)
}

func (m mockWasmQueryKeeper) GetCodeInfo(ctx context.Context, codeID uint64) *types.CodeInfo {
	if m.GetCodeInfoFn == nil {
		panic("not expected to be called")
	}
	return m.GetCodeInfoFn(ctx, codeID)
}

type bankKeeperMock struct {
	GetSupplyFn         func(ctx context.Context, denom string) sdk.Coin
	GetBalanceFn        func(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalancesFn    func(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetDenomMetadataFn  func(ctx context.Context, denom string) (banktypes.Metadata, bool)
	GetDenomsMetadataFn func(ctx context.Context, req *banktypes.QueryDenomsMetadataRequest) (*banktypes.QueryDenomsMetadataResponse, error)
}

func (m bankKeeperMock) GetSupply(ctx context.Context, denom string) sdk.Coin {
	if m.GetSupplyFn == nil {
		panic("not expected to be called")
	}
	return m.GetSupplyFn(ctx, denom)
}

func (m bankKeeperMock) GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin {
	if m.GetBalanceFn == nil {
		panic("not expected to be called")
	}
	return m.GetBalanceFn(ctx, addr, denom)
}

func (m bankKeeperMock) GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins {
	if m.GetAllBalancesFn == nil {
		panic("not expected to be called")
	}
	return m.GetAllBalancesFn(ctx, addr)
}

func (m bankKeeperMock) GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool) {
	if m.GetDenomMetadataFn == nil {
		panic("not expected to be called")
	}
	return m.GetDenomMetadataFn(ctx, denom)
}

func (m bankKeeperMock) DenomsMetadata(ctx context.Context, req *banktypes.QueryDenomsMetadataRequest) (*banktypes.QueryDenomsMetadataResponse, error) {
	if m.GetDenomsMetadataFn == nil {
		panic("not expected to be called")
	}
	return m.GetDenomsMetadataFn(ctx, req)
}

func TestConvertProtoToJSONMarshal(t *testing.T) {
	testCases := []struct {
		name                  string
		queryPath             string
		protoResponseStruct   proto.Message
		originalResponse      string
		expectedProtoResponse proto.Message
		expectedError         bool
	}{
		{
			name:                "successful conversion from proto response to json marshaled response",
			queryPath:           "/cosmos.bank.v1beta1.Query/AllBalances",
			originalResponse:    "0a090a036261721202333012050a03666f6f",
			protoResponseStruct: &banktypes.QueryAllBalancesResponse{},
			expectedProtoResponse: &banktypes.QueryAllBalancesResponse{
				Balances: sdk.NewCoins(sdk.NewCoin("bar", sdkmath.NewInt(30))),
				Pagination: &query.PageResponse{
					NextKey: []byte("foo"),
				},
			},
		},
		{
			name:                "invalid proto response struct",
			queryPath:           "/cosmos.bank.v1beta1.Query/AllBalances",
			originalResponse:    "0a090a036261721202333012050a03666f6f",
			protoResponseStruct: &authtypes.QueryAccountResponse{},
			expectedError:       true,
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			originalVersionBz, err := hex.DecodeString(tc.originalResponse)
			require.NoError(t, err)
			appCodec := app.MakeEncodingConfig(t).Codec

			jsonMarshalledResponse, err := keeper.ConvertProtoToJSONMarshal(appCodec, tc.protoResponseStruct, originalVersionBz)
			if tc.expectedError {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)

			// check response by json marshaling proto response into json response manually
			jsonMarshalExpectedResponse, err := appCodec.MarshalJSON(tc.expectedProtoResponse)
			require.NoError(t, err)
			require.JSONEq(t, string(jsonMarshalledResponse), string(jsonMarshalExpectedResponse))
		})
	}
}

func TestConvertSDKDecCoinToWasmDecCoin(t *testing.T) {
	specs := map[string]struct {
		src sdk.DecCoins
		exp []wasmvmtypes.DecCoin
	}{
		"one coin": {
			src: sdk.NewDecCoins(sdk.NewInt64DecCoin("alx", 1)),
			exp: []wasmvmtypes.DecCoin{{Amount: "1.000000000000000000", Denom: "alx"}},
		},
		"multiple coins": {
			src: sdk.NewDecCoins(sdk.NewInt64DecCoin("alx", 1), sdk.NewInt64DecCoin("blx", 2)),
			exp: []wasmvmtypes.DecCoin{{Amount: "1.000000000000000000", Denom: "alx"}, {Amount: "2.000000000000000000", Denom: "blx"}},
		},
		"small amount": {
			src: sdk.NewDecCoins(sdk.NewDecCoinFromDec("alx", sdkmath.LegacyNewDecWithPrec(1, 18))),
			exp: []wasmvmtypes.DecCoin{{Amount: "0.000000000000000001", Denom: "alx"}},
		},
		"big amount": {
			src: sdk.NewDecCoins(sdk.NewDecCoin("alx", sdkmath.NewIntFromUint64(math.MaxUint64))),
			exp: []wasmvmtypes.DecCoin{{Amount: "18446744073709551615.000000000000000000", Denom: "alx"}},
		},
		"empty": {
			src: sdk.NewDecCoins(),
			exp: []wasmvmtypes.DecCoin{},
		},
		"nil": {
			exp: []wasmvmtypes.DecCoin{},
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			got := keeper.ConvertSDKDecCoinsToWasmDecCoins(spec.src)
			assert.Equal(t, spec.exp, got)
		})
	}
}

func TestResetProtoMarshalerAfterJsonMarshal(t *testing.T) {
	appCodec := app.MakeEncodingConfig(t).Codec

	protoMarshaler := &banktypes.QueryAllBalancesResponse{}
	expected := appCodec.MustMarshalJSON(&banktypes.QueryAllBalancesResponse{
		Balances: sdk.NewCoins(sdk.NewCoin("bar", sdkmath.NewInt(30))),
		Pagination: &query.PageResponse{
			NextKey: []byte("foo"),
		},
	})

	bz, err := hex.DecodeString("0a090a036261721202333012050a03666f6f")
	require.NoError(t, err)

	// first marshal
	response, err := keeper.ConvertProtoToJSONMarshal(appCodec, protoMarshaler, bz)
	require.NoError(t, err)
	require.Equal(t, expected, response)

	// second marshal
	response, err = keeper.ConvertProtoToJSONMarshal(appCodec, protoMarshaler, bz)
	require.NoError(t, err)
	require.Equal(t, expected, response)
}

// TestDeterministicJsonMarshal tests that we get deterministic JSON marshaled response upon
// proto struct update in the state machine.
func TestDeterministicJsonMarshal(t *testing.T) {
	testCases := []struct {
		name                string
		originalResponse    string
		updatedResponse     string
		queryPath           string
		responseProtoStruct proto.Message
		expectedProto       func() proto.Message
	}{
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
		{
			"Query Account",
			"0a530a202f636f736d6f732e617574682e763162657461312e426173654163636f756e74122f0a2d636f736d6f733166387578756c746e3873717a687a6e72737a3371373778776171756867727367366a79766679",
			"0a530a202f636f736d6f732e617574682e763162657461312e426173654163636f756e74122f0a2d636f736d6f733166387578756c746e3873717a687a6e72737a3371373778776171756867727367366a79766679122d636f736d6f733166387578756c746e3873717a687a6e72737a3371373778776171756867727367366a79766679",
			"/cosmos.auth.v1beta1.Query/Account",
			&authtypes.QueryAccountResponse{},
			func() proto.Message {
				account := authtypes.BaseAccount{
					Address: "cosmos1f8uxultn8sqzhznrsz3q77xwaquhgrsg6jyvfy",
				}
				accountResponse, err := codectypes.NewAnyWithValue(&account)
				require.NoError(t, err)
				return &authtypes.QueryAccountResponse{
					Account: accountResponse,
				}
			},
		},
	}

	for _, tc := range testCases {
		t.Run(fmt.Sprintf("Case %s", tc.name), func(t *testing.T) {
			appCodec := app.MakeEncodingConfig(t).Codec

			originVersionBz, err := hex.DecodeString(tc.originalResponse)
			require.NoError(t, err)
			jsonMarshalledOriginalBz, err := keeper.ConvertProtoToJSONMarshal(appCodec, tc.responseProtoStruct, originVersionBz)
			require.NoError(t, err)

			newVersionBz, err := hex.DecodeString(tc.updatedResponse)
			require.NoError(t, err)
			jsonMarshalledUpdatedBz, err := keeper.ConvertProtoToJSONMarshal(appCodec, tc.responseProtoStruct, newVersionBz)
			require.NoError(t, err)

			// json marshaled bytes should be the same since we use the same proto struct for unmarshalling
			require.Equal(t, jsonMarshalledOriginalBz, jsonMarshalledUpdatedBz)

			// raw build also make same result
			jsonMarshalExpectedResponse, err := appCodec.MarshalJSON(tc.expectedProto())
			require.NoError(t, err)
			require.Equal(t, jsonMarshalledUpdatedBz, jsonMarshalExpectedResponse)
		})
	}
}
