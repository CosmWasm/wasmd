package types

import (
	"fmt"

	sdk "github.com/line/lbm-sdk/types"
	sdkerrors "github.com/line/lbm-sdk/types/errors"
	govtypes "github.com/line/lbm-sdk/x/gov/types"

	wasmtypes "github.com/line/wasmd/x/wasm/types"
)

const (
	ProposalTypeDeactivateContract wasmtypes.ProposalType = "DeactivateContract"
	ProposalTypeActivateContract   wasmtypes.ProposalType = "ActivateContract"
)

var EnableAllProposals = append([]wasmtypes.ProposalType{
	ProposalTypeDeactivateContract,
	ProposalTypeActivateContract,
}, wasmtypes.EnableAllProposals...)

func init() {
	govtypes.RegisterProposalType(string(ProposalTypeDeactivateContract))
	govtypes.RegisterProposalType(string(ProposalTypeActivateContract))
}

func (p DeactivateContractProposal) GetTitle() string { return p.Title }

func (p DeactivateContractProposal) GetDescription() string { return p.Description }

func (p DeactivateContractProposal) ProposalRoute() string { return wasmtypes.RouterKey }

func (p DeactivateContractProposal) ProposalType() string {
	return string(ProposalTypeDeactivateContract)
}

func (p DeactivateContractProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(p.Contract); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "contract")
	}

	return nil
}

func (p DeactivateContractProposal) String() string {
	return fmt.Sprintf(`Deactivate Contract Proposal:
  Title:       %s
  Description: %s
  Contract:    %s
`, p.Title, p.Description, p.Contract)
}

func (p ActivateContractProposal) GetTitle() string { return p.Title }

func (p ActivateContractProposal) GetDescription() string { return p.Description }

func (p ActivateContractProposal) ProposalRoute() string { return wasmtypes.RouterKey }

func (p ActivateContractProposal) ProposalType() string {
	return string(ProposalTypeActivateContract)
}

func (p ActivateContractProposal) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(p.Contract); err != nil {
		return sdkerrors.Wrap(sdkerrors.ErrInvalidRequest, "contract")
	}

	return nil
}

func (p ActivateContractProposal) String() string {
	return fmt.Sprintf(`Activate Contract Proposal:
  Title:       %s
  Description: %s
  Contract:    %s
`, p.Title, p.Description, p.Contract)
}
