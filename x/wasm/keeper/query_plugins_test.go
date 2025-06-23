package keeper_test

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sync/atomic"
	"testing"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	abci "github.com/cometbft/cometbft/abci/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/gogoproto/proto"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	sdkmath "cosmossdk.io/math"
	"cosmossdk.io/store"
	storemetrics "cosmossdk.io/store/metrics"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

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
				Aliases:  []string{},
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
				Aliases:  []string{},
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

func TestBankQuerierMetadataWithNilAliases(t *testing.T) {
	metadata := banktypes.Metadata{
		Name: "Test Token",
		Base: "utest",
		DenomUnits: []*banktypes.DenomUnit{
			{
				Denom:    "utest",
				Exponent: 0,
				Aliases:  nil,
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
				Aliases:  []string{},
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
					Aliases:  []string{},
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
						Aliases:  []string{},
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

func TestRawRangeWasmQuerier(t *testing.T) {
	myValidContractAddr := keeper.RandomBech32AccountAddress(t)
	validResponse := wasmvmtypes.RawRangeResponse{
		Data: []wasmvmtypes.RawRangeEntry{
			{
				Key:   []byte("key1"),
				Value: []byte("value"),
			},
		},
		NextKey: nil,
	}
	var ctx sdk.Context
	specs := map[string]struct {
		req    *wasmvmtypes.WasmQuery
		mock   mockWasmQueryKeeper
		expRes wasmvmtypes.RawRangeResponse
		expErr bool
	}{
		"all good": {
			req: &wasmvmtypes.WasmQuery{
				RawRange: &wasmvmtypes.RawRangeQuery{
					ContractAddr: myValidContractAddr,
					Start:        []byte("key0"),
					End:          []byte("key2"),
					Limit:        10,
					Order:        "ascending",
				},
			},
			mock: mockWasmQueryKeeper{
				QueryRawRangeFn: func(ctx context.Context, contractAddress sdk.AccAddress, start, end []byte, limit uint16, reverse bool) (results []wasmvmtypes.RawRangeEntry, nextKey []byte) {
					return validResponse.Data, validResponse.NextKey
				},
			},
			expRes: validResponse,
		},
		"all good - descending": {
			req: &wasmvmtypes.WasmQuery{
				RawRange: &wasmvmtypes.RawRangeQuery{
					ContractAddr: myValidContractAddr,
					Start:        []byte("start"),
					End:          []byte("end"),
					Limit:        10,
					Order:        "descending",
				},
			},
			mock: mockWasmQueryKeeper{
				QueryRawRangeFn: func(ctx context.Context, contractAddress sdk.AccAddress, start, end []byte, limit uint16, reverse bool) (results []wasmvmtypes.RawRangeEntry, nextKey []byte) {
					return []wasmvmtypes.RawRangeEntry{}, nil
				},
			},
			expRes: wasmvmtypes.RawRangeResponse{
				Data:    []wasmvmtypes.RawRangeEntry{},
				NextKey: nil,
			},
		},
		"invalid addr": {
			req: &wasmvmtypes.WasmQuery{
				RawRange: &wasmvmtypes.RawRangeQuery{
					ContractAddr: "not a valid addr",
					Order:        "ascending",
				},
			},
			expErr: true,
		},
		"invalid order": {
			req: &wasmvmtypes.WasmQuery{
				RawRange: &wasmvmtypes.RawRangeQuery{
					ContractAddr: myValidContractAddr,
					Order:        "not a valid order",
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
			var gotRes wasmvmtypes.RawRangeResponse
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
			ctx := sdk.NewContext(ms, cmtproto.Header{}, false, log.NewTestLogger(t)).WithGasMeter(storetypes.NewInfiniteGasMeter())
			q := keeper.NewQueryHandler(ctx, mock, sdk.AccAddress{}, types.NewDefaultWasmGasRegister())
			_, gotErr := q.Query(wasmvmtypes.QueryRequest{}, 1)
			assert.Equal(t, spec.expErr, gotErr)
		})
	}
}

type mockWasmQueryKeeper struct {
	GetContractInfoFn func(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo
	QueryRawFn        func(ctx context.Context, contractAddress sdk.AccAddress, key []byte) []byte
	QuerySmartFn      func(ctx context.Context, contractAddr sdk.AccAddress, req types.RawContractMessage) ([]byte, error)
	QueryRawRangeFn   func(ctx context.Context, contractAddress sdk.AccAddress, start, end []byte, limit uint16, reverse bool) (results []wasmvmtypes.RawRangeEntry, nextKey []byte)
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

func (m mockWasmQueryKeeper) QueryRawRange(ctx context.Context, contractAddress sdk.AccAddress, start, end []byte, limit uint16, reverse bool) (results []wasmvmtypes.RawRangeEntry, nextKey []byte) {
	if m.QueryRawRangeFn == nil {
		panic("not expected to be called")
	}
	return m.QueryRawRangeFn(ctx, contractAddress, start, end, limit, reverse)
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

var _ keeper.GRPCQueryRouter = mockedQueryRouter{}

type mockedQueryRouter struct {
	codec codec.Codec
}

func (m mockedQueryRouter) Route(_ string) baseapp.GRPCQueryHandler {
	return func(ctx sdk.Context, req *abci.RequestQuery) (*abci.ResponseQuery, error) {
		balanceReq := &banktypes.QueryBalanceRequest{}
		if err := m.codec.Unmarshal(req.Data, balanceReq); err != nil {
			return nil, err
		}

		coin := sdk.NewInt64Coin(balanceReq.Denom, 1)
		balanceRes := &banktypes.QueryBalanceResponse{
			Balance: &coin,
		}

		resValue, err := m.codec.Marshal(balanceRes)
		if err != nil {
			return nil, err
		}

		return &abci.ResponseQuery{
			Value: resValue,
		}, nil
	}
}

func TestGRPCQuerier(t *testing.T) {
	const (
		denom1 = "denom1"
		denom2 = "denom2"
	)
	_, keepers := keeper.CreateTestInput(t, false, keeper.AvailableCapabilities)
	cdc := keepers.EncodingConfig.Codec

	acceptedQueries := keeper.AcceptedQueries{
		"/bank.Balance": func() proto.Message { return &banktypes.QueryBalanceResponse{} },
	}

	router := mockedQueryRouter{
		codec: cdc,
	}
	querier := keeper.AcceptListStargateQuerier(acceptedQueries, router, keepers.EncodingConfig.Codec)

	addr := keeper.RandomBech32AccountAddress(t)

	eg := errgroup.Group{}
	errorsCount := atomic.Uint64{}
	for range 50 {
		for _, denom := range []string{denom1, denom2} {
			denom := denom // copy
			eg.Go(func() error {
				queryReq := &banktypes.QueryBalanceRequest{
					Address: addr,
					Denom:   denom,
				}
				grpcData, err := cdc.Marshal(queryReq)
				if err != nil {
					return err
				}

				wasmGrpcReq := &wasmvmtypes.StargateQuery{
					Data: grpcData,
					Path: "/bank.Balance",
				}

				wasmGrpcRes, err := querier(sdk.Context{}, wasmGrpcReq)
				if err != nil {
					return err
				}

				queryRes := &banktypes.QueryBalanceResponse{}
				if err := cdc.UnmarshalJSON(wasmGrpcRes, queryRes); err != nil {
					return err
				}

				expectedCoin := sdk.NewInt64Coin(denom, 1)
				if queryRes.Balance == nil {
					fmt.Printf(
						"Error: expected %s, got nil\n",
						expectedCoin.String(),
					)
					errorsCount.Add(1)
					return nil
				}
				if queryRes.Balance.String() != expectedCoin.String() {
					fmt.Printf(
						"Error: expected %s, got %s\n",
						expectedCoin.String(),
						queryRes.Balance.String(),
					)
					errorsCount.Add(1)
					return nil
				}

				return nil
			})
		}
	}

	require.NoError(t, eg.Wait())
	require.Zero(t, errorsCount.Load())
}
