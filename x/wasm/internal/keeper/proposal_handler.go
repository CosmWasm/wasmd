package keeper

import (
	"fmt"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

const ( // TODO: same as in handler

	AttributeKeyContract = "contract_address"
	AttributeKeyCodeID   = "code_id"
	AttributeSigner      = "signer"
)

// NewWasmProposalHandler creates a new governance Handler for wasm proposals
func NewWasmProposalHandler(k Keeper, enabledTypes map[string]struct{}) govtypes.Handler {
	return func(ctx sdk.Context, content govtypes.Content) error {
		if content == nil {
			return sdkerrors.Wrap(sdkerrors.ErrUnknownRequest, "content must not be empty")
		}
		if _, ok := enabledTypes[content.ProposalType()]; !ok {
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unsupported wasm proposal content type: %q", content.ProposalType())
		}
		switch c := content.(type) {
		case *types.StoreCodeProposal:
			return handleStoreCodeProposal(ctx, k, *c)
		case *types.InstantiateContractProposal:
			return handleInstantiateProposal(ctx, k, *c)
		case *types.MigrateContractProposal:
			return handleMigrateProposal(ctx, k, *c)
		case *types.UpdateAdminContractProposal:
			return handleUpdateAdminProposal(ctx, k, *c)
		case *types.ClearAdminContractProposal:
			return handleClearAdminProposal(ctx, k, *c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized wasm proposal content type: %T", c)
		}
	}
}

func handleStoreCodeProposal(ctx sdk.Context, k Keeper, p types.StoreCodeProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	codeID, err := k.Create(ctx, p.Creator, p.WASMByteCode, p.Source, p.Builder)
	if err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		//sdk.NewAttribute(AttributeSigner, p.Creator.String()), // todo: creator is not signer. rename attribute?
		sdk.NewAttribute(AttributeKeyCodeID, fmt.Sprintf("%d", codeID)),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handleInstantiateProposal(ctx sdk.Context, k Keeper, p types.InstantiateContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	contractAddr, err := k.Instantiate(ctx, p.Code, p.Creator, p.Admin, p.InitMsg, p.Label, p.InitFunds)
	if err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		//sdk.NewAttribute(AttributeSigner, p.Creator.String()),
		sdk.NewAttribute(AttributeKeyCodeID, fmt.Sprintf("%d", p.Code)),
		sdk.NewAttribute(AttributeKeyContract, contractAddr.String()),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handleMigrateProposal(ctx sdk.Context, k Keeper, p types.MigrateContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	var caller sdk.AccAddress
	res, err := k.migrate(ctx, p.Contract, caller, p.Code, p.MigrateMsg, GovAuthorizationPolicy{})
	if err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		//sdk.NewAttribute(AttributeSigner, p.Creator.String()),
		sdk.NewAttribute(AttributeKeyContract, p.Contract.String()),
	)
	ctx.EventManager().EmitEvents(append(res.Events, ourEvent))
	return nil
}

func handleUpdateAdminProposal(ctx sdk.Context, k Keeper, p types.UpdateAdminContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	var caller sdk.AccAddress
	if err := k.setContractAdmin(ctx, p.Contract, caller, p.NewAdmin, GovAuthorizationPolicy{}); err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		//sdk.NewAttribute(AttributeSigner, p.Creator.String()),
		sdk.NewAttribute(AttributeKeyContract, p.Contract.String()),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handleClearAdminProposal(ctx sdk.Context, k Keeper, p types.ClearAdminContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	var caller sdk.AccAddress
	if err := k.setContractAdmin(ctx, p.Contract, caller, nil, GovAuthorizationPolicy{}); err != nil {
		return err
	}
	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		//sdk.NewAttribute(AttributeSigner, p.Creator.String()),
		sdk.NewAttribute(AttributeKeyContract, p.Contract.String()),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}
