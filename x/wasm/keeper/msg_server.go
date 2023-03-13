package keeper

import (
	"context"

	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/CosmWasm/wasmd/x/wasm/types"
)

var _ types.MsgServer = msgServer{}

type msgServer struct {
	contractKeeper types.ContractOpsKeeper
	wasmKeeper     Keeper
}

func NewMsgServerImpl(ck types.ContractOpsKeeper, wk Keeper) types.MsgServer {
	return &msgServer{contractKeeper: ck, wasmKeeper: wk}
}

func (m msgServer) StoreCode(goCtx context.Context, msg *types.MsgStoreCode) (*types.MsgStoreCodeResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}

	codeID, checksum, err := m.contractKeeper.Create(ctx, senderAddr, msg.WASMByteCode, msg.InstantiatePermission)
	if err != nil {
		return nil, err
	}

	return &types.MsgStoreCodeResponse{
		CodeID:   codeID,
		Checksum: checksum,
	}, nil
}

// InstantiateContract instantiate a new contract with classic sequence based address generation
func (m msgServer) InstantiateContract(goCtx context.Context, msg *types.MsgInstantiateContract) (*types.MsgInstantiateContractResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}
	var adminAddr sdk.AccAddress
	if msg.Admin != "" {
		if adminAddr, err = sdk.AccAddressFromBech32(msg.Admin); err != nil {
			return nil, errorsmod.Wrap(err, "admin")
		}
	}

	contractAddr, data, err := m.contractKeeper.Instantiate(ctx, msg.CodeID, senderAddr, adminAddr, msg.Msg, msg.Label, msg.Funds)
	if err != nil {
		return nil, err
	}

	return &types.MsgInstantiateContractResponse{
		Address: contractAddr.String(),
		Data:    data,
	}, nil
}

// InstantiateContract2 instantiate a new contract with predicatable address generated
func (m msgServer) InstantiateContract2(goCtx context.Context, msg *types.MsgInstantiateContract2) (*types.MsgInstantiateContract2Response, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}
	ctx := sdk.UnwrapSDKContext(goCtx)

	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}
	var adminAddr sdk.AccAddress
	if msg.Admin != "" {
		if adminAddr, err = sdk.AccAddressFromBech32(msg.Admin); err != nil {
			return nil, errorsmod.Wrap(err, "admin")
		}
	}

	contractAddr, data, err := m.contractKeeper.Instantiate2(ctx, msg.CodeID, senderAddr, adminAddr, msg.Msg, msg.Label, msg.Funds, msg.Salt, msg.FixMsg)
	if err != nil {
		return nil, err
	}

	return &types.MsgInstantiateContract2Response{
		Address: contractAddr.String(),
		Data:    data,
	}, nil
}

func (m msgServer) ExecuteContract(goCtx context.Context, msg *types.MsgExecuteContract) (*types.MsgExecuteContractResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, errorsmod.Wrap(err, "contract")
	}

	data, err := m.contractKeeper.Execute(ctx, contractAddr, senderAddr, msg.Msg, msg.Funds)
	if err != nil {
		return nil, err
	}

	return &types.MsgExecuteContractResponse{
		Data: data,
	}, nil
}

func (m msgServer) MigrateContract(goCtx context.Context, msg *types.MsgMigrateContract) (*types.MsgMigrateContractResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, errorsmod.Wrap(err, "contract")
	}

	data, err := m.contractKeeper.Migrate(ctx, contractAddr, senderAddr, msg.CodeID, msg.Msg)
	if err != nil {
		return nil, err
	}

	return &types.MsgMigrateContractResponse{
		Data: data,
	}, nil
}

func (m msgServer) UpdateAdmin(goCtx context.Context, msg *types.MsgUpdateAdmin) (*types.MsgUpdateAdminResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, errorsmod.Wrap(err, "contract")
	}
	newAdminAddr, err := sdk.AccAddressFromBech32(msg.NewAdmin)
	if err != nil {
		return nil, errorsmod.Wrap(err, "new admin")
	}

	if err := m.contractKeeper.UpdateContractAdmin(ctx, contractAddr, senderAddr, newAdminAddr); err != nil {
		return nil, err
	}

	return &types.MsgUpdateAdminResponse{}, nil
}

func (m msgServer) ClearAdmin(goCtx context.Context, msg *types.MsgClearAdmin) (*types.MsgClearAdminResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, errorsmod.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, errorsmod.Wrap(err, "contract")
	}

	if err := m.contractKeeper.ClearContractAdmin(ctx, contractAddr, senderAddr); err != nil {
		return nil, err
	}

	return &types.MsgClearAdminResponse{}, nil
}

func (m msgServer) UpdateInstantiateConfig(goCtx context.Context, msg *types.MsgUpdateInstantiateConfig) (*types.MsgUpdateInstantiateConfigResponse, error) {
	if err := msg.ValidateBasic(); err != nil {
		return nil, err
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := m.contractKeeper.SetAccessConfig(ctx, msg.CodeID, sdk.AccAddress(msg.Sender), *msg.NewInstantiatePermission); err != nil {
		return nil, err
	}

	return &types.MsgUpdateInstantiateConfigResponse{}, nil
}

func (m msgServer) UpdateParams(goCtx context.Context, req *types.MsgUpdateParams) (*types.MsgUpdateParamsResponse, error) {
	authority := m.wasmKeeper.authority
	if authority != req.Authority {
		return nil, errorsmod.Wrapf(types.ErrInvalid, "invalid authority; expected %s, got %s", authority, req.Authority)
	}

	ctx := sdk.UnwrapSDKContext(goCtx)
	if err := m.wasmKeeper.SetParams(ctx, req.Params); err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(sdk.AttributeKeySender, req.Authority),
		),
	)

	return &types.MsgUpdateParamsResponse{}, nil
}
