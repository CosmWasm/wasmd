package keeper

import (
	sdk "github.com/Finschia/finschia-sdk/types"
	sdkerrors "github.com/Finschia/finschia-sdk/types/errors"
	govtypes "github.com/Finschia/finschia-sdk/x/gov/types"

	wasmkeeper "github.com/Finschia/wasmd/x/wasm/keeper"
	wasmtypes "github.com/Finschia/wasmd/x/wasm/types"
	"github.com/Finschia/wasmd/x/wasmplus/types"
)

// NewWasmProposalHandler creates a new governance Handler for wasm proposals
func NewWasmProposalHandler(k *Keeper, enabledProposalType []wasmtypes.ProposalType) govtypes.Handler {
	govPerm := wasmkeeper.NewGovPermissionKeeper(k)
	return NewWasmProposalHandlerX(NewPermissionedKeeper(*govPerm, k), enabledProposalType)
}

// NewWasmProposalHandlerX creates a new governance Handler for wasm proposals
func NewWasmProposalHandlerX(k types.ContractOpsKeeper, enabledProposalTypes []wasmtypes.ProposalType) govtypes.Handler {
	handler := wasmkeeper.NewWasmProposalHandlerX(k, enabledProposalTypes)
	enabledTypes := make(map[string]struct{}, len(enabledProposalTypes))
	for i := range enabledProposalTypes {
		enabledTypes[string(enabledProposalTypes[i])] = struct{}{}
	}
	return func(ctx sdk.Context, content govtypes.Content) error {
		err := handler(ctx, content)
		if err != nil {
			if _, ok := enabledTypes[content.ProposalType()]; !ok {
				return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unsupported wasm proposal content type: %q", content.ProposalType())
			}
			switch c := content.(type) {
			case *types.DeactivateContractProposal:
				return handleDeactivateContractProposal(ctx, k, *c)
			case *types.ActivateContractProposal:
				return handleActivateContractProposal(ctx, k, *c)
			default:
				return sdkerrors.Wrapf(sdkerrors.ErrUnknownRequest, "unrecognized wasm proposal content type: %T", c)
			}
		}
		return nil
	}
}

func handleDeactivateContractProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.DeactivateContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	// The error is already checked in ValidateBasic.
	//nolint:errcheck
	contractAddr, _ := sdk.AccAddressFromBech32(p.Contract)

	err := k.DeactivateContract(ctx, contractAddr)
	if err != nil {
		return err
	}

	event := types.EventDeactivateContractProposal{
		Contract: contractAddr.String(),
	}
	if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
		return err
	}

	return nil
}

func handleActivateContractProposal(ctx sdk.Context, k types.ContractOpsKeeper, p types.ActivateContractProposal) error {
	if err := p.ValidateBasic(); err != nil {
		return err
	}

	// The error is already checked in ValidateBasic.
	//nolint:errcheck
	contractAddr, _ := sdk.AccAddressFromBech32(p.Contract)

	err := k.ActivateContract(ctx, contractAddr)
	if err != nil {
		return err
	}

	event := types.EventActivateContractProposal{Contract: contractAddr.String()}
	if err := ctx.EventManager().EmitTypedEvent(&event); err != nil {
		return nil
	}

	return nil
}
