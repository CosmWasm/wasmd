package app

import (
	"fmt"
	"sync"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"
	proto "github.com/gogo/protobuf/proto"
)

// stargateResponsePools keeps whitelist and its deterministic
// response binding for stargate queries.
// CONTRACT: since results of queries go into blocks, queries being added here should always be
// deterministic or can cause non-determinism in the state machine.
//
// The query is multi-threaded so we're using a sync.Pool
// to manage the allocation and de-allocation of newly created
// pb objects.
var stargateResponsePools = make(map[string]*sync.Pool)

// Note: When adding a migration here, we should also add it to the Async ICQ params in the upgrade.
// In the future we may want to find a better way to keep these in sync

//nolint:staticcheck
func init() {

	// staking
	setWhitelistedQuery("/cosmos.staking.v1beta1.Query/Delegation", &stakingtypes.QueryDelegationResponse{})
	setWhitelistedQuery("/cosmos.staking.v1beta1.Query/Params", &stakingtypes.QueryParamsResponse{})
	setWhitelistedQuery("/cosmos.staking.v1beta1.Query/Validator", &stakingtypes.QueryValidatorResponse{})

}

// getWhitelistedQuery returns the whitelisted query at the provided path.
// If the query does not exist, or it was setup wrong by the chain, this returns an error.
// CONTRACT: must call returnStargateResponseToPool in order to avoid pointless allocs.
func getWhitelistedQuery(queryPath string) (proto.Message, error) {
	protoResponseAny, isWhitelisted := stargateResponsePools[queryPath]
	if !isWhitelisted {
		return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("'%s' path is not allowed from the contract", queryPath)}
	}
	protoMarshaler, ok := protoResponseAny.Get().(proto.Message)
	if !ok {
		return nil, fmt.Errorf("failed to assert type to proto.Messager")
	}
	return protoMarshaler, nil
}

type protoTypeG[T any] interface {
	*T
	proto.Message
}

// setWhitelistedQuery sets the whitelisted query at the provided path.
// This method also creates a sync.Pool for the provided protoMarshaler.
// We use generics so we can properly instantiate an object that the
// queryPath expects as a response.
func setWhitelistedQuery[T any, PT protoTypeG[T]](queryPath string, _ PT) {
	stargateResponsePools[queryPath] = &sync.Pool{
		New: func() any {
			return PT(new(T))
		},
	}
}

// returnStargateResponseToPool returns the provided protoMarshaler to the appropriate pool based on it's query path.
func returnStargateResponseToPool(queryPath string, pb proto.Message) {
	stargateResponsePools[queryPath].Put(pb)
}

// StargateQuerier dispatches whitelisted stargate queries
func StargateQuerier(queryRouter baseapp.GRPCQueryRouter, cdc codec.Codec) func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
	return func(ctx sdk.Context, request *wasmvmtypes.StargateQuery) ([]byte, error) {
		protoResponseType, err := getWhitelistedQuery(request.Path)
		if err != nil {
			return nil, err
		}

		// no matter what happens after this point, we must return
		// the response type to prevent sync.Pool from leaking.
		defer returnStargateResponseToPool(request.Path, protoResponseType)

		route := queryRouter.Route(request.Path)
		if route == nil {
			return nil, wasmvmtypes.UnsupportedRequest{Kind: fmt.Sprintf("No route to query '%s'", request.Path)}
		}

		res, err := route(ctx, &abci.RequestQuery{
			Data:   request.Data,
			Path:   request.Path,
			Height: ctx.BlockHeight(),
			Prove:  false,
		})
		if err != nil {
			return nil, err
		}

		if res.Value == nil {
			return nil, fmt.Errorf("res returned from abci query route is nil")
		}

		return res.Value, nil
	}
}

func RegisterStargateQueries(queryRouter baseapp.GRPCQueryRouter, codec codec.Codec) []wasmkeeper.Option {
	queryPluginOpt := wasmkeeper.WithQueryPlugins(&wasmkeeper.QueryPlugins{
		Stargate: StargateQuerier(queryRouter, codec),
	})

	return []wasmkeeper.Option{
		queryPluginOpt,
	}
}
