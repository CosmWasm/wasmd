package keeper

import (
	"context"

	sdk "github.com/Finschia/finschia-sdk/types"
	sdkerrors "github.com/Finschia/finschia-sdk/types/errors"

	wasmtypes "github.com/Finschia/wasmd/x/wasm/types"
	"github.com/Finschia/wasmd/x/wasmplus/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	keeper wasmtypes.ContractOpsKeeper
}

func NewMsgServerImpl(k wasmtypes.ContractOpsKeeper) types.MsgServer {
	return &msgServer{keeper: k}
}

func (m msgServer) StoreCodeAndInstantiateContract(goCtx context.Context,
	msg *types.MsgStoreCodeAndInstantiateContract,
) (*types.MsgStoreCodeAndInstantiateContractResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "sender")
	}
	codeID, _, err := m.keeper.Create(ctx, senderAddr, msg.WASMByteCode, msg.InstantiatePermission)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(sdk.AttributeKeySender, msg.Sender),
	))

	var adminAddr sdk.AccAddress
	if msg.Admin != "" {
		adminAddr, err = sdk.AccAddressFromBech32(msg.Admin)
		if err != nil {
			return nil, sdkerrors.Wrap(err, "admin")
		}
	}

	contractAddr, data, err := m.keeper.Instantiate(ctx, codeID, senderAddr, adminAddr, msg.Msg,
		msg.Label, msg.Funds)
	if err != nil {
		return nil, err
	}

	return &types.MsgStoreCodeAndInstantiateContractResponse{
		CodeID:  codeID,
		Address: contractAddr.String(),
		Data:    data,
	}, nil
}
