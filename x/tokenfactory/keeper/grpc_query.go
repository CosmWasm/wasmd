package keeper

import (
	"context"

	"github.com/CosmWasm/wasmd/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var _ types.QueryServer = Keeper{}

func (k Keeper) Params(goCtx context.Context, req *types.QueryParamsRequest) (*types.QueryParamsResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	params := k.GetParams(ctx)

	return &types.QueryParamsResponse{Params: params}, nil
}

func (k Keeper) DenomsFromCreator(goCtx context.Context, req *types.QueryDenomsFromCreatorRequest) (*types.QueryDenomsFromCreatorResponse, error) {
	if req == nil {
		return nil, status.Error(codes.InvalidArgument, "invalid request")
	}

	_, err := sdk.AccAddressFromBech32(req.Creator)
	if err != nil {
		return nil, sdkerrors.Wrapf(sdkerrors.ErrUnknownAddress, "Unkown address of creator %v", req.Creator)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)

	denoms := k.getAllDenomsFromCreator(ctx, req.Creator)

	return &types.QueryDenomsFromCreatorResponse{
		Denoms: denoms,
	}, nil
}
