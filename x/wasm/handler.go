package wasm

import (
	"fmt"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	abci "github.com/tendermint/tendermint/abci/types"
)

// NewHandler returns a handler for "bank" type messages.
func NewHandler(k *Keeper) sdk.Handler {
	return func(ctx sdk.Context, msg sdk.Msg) (*sdk.Result, error) {
		ctx = ctx.WithEventManager(sdk.NewEventManager())

		switch msg := msg.(type) {
		case *MsgStoreCode:
			return handleStoreCode(ctx, k, msg)
		case *MsgInstantiateContract:
			return handleInstantiate(ctx, k, msg)
		case *MsgExecuteContract:
			return handleExecute(ctx, k, msg)
		case *MsgMigrateContract:
			return handleMigration(ctx, k, msg)
		case *MsgUpdateAdmin:
			return handleUpdateContractAdmin(ctx, k, msg)
		case *MsgClearAdmin:
			return handleClearContractAdmin(ctx, k, msg)
		default:
			errMsg := fmt.Sprintf("unrecognized wasm message type: %T", msg)
			return nil, sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, errMsg)
		}
	}
}

// filterMessageEvents returns the same events with all of type == EventTypeMessage removed.
// this is so only our top-level message event comes through
func filteredMessageEvents(manager *sdk.EventManager) []abci.Event {
	events := manager.ABCIEvents()
	res := make([]abci.Event, 0, len(events))
	for _, e := range events {
		if e.Type != sdk.EventTypeMessage {
			res = append(res, e)
		}
	}
	return res
}

func handleStoreCode(ctx sdk.Context, k *Keeper, msg *MsgStoreCode) (*sdk.Result, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "sender")
	}
	codeID, err := k.Create(ctx, senderAddr, msg.WASMByteCode, msg.Source, msg.Builder, msg.InstantiatePermission)
	if err != nil {
		return nil, err
	}

	ctx.EventManager().EmitEvents(sdk.Events{
		sdk.NewEvent(
			sdk.EventTypeMessage,
			sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
			sdk.NewAttribute(types.AttributeKeySigner, msg.Sender),
			sdk.NewAttribute(types.AttributeKeyCodeID, fmt.Sprintf("%d", codeID)),
		),
	})

	return &sdk.Result{
		Data:   []byte(fmt.Sprintf("%d", codeID)),
		Events: ctx.EventManager().ABCIEvents(),
	}, nil
}

func handleInstantiate(ctx sdk.Context, k *Keeper, msg *MsgInstantiateContract) (*sdk.Result, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "sender")
	}
	var adminAddr sdk.AccAddress
	if msg.Admin != "" {
		if adminAddr, err = sdk.AccAddressFromBech32(msg.Admin); err != nil {
			return nil, sdkerrors.Wrap(err, "admin")
		}
	}

	contractAddr, err := k.Instantiate(ctx, msg.CodeID, senderAddr, adminAddr, msg.InitMsg, msg.Label, msg.InitFunds)
	if err != nil {
		return nil, err
	}

	events := filteredMessageEvents(ctx.EventManager())
	custom := sdk.Events{sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
		sdk.NewAttribute(types.AttributeKeySigner, msg.Sender),
		sdk.NewAttribute(types.AttributeKeyCodeID, fmt.Sprintf("%d", msg.CodeID)),
		sdk.NewAttribute(types.AttributeKeyContract, contractAddr.String()),
	)}
	events = append(events, custom.ToABCIEvents()...)

	return &sdk.Result{
		Data:   []byte(contractAddr.String()),
		Events: events,
	}, nil
}

func handleExecute(ctx sdk.Context, k *Keeper, msg *MsgExecuteContract) (*sdk.Result, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "contract")
	}

	res, err := k.Execute(ctx, contractAddr, senderAddr, msg.Msg, msg.SentFunds)
	if err != nil {
		return nil, err
	}

	events := filteredMessageEvents(ctx.EventManager())
	custom := sdk.Events{sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
		sdk.NewAttribute(types.AttributeKeySigner, msg.Sender),
		sdk.NewAttribute(types.AttributeKeyContract, msg.Contract),
	),
	}
	events = append(events, custom.ToABCIEvents()...)

	res.Events = events
	return res, nil
}

func handleMigration(ctx sdk.Context, k *Keeper, msg *MsgMigrateContract) (*sdk.Result, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "contract")
	}

	res, err := k.Migrate(ctx, contractAddr, senderAddr, msg.CodeID, msg.MigrateMsg)
	if err != nil {
		return nil, err
	}

	events := filteredMessageEvents(ctx.EventManager())
	custom := sdk.Events{sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
		sdk.NewAttribute(types.AttributeKeySigner, msg.Sender),
		sdk.NewAttribute(types.AttributeKeyContract, msg.Contract),
	)}
	events = append(events, custom.ToABCIEvents()...)
	res.Events = events
	return res, nil
}

func handleUpdateContractAdmin(ctx sdk.Context, k *Keeper, msg *MsgUpdateAdmin) (*sdk.Result, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "contract")
	}
	newAdminAddr, err := sdk.AccAddressFromBech32(msg.NewAdmin)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "new admin")
	}

	if err := k.UpdateContractAdmin(ctx, contractAddr, senderAddr, newAdminAddr); err != nil {
		return nil, err
	}
	events := ctx.EventManager().Events()
	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
		sdk.NewAttribute(types.AttributeKeySigner, msg.Sender),
		sdk.NewAttribute(types.AttributeKeyContract, msg.Contract),
	)
	return &sdk.Result{
		Events: append(events, ourEvent).ToABCIEvents(),
	}, nil
}

func handleClearContractAdmin(ctx sdk.Context, k *Keeper, msg *MsgClearAdmin) (*sdk.Result, error) {
	senderAddr, err := sdk.AccAddressFromBech32(msg.Sender)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "sender")
	}
	contractAddr, err := sdk.AccAddressFromBech32(msg.Contract)
	if err != nil {
		return nil, sdkerrors.Wrap(err, "contract")
	}

	if err := k.ClearContractAdmin(ctx, contractAddr, senderAddr); err != nil {
		return nil, err
	}
	events := ctx.EventManager().Events()
	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, ModuleName),
		sdk.NewAttribute(types.AttributeKeySigner, msg.Sender),
		sdk.NewAttribute(types.AttributeKeyContract, msg.Contract),
	)
	return &sdk.Result{
		Events: append(events, ourEvent).ToABCIEvents(),
	}, nil
}
