package keeper

import (
	"context"
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/evmutil/types"
)

type msgServer struct {
	keeper Keeper
}

// NewMsgServerImpl returns an implementation of the evmutil MsgServer interface
// for the provided Keeper.
func NewMsgServerImpl(keeper Keeper) types.MsgServer {
	return &msgServer{keeper: keeper}
}

var _ types.MsgServer = msgServer{}

// ConvertCoinToERC20 handles a MsgConvertCoinToERC20 message to convert
// sdk.Coin to Kava EVM tokens.
func (s msgServer) ConvertCoinToERC20(
	goCtx context.Context,
	msg *types.MsgConvertCoinToERC20,
) (*types.MsgConvertCoinToERC20Response, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	initiator, err := sdk.AccAddressFromBech32(msg.Initiator)
	if err != nil {
		return nil, fmt.Errorf("invalid Initiator address: %w", err)
	}

	receiver, err := types.NewInternalEVMAddressFromString(msg.Receiver)
	if err != nil {
		return nil, fmt.Errorf("invalid Receiver address: %w", err)
	}

	if err := s.keeper.ConvertCoinToERC20(
		ctx,
		initiator,
		receiver,
		*msg.Amount,
	); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Initiator),
		),
	)

	return &types.MsgConvertCoinToERC20Response{}, nil
}

// ConvertERC20ToCoin handles a MsgConvertERC20ToCoin message to convert
// sdk.Coin to Kava EVM tokens.
func (s msgServer) ConvertERC20ToCoin(
	goCtx context.Context,
	msg *types.MsgConvertERC20ToCoin,
) (*types.MsgConvertERC20ToCoinResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)

	initiator, err := types.NewInternalEVMAddressFromString(msg.Initiator)
	if err != nil {
		return nil, fmt.Errorf("invalid initiator address: %w", err)
	}

	receiver, err := sdk.AccAddressFromBech32(msg.Receiver)
	if err != nil {
		return nil, fmt.Errorf("invalid receiver address: %w", err)
	}

	contractAddr, err := types.NewInternalEVMAddressFromString(msg.OraiERC20Address)
	if err != nil {
		return nil, fmt.Errorf("invalid contract address: %w", err)
	}

	if err := s.keeper.ConvertERC20ToCoin(
		ctx,
		initiator,
		receiver,
		contractAddr,
		msg.Amount,
	); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.AttributeValueCategory),
			sdk.NewAttribute(sdk.AttributeKeySender, msg.Initiator),
		),
	)

	return &types.MsgConvertERC20ToCoinResponse{}, nil
}
