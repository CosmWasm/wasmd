package keeper

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	wasmvmtypes "github.com/CosmWasm/wasmvm/v3/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/gogoproto/proto"
	channeltypes "github.com/cosmos/ibc-go/v10/modules/core/04-channel/types"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"github.com/cosmos/cosmos-sdk/types/query"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	distributiontypes "github.com/cosmos/cosmos-sdk/x/distribution/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

type QueryHandler struct {
	Ctx         sdk.Context
	Plugins     WasmVMQueryHandler
	Caller      sdk.AccAddress
	gasRegister types.GasRegister
}

func NewQueryHandler(ctx sdk.Context, vmQueryHandler WasmVMQueryHandler, caller sdk.AccAddress, gasRegister types.GasRegister) QueryHandler {
	return QueryHandler{
		Ctx:         ctx,
		Plugins:     vmQueryHandler,
		Caller:      caller,
		gasRegister: gasRegister,
	}
}

type GRPCQueryRouter interface {
	Route(path string) baseapp.GRPCQueryHandler
}

// -- end baseapp interfaces --

var _ wasmvmtypes.Querier = QueryHandler{}

func (q QueryHandler) Query(request wasmvmtypes.QueryRequest, gasLimit uint64) ([]byte, error) {
	// set a limit for a subCtx
	sdkGas := q.gasRegister.FromWasmVMGas(gasLimit)
	// discard all changes/ events in subCtx by not committing the cached context
	subCtx, _ := q.Ctx.WithGasMeter(storetypes.NewGasMeter(sdkGas)).CacheContext()

	// make sure we charge the higher level context even on panic
	defer func() {
		q.Ctx.GasMeter().ConsumeGas(subCtx.GasMeter().GasConsumed(), "contract sub-query")
	}()

	res, err := q.Plugins.HandleQuery(subCtx, q.Caller, request)
	if err == nil {
		// short-circuit, the rest is dealing with handling existing errors
		return res, nil
	}

	// special mappings to wasmvm system error (which are not redacted)
	var wasmvmErr types.WasmVMErrorable
	if ok := errors.As(err, &wasmvmErr); ok {
		err = wasmvmErr.ToWasmVMError()
	}

	// Issue #759 - we don't return error string for worries of non-determinism
	moduleLogger(q.Ctx).Debug("Redacting submessage error", "cause", err)
	return nil, redactError(err)
}

func (q QueryHandler) GasConsumed() uint64 {
	return q.gasRegister.ToWasmVMGas(q.Ctx.GasMeter().GasConsumed())
}

type CustomQuerier func(ctx sdk.Context, request json.RawMessage) ([]byte, error)

type (
	stargateQuerierFn func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error)
	grpcQuerierFn     func(ctx sdk.Context, request *wasmvmtypes.GrpcQuery) (proto.Message, error)
)

type QueryPlugins struct {
	Bank         func(ctx sdk.Context, request *wasmvmtypes.BankQuery) ([]byte, error)
	Custom       CustomQuerier
	IBC          func(ctx sdk.Context, caller sdk.AccAddress, request *wasmvmtypes.IBCQuery) ([]byte, error)
	Staking      func(ctx sdk.Context, request *wasmvmtypes.StakingQuery) ([]byte, error)
	Stargate     stargateQuerierFn
	Grpc         grpcQuerierFn
	Wasm         func(ctx sdk.Context, request *wasmvmtypes.WasmQuery) ([]byte, error)
	Distribution func(ctx sdk.Context, request *wasmvmtypes.DistributionQuery) ([]byte, error)
}

type contractMetaDataSource interface {
	GetContractInfo(ctx context.Context, contractAddress sdk.AccAddress) *types.ContractInfo
}

type wasmQueryKeeper interface {
	contractMetaDataSource
	GetCodeInfo(ctx context.Context, codeID uint64) *types.CodeInfo
	QueryRaw(ctx context.Context, contractAddress sdk.AccAddress, key []byte) []byte
	QueryRawRange(ctx context.Context, contractAddress sdk.AccAddress, start, end []byte, limit uint16, reverse bool) (results []wasmvmtypes.RawRangeEntry, nextKey []byte)
	QuerySmart(ctx context.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error)
	IsPinnedCode(ctx context.Context, codeID uint64) bool
}

func DefaultQueryPlugins(
	bank types.BankViewKeeper,
	staking types.StakingKeeper,
	distKeeper types.DistributionKeeper,
	channelKeeper types.ChannelKeeper,
	wasm wasmQueryKeeper,
) QueryPlugins {
	// By default, we reject all stargate and gRPC queries.
	// The chain needs to provide a querier plugin that only allows deterministic queries.
	return QueryPlugins{
		Bank:         BankQuerier(bank),
		Custom:       NoCustomQuerier,
		IBC:          IBCQuerier(wasm, channelKeeper),
		Staking:      StakingQuerier(staking, distKeeper),
		Stargate:     RejectStargateQuerier,
		Grpc:         RejectGrpcQuerier,
		Wasm:         WasmQuerier(wasm),
		Distribution: DistributionQuerier(distKeeper),
	}
}

func (e QueryPlugins) Merge(o *QueryPlugins) QueryPlugins {
	// only update if this is non-nil and then only set values
	if o == nil {
		return e
	}
	if o.Bank != nil {
		e.Bank = o.Bank
	}
	if o.Custom != nil {
		e.Custom = o.Custom
	}
	if o.IBC != nil {
		e.IBC = o.IBC
	}
	if o.Staking != nil {
		e.Staking = o.Staking
	}
	if o.Stargate != nil {
		e.Stargate = o.Stargate
	}
	if o.Grpc != nil {
		e.Grpc = o.Grpc
	}
	if o.Wasm != nil {
		e.Wasm = o.Wasm
	}
	if o.Distribution != nil {
		e.Distribution = o.Distribution
	}
	return e
}

// HandleQuery executes the requested query
func (e QueryPlugins) HandleQuery(ctx sdk.Context, caller sdk.AccAddress, req wasmvmtypes.QueryRequest) ([]byte, error) {
	// do the query
	switch {
	case req.Bank != nil:
		return e.Bank(ctx, req.Bank)
	case req.Custom != nil:
		return e.Custom(ctx, req.Custom)
	case req.IBC != nil:
		return e.IBC(ctx, caller, req.IBC)
	case req.Staking != nil:
		return e.Staking(ctx, req.Staking)
	case req.Stargate != nil:
		return e.Stargate(ctx, req.Stargate)
	case req.Grpc != nil:
		resp, err := e.Grpc(ctx, req.Grpc)
		if err != nil {
			return nil, err
		}
		// Marshaling the response here instead of inside the query
		// plugin makes sure that the response is always protobuf encoded.
		return proto.Marshal(resp)
	case req.Wasm != nil:
		return e.Wasm(ctx, req.Wasm)
	case req.Distribution != nil:
		return e.Distribution(ctx, req.Distribution)
	}
	return nil, wasmvmtypes.Unknown{}
}

func BankQuerier(bankKeeper types.BankViewKeeper) func(ctx sdk.Context, request *wasmvmtypes.BankQuery) ([]byte, error) {
	return func(ctx sdk.Context, request *wasmvmtypes.BankQuery) ([]byte, error) {
		if request.AllBalances != nil {
			addr, err := sdk.AccAddressFromBech32(request.AllBalances.Address)
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, request.AllBalances.Address)
			}
			coins := bankKeeper.GetAllBalances(ctx, addr)
			res := wasmvmtypes.AllBalancesResponse{
				Amount: ConvertSdkCoinsToWasmCoins(coins),
			}
			return json.Marshal(res)
		}
		if request.Balance != nil {
			addr, err := sdk.AccAddressFromBech32(request.Balance.Address)
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, request.Balance.Address)
			}
			coin := bankKeeper.GetBalance(ctx, addr, request.Balance.Denom)
			res := wasmvmtypes.BalanceResponse{
				Amount: wasmvmtypes.Coin{
					Denom:  coin.Denom,
					Amount: coin.Amount.String(),
				},
			}
			return json.Marshal(res)
		}
		if request.Supply != nil {
			coin := bankKeeper.GetSupply(ctx, request.Supply.Denom)
			res := wasmvmtypes.SupplyResponse{
				Amount: wasmvmtypes.Coin{
					Denom:  coin.Denom,
					Amount: coin.Amount.String(),
				},
			}
			return json.Marshal(res)
		}
		if request.DenomMetadata != nil {
			denomMetadata, ok := bankKeeper.GetDenomMetaData(ctx, request.DenomMetadata.Denom)
			if !ok {
				return nil, errorsmod.Wrap(sdkerrors.ErrNotFound, request.DenomMetadata.Denom)
			}
			res := wasmvmtypes.DenomMetadataResponse{
				Metadata: ConvertSdkDenomMetadataToWasmDenomMetadata(denomMetadata),
			}
			return json.Marshal(res)
		}
		if request.AllDenomMetadata != nil {
			bankQueryRes, err := bankKeeper.DenomsMetadata(ctx, ConvertToDenomsMetadataRequest(request.AllDenomMetadata))
			if err != nil {
				return nil, sdkerrors.ErrInvalidRequest
			}
			res := wasmvmtypes.AllDenomMetadataResponse{
				Metadata: ConvertSdkDenomMetadatasToWasmDenomMetadatas(bankQueryRes.Metadatas),
				NextKey:  bankQueryRes.Pagination.NextKey,
			}
			return json.Marshal(res)
		}
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown BankQuery variant"}
	}
}

func NoCustomQuerier(sdk.Context, json.RawMessage) ([]byte, error) {
	return nil, wasmvmtypes.UnsupportedRequest{Kind: "custom"}
}

func IBCQuerier(wasm contractMetaDataSource, channelKeeper types.ChannelKeeper) func(ctx sdk.Context, caller sdk.AccAddress, request *wasmvmtypes.IBCQuery) ([]byte, error) {
	return func(ctx sdk.Context, caller sdk.AccAddress, request *wasmvmtypes.IBCQuery) ([]byte, error) {
		if request.PortID != nil {
			contractInfo := wasm.GetContractInfo(ctx, caller)
			res := wasmvmtypes.PortIDResponse{
				PortID: contractInfo.IBCPortID,
			}
			return json.Marshal(res)
		}
		if request.ListChannels != nil {
			portID := request.ListChannels.PortID
			if portID == "" { // then fallback to contract port address
				portID = wasm.GetContractInfo(ctx, caller).IBCPortID
			}
			var channels wasmvmtypes.Array[wasmvmtypes.IBCChannel]
			if portID != "" { // then return empty list for non ibc contracts; no channels possible
				gotChannels := channelKeeper.GetAllChannelsWithPortPrefix(ctx, portID)
				channels = make(wasmvmtypes.Array[wasmvmtypes.IBCChannel], 0, len(gotChannels))
				for _, ch := range gotChannels {
					if ch.State != channeltypes.OPEN {
						continue
					}
					channels = append(channels, wasmvmtypes.IBCChannel{
						Endpoint: wasmvmtypes.IBCEndpoint{
							PortID:    ch.PortId,
							ChannelID: ch.ChannelId,
						},
						CounterpartyEndpoint: wasmvmtypes.IBCEndpoint{
							PortID:    ch.Counterparty.PortId,
							ChannelID: ch.Counterparty.ChannelId,
						},
						Order:        ch.Ordering.String(),
						Version:      ch.Version,
						ConnectionID: ch.ConnectionHops[0],
					})
				}
			}
			res := wasmvmtypes.ListChannelsResponse{
				Channels: channels,
			}
			return json.Marshal(res)
		}
		if request.Channel != nil {
			channelID := request.Channel.ChannelID
			portID := request.Channel.PortID
			if portID == "" {
				contractInfo := wasm.GetContractInfo(ctx, caller)
				portID = contractInfo.IBCPortID
			}
			got, found := channelKeeper.GetChannel(ctx, portID, channelID)
			var channel *wasmvmtypes.IBCChannel
			// it must be in open state
			if found && got.State == channeltypes.OPEN {
				channel = &wasmvmtypes.IBCChannel{
					Endpoint: wasmvmtypes.IBCEndpoint{
						PortID:    portID,
						ChannelID: channelID,
					},
					CounterpartyEndpoint: wasmvmtypes.IBCEndpoint{
						PortID:    got.Counterparty.PortId,
						ChannelID: got.Counterparty.ChannelId,
					},
					Order:        got.Ordering.String(),
					Version:      got.Version,
					ConnectionID: got.ConnectionHops[0],
				}
			}
			res := wasmvmtypes.ChannelResponse{
				Channel: channel,
			}
			return json.Marshal(res)
		}
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown IBCQuery variant"}
	}
}

// RejectGrpcQuerier is a querier that rejects all gRPC queries.
//
// Use AcceptListGrpcQuerier instead to create a list of accepted query types.
func RejectGrpcQuerier(ctx sdk.Context, request *wasmvmtypes.GrpcQuery) (proto.Message, error) {
	return nil, wasmvmtypes.UnsupportedRequest{Kind: "gRPC queries are disabled on this chain"}
}

var _ grpcQuerierFn = RejectGrpcQuerier // just a type check

// AcceptListGrpcQuerier supports a preconfigured set of gRPC queries only.
// All arguments must be non nil.
//
// Warning: Chains need to test and maintain their accept list carefully.
// There were critical consensus breaking issues in the past with non-deterministic behavior in the SDK.
//
// These queries can be set via WithQueryPlugins option in the wasm keeper constructor:
// WithQueryPlugins(&QueryPlugins{Grpc: AcceptListGrpcQuerier(acceptList, queryRouter, codec)})
func AcceptListGrpcQuerier(acceptList AcceptedQueries, queryRouter GRPCQueryRouter, codec codec.Codec) grpcQuerierFn {
	return func(ctx sdk.Context, request *wasmvmtypes.GrpcQuery) (proto.Message, error) {
		protoResponseFn, accepted := acceptList[request.Path]
		if !accepted {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", request.Path)}
		}

		handler := queryRouter.Route(request.Path)
		if handler == nil {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("No route to query '%s'", request.Path)}
		}

		res, err := handler(ctx, &abci.RequestQuery{
			Data: request.Data,
			Path: request.Path,
		})
		if err != nil {
			return nil, err
		}

		protoResponse := protoResponseFn()
		// decode the query response into the expected protobuf message
		err = codec.Unmarshal(res.Value, protoResponse)
		if err != nil {
			return nil, err
		}

		return protoResponse, nil
	}
}

// RejectStargateQuerier is a querier that rejects all stargate queries.
//
// Use AcceptListStargateQuerier instead to create a list of accepted query types.
func RejectStargateQuerier(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
	return nil, wasmvmtypes.UnsupportedRequest{Kind: "Stargate queries are disabled on this chain"}
}

var _ stargateQuerierFn = RejectStargateQuerier // just a type check

// AcceptedQueries defines accepted Stargate or gRPC queries as a map where the key is the query path
// and the value is a function returning a proto.Message.
//
// For example:
//
//	acceptList["/cosmos.auth.v1beta1.Query/Account"] = func() proto.Message {
//	    return &authtypes.QueryAccountResponse{}
//	}
type AcceptedQueries map[string]func() proto.Message

// AcceptListStargateQuerier supports a preconfigured set of stargate queries only.
// All arguments must be non nil.
//
// Warning: Chains need to test and maintain their accept list carefully.
// There were critical consensus breaking issues in the past with non-deterministic behavior in the SDK.
//
// These queries can be set via WithQueryPlugins option in the wasm keeper constructor:
// WithQueryPlugins(&QueryPlugins{Stargate: AcceptListStargateQuerier(acceptList, queryRouter, codec)})
func AcceptListStargateQuerier(acceptList AcceptedQueries, queryRouter GRPCQueryRouter, codec codec.Codec) stargateQuerierFn {
	return func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
		protoResponseFn, accepted := acceptList[request.Path]
		if !accepted {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", request.Path)}
		}

		route := queryRouter.Route(request.Path)
		if route == nil {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("No route to query '%s'", request.Path)}
		}

		res, err := route(ctx, &abci.RequestQuery{
			Data: request.Data,
			Path: request.Path,
		})
		if err != nil {
			return nil, err
		}

		protoResponse := protoResponseFn()
		return ConvertProtoToJSONMarshal(codec, protoResponse, res.Value)
	}
}

func StakingQuerier(keeper types.StakingKeeper, distKeeper types.DistributionKeeper) func(ctx sdk.Context, request *wasmvmtypes.StakingQuery) ([]byte, error) {
	return func(ctx sdk.Context, request *wasmvmtypes.StakingQuery) ([]byte, error) {
		if request.BondedDenom != nil {
			denom, err := keeper.BondDenom(ctx)
			if err != nil {
				return nil, errorsmod.Wrap(err, "bond denom")
			}
			res := wasmvmtypes.BondedDenomResponse{
				Denom: denom,
			}
			return json.Marshal(res)
		}
		if request.AllValidators != nil {
			validators, err := keeper.GetBondedValidatorsByPower(ctx)
			if err != nil {
				return nil, err
			}
			// validators := keeper.GetAllValidators(ctx)
			wasmVals := make([]wasmvmtypes.Validator, len(validators))
			for i, v := range validators {
				wasmVals[i] = wasmvmtypes.Validator{
					Address:       v.OperatorAddress,
					Commission:    v.Commission.Rate.String(),
					MaxCommission: v.Commission.MaxRate.String(),
					MaxChangeRate: v.Commission.MaxChangeRate.String(),
				}
			}
			res := wasmvmtypes.AllValidatorsResponse{
				Validators: wasmVals,
			}
			return json.Marshal(res)
		}
		if request.Validator != nil {
			valAddr, err := sdk.ValAddressFromBech32(request.Validator.Address)
			if err != nil {
				return nil, err
			}

			res := wasmvmtypes.ValidatorResponse{}
			v, err := keeper.GetValidator(ctx, valAddr)
			switch {
			case stakingtypes.ErrNoValidatorFound.Is(err): // return empty result for backwards compatibility. Changed in SDK 50
			case err != nil:
				return nil, err
			default:
				res.Validator = &wasmvmtypes.Validator{
					Address:       v.OperatorAddress,
					Commission:    v.Commission.Rate.String(),
					MaxCommission: v.Commission.MaxRate.String(),
					MaxChangeRate: v.Commission.MaxChangeRate.String(),
				}
			}
			return json.Marshal(res)
		}
		if request.AllDelegations != nil {
			delegator, err := sdk.AccAddressFromBech32(request.AllDelegations.Delegator)
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, request.AllDelegations.Delegator)
			}
			sdkDels, err := keeper.GetAllDelegatorDelegations(ctx, delegator)
			if err != nil {
				return nil, err
			}
			delegations, err := sdkToDelegations(ctx, keeper, sdkDels)
			if err != nil {
				return nil, err
			}
			res := wasmvmtypes.AllDelegationsResponse{
				Delegations: delegations,
			}
			return json.Marshal(res)
		}
		if request.Delegation != nil {
			delegator, err := sdk.AccAddressFromBech32(request.Delegation.Delegator)
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, request.Delegation.Delegator)
			}
			validator, err := sdk.ValAddressFromBech32(request.Delegation.Validator)
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, request.Delegation.Validator)
			}

			var res wasmvmtypes.DelegationResponse
			d, err := keeper.GetDelegation(ctx, delegator, validator)
			switch {
			case stakingtypes.ErrNoDelegation.Is(err): // return empty result for backwards compatibility. Changed in SDK 50
			case err != nil:
				return nil, err
			default:
				res.Delegation, err = sdkToFullDelegation(ctx, keeper, distKeeper, d)
				if err != nil {
					return nil, err
				}
			}
			return json.Marshal(res)
		}
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown Staking variant"}
	}
}

func sdkToDelegations(ctx sdk.Context, keeper types.StakingKeeper, delegations []stakingtypes.Delegation) (wasmvmtypes.Array[wasmvmtypes.Delegation], error) {
	result := make([]wasmvmtypes.Delegation, len(delegations))
	bondDenom, err := keeper.BondDenom(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "bond denom")
	}

	for i, d := range delegations {
		delAddr, err := sdk.AccAddressFromBech32(d.DelegatorAddress)
		if err != nil {
			return nil, errorsmod.Wrap(err, "delegator address")
		}
		valAddr, err := sdk.ValAddressFromBech32(d.ValidatorAddress)
		if err != nil {
			return nil, errorsmod.Wrap(err, "validator address")
		}

		// shares to amount logic comes from here:
		// https://github.com/cosmos/cosmos-sdk/blob/v0.38.3/x/staking/keeper/querier.go#L404
		val, err := keeper.GetValidator(ctx, valAddr)
		if err != nil { // is stakingtypes.ErrNoValidatorFound
			return nil, errorsmod.Wrap(err, "can't load validator for delegation")
		}
		amount := sdk.NewCoin(bondDenom, val.TokensFromShares(d.Shares).TruncateInt())

		result[i] = wasmvmtypes.Delegation{
			Delegator: delAddr.String(),
			Validator: valAddr.String(),
			Amount:    ConvertSdkCoinToWasmCoin(amount),
		}
	}
	return result, nil
}

func sdkToFullDelegation(ctx sdk.Context, keeper types.StakingKeeper, distKeeper types.DistributionKeeper, delegation stakingtypes.Delegation) (*wasmvmtypes.FullDelegation, error) {
	delAddr, err := sdk.AccAddressFromBech32(delegation.DelegatorAddress)
	if err != nil {
		return nil, errorsmod.Wrap(err, "delegator address")
	}
	valAddr, err := sdk.ValAddressFromBech32(delegation.ValidatorAddress)
	if err != nil {
		return nil, errorsmod.Wrap(err, "validator address")
	}
	val, err := keeper.GetValidator(ctx, valAddr)
	if err != nil { // is stakingtypes.ErrNoValidatorFound
		return nil, errorsmod.Wrap(err, "can't load validator for delegation")
	}
	bondDenom, err := keeper.BondDenom(ctx)
	if err != nil {
		return nil, errorsmod.Wrap(err, "bond denom")
	}

	amount := sdk.NewCoin(bondDenom, val.TokensFromShares(delegation.Shares).TruncateInt())

	delegationCoins := ConvertSdkCoinToWasmCoin(amount)

	// FIXME: this is very rough but better than nothing...
	// https://github.com/CosmWasm/wasmd/issues/282
	// if this (val, delegate) pair is receiving a redelegation, it cannot redelegate more
	// otherwise, it can redelegate the full amount
	// (there are cases of partial funds redelegated, but this is a start)
	redelegateCoins := wasmvmtypes.NewCoin(0, bondDenom)
	found, err := keeper.HasReceivingRedelegation(ctx, delAddr, valAddr)
	if err != nil {
		return nil, err
	}
	if !found {
		redelegateCoins = delegationCoins
	}

	// FIXME: make a cleaner way to do this (modify the sdk)
	// we need the info from `distKeeper.calculateDelegationRewards()`, but it is not public
	// neither is `queryDelegationRewards(ctx sdk.Context, _ []string, req abci.RequestQuery, k Keeper)`
	// so we go through the front door of the querier....
	accRewards, err := getAccumulatedRewards(ctx, distKeeper, delegation)
	if err != nil {
		return nil, err
	}

	return &wasmvmtypes.FullDelegation{
		Delegator:          delAddr.String(),
		Validator:          valAddr.String(),
		Amount:             delegationCoins,
		AccumulatedRewards: accRewards,
		CanRedelegate:      redelegateCoins,
	}, nil
}

// FIXME: simplify this enormously when
// https://github.com/cosmos/cosmos-sdk/issues/7466 is merged
func getAccumulatedRewards(ctx sdk.Context, distKeeper types.DistributionKeeper, delegation stakingtypes.Delegation) ([]wasmvmtypes.Coin, error) {
	// Try to get *delegator* reward info!
	params := distributiontypes.QueryDelegationRewardsRequest{
		DelegatorAddress: delegation.DelegatorAddress,
		ValidatorAddress: delegation.ValidatorAddress,
	}
	cache, _ := ctx.CacheContext()
	qres, err := distKeeper.DelegationRewards(cache, &params)
	if err != nil {
		return nil, err
	}

	// now we have it, convert it into wasmvm types
	rewards := make([]wasmvmtypes.Coin, len(qres.Rewards))
	for i, r := range qres.Rewards {
		rewards[i] = wasmvmtypes.Coin{
			Denom:  r.Denom,
			Amount: r.Amount.TruncateInt().String(),
		}
	}
	return rewards, nil
}

func WasmQuerier(k wasmQueryKeeper) func(ctx sdk.Context, request *wasmvmtypes.WasmQuery) ([]byte, error) {
	return func(ctx sdk.Context, request *wasmvmtypes.WasmQuery) ([]byte, error) {
		switch {
		case request.Smart != nil:
			addr, err := sdk.AccAddressFromBech32(request.Smart.ContractAddr)
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, request.Smart.ContractAddr)
			}
			msg := types.RawContractMessage(request.Smart.Msg)
			if err := msg.ValidateBasic(); err != nil {
				return nil, errorsmod.Wrap(err, "json msg")
			}
			return k.QuerySmart(ctx, addr, msg)
		case request.Raw != nil:
			addr, err := sdk.AccAddressFromBech32(request.Raw.ContractAddr)
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, request.Raw.ContractAddr)
			}
			return k.QueryRaw(ctx, addr, request.Raw.Key), nil
		case request.ContractInfo != nil:
			contractAddr := request.ContractInfo.ContractAddr
			addr, err := sdk.AccAddressFromBech32(contractAddr)
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, contractAddr)
			}
			info := k.GetContractInfo(ctx, addr)
			if info == nil {
				return nil, types.ErrNoSuchContractFn(contractAddr).
					Wrapf("address %s", contractAddr)
			}
			res := wasmvmtypes.ContractInfoResponse{
				CodeID:   info.CodeID,
				Creator:  info.Creator,
				Admin:    info.Admin,
				Pinned:   k.IsPinnedCode(ctx, info.CodeID),
				IBCPort:  info.IBCPortID,
				IBC2Port: info.IBC2PortID,
			}
			return json.Marshal(res)
		case request.CodeInfo != nil:
			if request.CodeInfo.CodeID == 0 {
				return nil, types.ErrEmpty.Wrap("code id")
			}
			info := k.GetCodeInfo(ctx, request.CodeInfo.CodeID)
			if info == nil {
				return nil, types.ErrNoSuchCodeFn(request.CodeInfo.CodeID).
					Wrapf("code id %d", request.CodeInfo.CodeID)
			}

			res := wasmvmtypes.CodeInfoResponse{
				CodeID:   request.CodeInfo.CodeID,
				Creator:  info.Creator,
				Checksum: info.CodeHash,
			}
			return json.Marshal(res)
		case request.RawRange != nil:
			contractAddr := request.RawRange.ContractAddr
			addr, err := sdk.AccAddressFromBech32(contractAddr)
			if err != nil {
				return nil, errorsmod.Wrap(sdkerrors.ErrInvalidAddress, contractAddr)
			}

			var reverse bool
			switch request.RawRange.Order {
			case "ascending":
				reverse = false
			case "descending":
				reverse = true
			default:
				return nil, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "unknown order %s", request.RawRange.Order)
			}
			data, nextKey := k.QueryRawRange(ctx, addr, request.RawRange.Start, request.RawRange.End, request.RawRange.Limit, reverse)
			res := wasmvmtypes.RawRangeResponse{
				Data:    data,
				NextKey: nextKey,
			}
			return json.Marshal(res)

		}
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown WasmQuery variant"}
	}
}

func DistributionQuerier(k types.DistributionKeeper) func(ctx sdk.Context, request *wasmvmtypes.DistributionQuery) ([]byte, error) {
	return func(ctx sdk.Context, req *wasmvmtypes.DistributionQuery) ([]byte, error) {
		switch {
		case req.DelegatorWithdrawAddress != nil:
			got, err := k.DelegatorWithdrawAddress(ctx, &distributiontypes.QueryDelegatorWithdrawAddressRequest{DelegatorAddress: req.DelegatorWithdrawAddress.DelegatorAddress})
			if err != nil {
				return nil, err
			}
			return json.Marshal(wasmvmtypes.DelegatorWithdrawAddressResponse{
				WithdrawAddress: got.WithdrawAddress,
			})
		case req.DelegationRewards != nil:
			got, err := k.DelegationRewards(ctx, &distributiontypes.QueryDelegationRewardsRequest{
				DelegatorAddress: req.DelegationRewards.DelegatorAddress,
				ValidatorAddress: req.DelegationRewards.ValidatorAddress,
			})
			if err != nil {
				return nil, err
			}
			return json.Marshal(wasmvmtypes.DelegationRewardsResponse{
				Rewards: ConvertSDKDecCoinsToWasmDecCoins(got.Rewards),
			})
		case req.DelegationTotalRewards != nil:
			got, err := k.DelegationTotalRewards(ctx, &distributiontypes.QueryDelegationTotalRewardsRequest{
				DelegatorAddress: req.DelegationTotalRewards.DelegatorAddress,
			})
			if err != nil {
				return nil, err
			}
			return json.Marshal(wasmvmtypes.DelegationTotalRewardsResponse{
				Rewards: ConvertSDKDelegatorRewardsToWasmRewards(got.Rewards),
				Total:   ConvertSDKDecCoinsToWasmDecCoins(got.Total),
			})
		case req.DelegatorValidators != nil:
			got, err := k.DelegatorValidators(ctx, &distributiontypes.QueryDelegatorValidatorsRequest{
				DelegatorAddress: req.DelegatorValidators.DelegatorAddress,
			})
			if err != nil {
				return nil, err
			}
			return json.Marshal(wasmvmtypes.DelegatorValidatorsResponse{
				Validators: got.Validators,
			})
		}
		return nil, wasmvmtypes.UnsupportedRequest{Kind: "unknown distribution query"}
	}
}

// ConvertSDKDelegatorRewardsToWasmRewards convert sdk to wasmvm type
func ConvertSDKDelegatorRewardsToWasmRewards(rewards []distributiontypes.DelegationDelegatorReward) []wasmvmtypes.DelegatorReward {
	r := make([]wasmvmtypes.DelegatorReward, len(rewards))
	for i, v := range rewards {
		r[i] = wasmvmtypes.DelegatorReward{
			Reward:           ConvertSDKDecCoinsToWasmDecCoins(v.Reward),
			ValidatorAddress: v.ValidatorAddress,
		}
	}
	return r
}

// ConvertSDKDecCoinsToWasmDecCoins convert sdk to wasmvm type
func ConvertSDKDecCoinsToWasmDecCoins(src sdk.DecCoins) []wasmvmtypes.DecCoin {
	r := make([]wasmvmtypes.DecCoin, len(src))
	for i, v := range src {
		r[i] = wasmvmtypes.DecCoin{
			Amount: v.Amount.String(),
			Denom:  v.Denom,
		}
	}
	return r
}

// ConvertSdkCoinsToWasmCoins convert sdk type to wasmvm coins type
func ConvertSdkCoinsToWasmCoins(coins []sdk.Coin) wasmvmtypes.Array[wasmvmtypes.Coin] {
	converted := make(wasmvmtypes.Array[wasmvmtypes.Coin], len(coins))
	for i, c := range coins {
		converted[i] = ConvertSdkCoinToWasmCoin(c)
	}
	return converted
}

// ConvertSdkCoinToWasmCoin convert sdk type to wasmvm coin type
func ConvertSdkCoinToWasmCoin(coin sdk.Coin) wasmvmtypes.Coin {
	return wasmvmtypes.Coin{
		Denom:  coin.Denom,
		Amount: coin.Amount.String(),
	}
}

func ConvertToDenomsMetadataRequest(wasmRequest *wasmvmtypes.AllDenomMetadataQuery) *banktypes.QueryDenomsMetadataRequest {
	ret := &banktypes.QueryDenomsMetadataRequest{}
	if wasmRequest.Pagination != nil {
		ret.Pagination = &query.PageRequest{
			Key:     wasmRequest.Pagination.Key,
			Limit:   uint64(wasmRequest.Pagination.Limit),
			Reverse: wasmRequest.Pagination.Reverse,
		}
	}
	return ret
}

func ConvertSdkDenomMetadatasToWasmDenomMetadatas(metadata []banktypes.Metadata) []wasmvmtypes.DenomMetadata {
	converted := make([]wasmvmtypes.DenomMetadata, len(metadata))
	for i, m := range metadata {
		converted[i] = ConvertSdkDenomMetadataToWasmDenomMetadata(m)
	}
	return converted
}

func ConvertSdkDenomMetadataToWasmDenomMetadata(metadata banktypes.Metadata) wasmvmtypes.DenomMetadata {
	return wasmvmtypes.DenomMetadata{
		Description: metadata.Description,
		DenomUnits:  ConvertSdkDenomUnitsToWasmDenomUnits(metadata.DenomUnits),
		Base:        metadata.Base,
		Display:     metadata.Display,
		Name:        metadata.Name,
		Symbol:      metadata.Symbol,
		URI:         metadata.URI,
		URIHash:     metadata.URIHash,
	}
}

func ConvertSdkDenomUnitsToWasmDenomUnits(denomUnits []*banktypes.DenomUnit) []wasmvmtypes.DenomUnit {
	converted := make([]wasmvmtypes.DenomUnit, len(denomUnits))
	for i, u := range denomUnits {
		converted[i] = wasmvmtypes.DenomUnit{
			Denom:    u.Denom,
			Exponent: u.Exponent,
			Aliases:  u.Aliases,
		}
		// Returning nil may break cosmwasm-std
		if u.Aliases == nil {
			converted[i].Aliases = []string{}
		}
	}
	return converted
}

// ConvertProtoToJSONMarshal  unmarshals the given bytes into a proto message and then marshals it to json.
// This is done so that clients calling stargate queries do not need to define their own proto unmarshalers,
// being able to use response directly by json marshaling, which is supported in cosmwasm.
func ConvertProtoToJSONMarshal(cdc codec.Codec, protoResponse proto.Message, bz []byte) ([]byte, error) {
	// unmarshal binary into stargate response data structure
	err := cdc.Unmarshal(bz, protoResponse)
	if err != nil {
		return nil, errorsmod.Wrap(err, "to proto")
	}

	bz, err = cdc.MarshalJSON(protoResponse)
	if err != nil {
		return nil, errorsmod.Wrap(err, "to json")
	}

	protoResponse.Reset()
	return bz, nil
}

var _ WasmVMQueryHandler = WasmVMQueryHandlerFn(nil)

// WasmVMQueryHandlerFn is a helper to construct a function based query handler.
type WasmVMQueryHandlerFn func(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error)

// HandleQuery delegates call into wrapped WasmVMQueryHandlerFn
func (w WasmVMQueryHandlerFn) HandleQuery(ctx sdk.Context, caller sdk.AccAddress, request wasmvmtypes.QueryRequest) ([]byte, error) {
	return w(ctx, caller, request)
}
