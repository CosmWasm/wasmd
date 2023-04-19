package keeper

import (
	"context"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/Finschia/finschia-sdk/store/prefix"
	sdk "github.com/Finschia/finschia-sdk/types"
	"github.com/Finschia/finschia-sdk/types/query"

	wasmtypes "github.com/Finschia/wasmd/x/wasm/types"
	"github.com/Finschia/wasmd/x/wasmplus/types"
)

type queryKeeper interface {
	IterateInactiveContracts(ctx sdk.Context, fn func(contractAddress sdk.AccAddress) bool)
	IsInactiveContract(ctx sdk.Context, contractAddress sdk.AccAddress) bool
	HasContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress) bool
}

var _ types.QueryServer = &grpcQuerier{}

type grpcQuerier struct {
	keeper   queryKeeper
	storeKey sdk.StoreKey
}

// newGrpcQuerier constructor
func newGrpcQuerier(storeKey sdk.StoreKey, keeper queryKeeper) *grpcQuerier {
	return &grpcQuerier{storeKey: storeKey, keeper: keeper}
}

func (q grpcQuerier) InactiveContracts(c context.Context, req *types.QueryInactiveContractsRequest) (*types.QueryInactiveContractsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	addresses := make([]string, 0)
	prefixStore := prefix.NewStore(ctx.KVStore(q.storeKey), types.InactiveContractPrefix)
	pageRes, err := query.FilteredPaginate(prefixStore, req.Pagination, func(key []byte, value []byte, accumulate bool) (bool, error) {
		if accumulate {
			contractAddress := sdk.AccAddress(value)
			addresses = append(addresses, contractAddress.String())
		}
		return true, nil
	})
	if err != nil {
		return nil, err
	}
	return &types.QueryInactiveContractsResponse{
		Addresses:  addresses,
		Pagination: pageRes,
	}, nil
}

func (q grpcQuerier) InactiveContract(c context.Context, req *types.QueryInactiveContractRequest) (*types.QueryInactiveContractResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "empty request")
	}
	ctx := sdk.UnwrapSDKContext(c)

	contractAddr, err := sdk.AccAddressFromBech32(req.Address)
	if err != nil {
		return nil, err
	}

	if !q.keeper.HasContractInfo(ctx, contractAddr) {
		return nil, wasmtypes.ErrNotFound
	}

	inactivated := q.keeper.IsInactiveContract(ctx, contractAddr)
	return &types.QueryInactiveContractResponse{
		Inactivated: inactivated,
	}, nil
}
