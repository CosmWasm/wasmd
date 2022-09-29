package keeper

import (
	"context"

	"github.com/CosmWasm/wasmd/x/tokenfactory/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

type msgServer struct {
	Keeper
}

var _ types.MsgServer = msgServer{}

func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{Keeper: keeper}
}

func (k msgServer) CreateDenom(goCtx context.Context, msg *types.MsgCreateDenom) (*types.MsgCreateDenomResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	denom, err := k.Keeper.CreateDenom(ctx, msg.Sender, msg.Subdenom)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.TypeMsgCreateDenom,
			sdk.NewAttribute(types.AttributeCreator, msg.Sender),
			sdk.NewAttribute(types.AttributeNewTokenDenom, denom),
		),
	})

	return &types.MsgCreateDenomResponse{
		NewTokenDenom: msg.Subdenom,
	}, nil
}

func (k msgServer) Mint(goCtx context.Context, msg *types.MsgMint) (*types.MsgMintResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// TODO: logic mint
	// Check if denom exists
	_, found := k.bankKeeper.GetDenomMetaData(ctx, msg.Amount.Denom)
	if !found {
		return nil, types.ErrDenomDoesNotExist.Wrapf("denom: %s doesn't exists", msg.Amount.Denom)
	}

	// Check if sender is admin
	authorityMetadata, err := k.Keeper.GetAuthorityMetadata(ctx, msg.Amount.Denom)
	if err != nil {
		return nil, err
	}
	if msg.Sender != authorityMetadata.Admin {
		return nil, types.ErrUnauthorized.Wrapf("Only admin can mint coin")
	}

	// Mint coin
	accAddress, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	err = k.Keeper.mintTo(ctx, msg.Amount, accAddress)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.TypeMsgMint,
			sdk.NewAttribute(types.AttributeMintToAddress, msg.Sender),
			sdk.NewAttribute(types.AttributeAmount, msg.Amount.String()),
		),
	})

	return &types.MsgMintResponse{}, nil
}

func (k msgServer) Burn(goCtx context.Context, msg *types.MsgBurn) (*types.MsgBurnResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// TODO: logic burn
	// Check if denom exists
	_, found := k.bankKeeper.GetDenomMetaData(ctx, msg.Amount.Denom)
	if !found {
		return nil, types.ErrDenomDoesNotExist.Wrapf("denom: %s doesn't exists", msg.Amount.Denom)
	}

	// Check if sender is admin
	authorityMetadata, err := k.Keeper.GetAuthorityMetadata(ctx, msg.Amount.Denom)
	if err != nil {
		return nil, err
	}
	if msg.Sender != authorityMetadata.Admin {
		return nil, types.ErrUnauthorized.Wrapf("Only admin can burn coin")
	}

	// Burn coin
	accAddress, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, err
	}
	err = k.Keeper.burnFrom(ctx, msg.Amount, accAddress)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.TypeMsgBurn,
			sdk.NewAttribute(types.AttributeBurnFromAddress, msg.Sender),
			sdk.NewAttribute(types.AttributeAmount, msg.Amount.String()),
		),
	})

	return &types.MsgBurnResponse{}, nil
}

func (k msgServer) ChangeAdmin(goCtx context.Context, msg *types.MsgChangeAdmin) (*types.MsgChangeAdminResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	// TODO: logic change admin
	// Check if denom exists
	_, found := k.bankKeeper.GetDenomMetaData(ctx, msg.Denom)
	if !found {
		return nil, types.ErrDenomDoesNotExist.Wrapf("denom: %s doesn't exists", msg.Denom)
	}

	// Check if sender is admin
	authorityMetadata, err := k.Keeper.GetAuthorityMetadata(ctx, msg.Denom)
	if err != nil {
		return nil, err
	}
	if msg.Sender != authorityMetadata.Admin {
		return nil, types.ErrUnauthorized
	}

	// set new addmin
	err = k.Keeper.setAdmin(ctx, msg.Denom, msg.NewAdmin)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			types.TypeMsgChangeAdmin,
			sdk.NewAttribute(types.AttributeDenom, msg.Denom),
			sdk.NewAttribute(types.AttributeNewAdmin, msg.NewAdmin),
		),
	})

	return &types.MsgChangeAdminResponse{}, nil
}
