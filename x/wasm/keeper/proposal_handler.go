package keeper

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/CosmWasm/wasmd/x/wasm/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	govtypes "github.com/cosmos/cosmos-sdk/x/gov/types"
)

// governing contains a subset of the wasm keeper used by gov processes
type governing interface {
	create(ctx sdk.Context, creator sdk.AccAddress, wasmCode []byte, source string, builder string, instantiateAccess *types.AccessConfig, authZ AuthorizationPolicy) (codeID uint64, err error)
	instantiate(ctx sdk.Context, codeID uint64, creator, admin sdk.AccAddress, initMsg []byte, label string, deposit sdk.Coins, authZ AuthorizationPolicy) (sdk.AccAddress, []byte, error)
	migrate(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, newCodeID uint64, msg []byte, authZ AuthorizationPolicy) (*sdk.Result, error)
	setContractAdmin(ctx sdk.Context, contractAddress, caller, newAdmin sdk.AccAddress, authZ AuthorizationPolicy) error
	PinCode(ctx sdk.Context, codeID uint64) error
	UnpinCode(ctx sdk.Context, codeID uint64) error
}

// NewWasmProposalHandler creates a new governance Handler for wasm proposals
func NewWasmProposalHandler(k governing, enabledProposalTypes []types.ProposalType) govtypes.Handler {
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

func handleStoreCodeProposal(ctx sdk.Context, k governing, p types.StoreCodeProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	runAsAddr, err := sdk.AccAddressFromBech32(p.RunAs)
	if err != nil {
		return sdkerrors.Wrap(err, "run as address")
	}
	codeID, err := k.create(ctx, runAsAddr, p.WASMByteCode, p.Source, p.Builder, p.InstantiatePermission, GovAuthorizationPolicy{})
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

func handleInstantiateProposal(ctx sdk.Context, k governing, p types.InstantiateContractProposal) error {
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

	contractAddr, _, err := k.instantiate(ctx, p.CodeID, runAsAddr, adminAddr, p.InitMsg, p.Label, p.Funds, GovAuthorizationPolicy{})
	if err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyCodeID, fmt.Sprintf("%d", p.CodeID)),
		sdk.NewAttribute(types.AttributeKeyContract, contractAddr.String()),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handleMigrateProposal(ctx sdk.Context, k governing, p types.MigrateContractProposal) error {
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
	res, err := k.migrate(ctx, contractAddr, runAsAddr, p.CodeID, p.MigrateMsg, GovAuthorizationPolicy{})
	if err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyContract, p.Contract),
	)
	ctx.EventManager().EmitEvent(ourEvent)

	for _, e := range res.Events {
		attr := make([]sdk.Attribute, len(e.Attributes))
		for i, a := range e.Attributes {
			attr[i] = sdk.NewAttribute(string(a.Key), string(a.Value))
		}
		ctx.EventManager().EmitEvent(sdk.NewEvent(e.Type, attr...))
	}
	return nil
}

func handleUpdateAdminProposal(ctx sdk.Context, k governing, p types.UpdateAdminProposal) error {
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

	if err := k.setContractAdmin(ctx, contractAddr, nil, newAdminAddr, GovAuthorizationPolicy{}); err != nil {
		return err
	}

	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyContract, p.Contract),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handleClearAdminProposal(ctx sdk.Context, k governing, p types.ClearAdminProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	contractAddr, err := sdk.AccAddressFromBech32(p.Contract)
	if err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	if err := k.setContractAdmin(ctx, contractAddr, nil, nil, GovAuthorizationPolicy{}); err != nil {
		return err
	}
	ourEvent := sdk.NewEvent(
		sdk.EventTypeMessage,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyContract, p.Contract),
	)
	ctx.EventManager().EmitEvent(ourEvent)
	return nil
}

func handlePinCodesProposal(ctx sdk.Context, k governing, p types.PinCodesProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	for _, v := range p.CodeIDs {
		if err := k.PinCode(ctx, v); err != nil {
			return sdkerrors.Wrapf(err, "code id: %d", v)
		}
	}
	s := make([]string, len(p.CodeIDs))
	for i, v := range p.CodeIDs {
		s[i] = strconv.FormatUint(v, 10)
	}
	ourEvent := sdk.NewEvent(
		types.EventTypePinCode,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyCodeIDs, strings.Join(s, ",")),
	)
	ctx.EventManager().EmitEvent(ourEvent)

	return nil
}

func handleUnpinCodesProposal(ctx sdk.Context, k governing, p types.UnpinCodesProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}
	for _, v := range p.CodeIDs {
		if err := k.UnpinCode(ctx, v); err != nil {
			return sdkerrors.Wrapf(err, "code id: %d", v)
		}
	}
	s := make([]string, len(p.CodeIDs))
	for i, v := range p.CodeIDs {
		s[i] = strconv.FormatUint(v, 10)
	}
	ourEvent := sdk.NewEvent(
		types.EventTypeUnpinCode,
		sdk.NewAttribute(sdk.AttributeKeyModule, types.ModuleName),
		sdk.NewAttribute(types.AttributeKeyCodeIDs, strings.Join(s, ",")),
	)
	ctx.EventManager().EmitEvent(ourEvent)

	return nil
}
