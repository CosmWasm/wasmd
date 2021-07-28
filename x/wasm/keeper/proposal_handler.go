package keeper

import (
	"encoding/hex"
	"fmt"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
	"strconv"
)

// NewWasmProposalHandler creates a new governance Handler for wasm proposals
func NewWasmProposalHandler(k decoratedKeeper, enabledProposalTypes []types.ProposalType) govtypes.Handler {
	return NewWasmProposalHandlerX(NewGovPermissionKeeper(k), enabledProposalTypes)
}

// NewWasmProposalHandlerX creates a new governance Handler for wasm proposals
func NewWasmProposalHandlerX(k types.ContractOpsKeeper, enabledProposalTypes []types.ProposalType) govtypes.Handler {
	enabledTypes := make(map[string]struct{}, len(enabledProposalTypes))
	for i := range enabledProposalTypes {
		enabledTypes[string(enabledProposalTypes[i])] = struct{}{}
	}
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
		case *types.UpdateAdminProposal:
			return handleUpdateAdminProposal(ctx, k, *c)
		case *types.ClearAdminProposal:
			return handleClearAdminProposal(ctx, k, *c)
		case *types.PinCodesProposal:
			return handlePinCodesProposal(ctx, k, *c)
		case *types.UnpinCodesProposal:
			return handleUnpinCodesProposal(ctx, k, *c)
		default:
			return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized wasm proposal content type: %T", c)
		}
	}
}

func handleStoreCodeProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.StoreCodeProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	runAsAddr, err := sdk.AccAddressFromBech32(p.RunAs)
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}
	codeID, err := k.Create(ctx, runAsAddr, p.WASMByteCode, p.InstantiatePermission)
	if err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyCodeID, fmt.Sprintf("%d", codeID)),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handleInstantiateProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.InstantiateContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	runAsAddr, err := sdk.AccAddressFromBech32(p.RunAs)
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}
	adminAddr, err := sdk.AccAddressFromBech32(p.Admin)
	if err != nil {
		return sdkerrors.Wrap(err, "admin")
	}

	contractAddr, data, err := k.Instantiate(ctx, p.CodeID, runAsAddr, adminAddr, p.Msg, p.Label, p.Funds)
	if err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyCodeID, fmt.Sprintf("%d", p.CodeID)),
		sdk.NewAttribute(types.AttributeKeyContractAddr, contractAddr.String()),
		sdk.NewAttribute(types.AttributeResultDataHex, hex.EncodeToString(data)),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handleMigrateProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.MigrateContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	contractAddr, err := sdk.AccAddressFromBech32(p.Contract)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	runAsAddr, err := sdk.AccAddressFromBech32(p.RunAs)
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}
	data, err := k.Migrate(ctx, contractAddr, runAsAddr, p.CodeID, p.Msg)
	if err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyCodeID, fmt.Sprintf("%d", p.CodeID)),
		sdk.NewAttribute(types.AttributeKeyContractAddr, p.Contract),
		sdk.NewAttribute(types.AttributeResultDataHex, hex.EncodeToString(data)),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handleUpdateAdminProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.UpdateAdminProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	contractAddr, err := sdk.AccAddressFromBech32(p.Contract)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	newAdminAddr, err := sdk.AccAddressFromBech32(p.NewAdmin)
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}

	if err := k.UpdateContractAdmin(ctx, contractAddr, nil, newAdminAddr); err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyContractAddr, p.Contract),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handleClearAdminProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.ClearAdminProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	contractAddr, err := sdk.AccAddressFromBech32(p.Contract)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err := k.ClearContractAdmin(ctx, contractAddr, nil); err != nil {
		return err
	}
	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyContractAddr, p.Contract),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handlePinCodesProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.PinCodesProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	for _, v := range p.CodeIDs {
		if err := k.PinCode(ctx, v); err != nil {
			return sdkerrors.Wrapf(err, "code id: %d", v)
		}
	}
	for _, v := range p.CodeIDs {
		ourEvent := sdk.NewEvent(
			types.EventTypePinCode,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(types.AttributeKeyCodeID, strconv.FormatUint(v, 10)),
		)
		ctx.EventManager().EmitEvent(ourEvent)
	}
	return nil
}

func handleUnpinCodesProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.UnpinCodesProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	for _, v := range p.CodeIDs {
		if err := k.UnpinCode(ctx, v); err != nil {
			return sdkerrors.Wrapf(err, "code id: %d", v)
		}
	}
	for _, v := range p.CodeIDs {
		ourEvent := sdk.NewEvent(
			types.EventTypeUnpinCode,
			sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
			sdk.NewAttribute(types.AttributeKeyCodeID, strconv.FormatUint(v, 10)),
		)
		ctx.EventManager().EmitEvent(ourEvent)
	}
	return nil
}
