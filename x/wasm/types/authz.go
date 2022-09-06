package types

import (
	"strings"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	authztypes "github.com/cosmos/cosmos-sdk/x/authz"
)

const gasDeserializationCostPerByte = uint64(1)

var _ authztypes.Authorization = &ContractExecutionAuthorization{}

// NewContractAuthorization creates a new ContractExecutionAuthorization object.
func NewContractAuthorization(contractAddr sdk.AccAddress, filter isContractExecutionAuthorization_ContractExecutionGrant_Filter, max isContractExecutionAuthorization_ContractExecutionGrant_MaxExecutions) *ContractExecutionAuthorization {
	return &ContractExecutionAuthorization{
		Grants: []ContractExecutionAuthorization_ContractExecutionGrant{
			{
				Contract:      contractAddr.String(),
				MaxExecutions: max,
				Filter:        filter,
			},
		},
	}
}

// MsgTypeURL implements Authorization.MsgTypeURL.
func (a ContractExecutionAuthorization) MsgTypeURL() string {
	return sdk.MsgTypeURL(&MsgExecuteContract{})
}

// Accept implements Authorization.Accept.
func (a *ContractExecutionAuthorization) Accept(ctx sdk.Context, msg sdk.Msg) (authztypes.AcceptResponse, error) {
	exec, ok := msg.(*MsgExecuteContract)
	if !ok {
		return authztypes.AcceptResponse{}, sdkerrors.ErrInvalidType.Wrap("type mismatch")
	}
	if exec == nil || exec.Msg == nil {
		return authztypes.AcceptResponse{}, sdkerrors.ErrInvalidType.Wrap("empty message")
	}

	updatedGrants, modified, err := a.applyGrants(ctx, exec)
	switch {
	case err != nil:
		return authztypes.AcceptResponse{}, err
	case len(updatedGrants) == 0:
		return authztypes.AcceptResponse{Accept: true, Delete: true}, nil
	case modified:
		return authztypes.AcceptResponse{Accept: true, Updated: &ContractExecutionAuthorization{
			Grants: updatedGrants,
		}}, nil
	default:
		return authztypes.AcceptResponse{Accept: true}, nil
	}
}

func (a ContractExecutionAuthorization) applyGrants(ctx sdk.Context, exec *MsgExecuteContract) ([]ContractExecutionAuthorization_ContractExecutionGrant, bool, error) {
	for i, g := range a.Grants {
		if g.Contract != exec.Contract {
			continue
		}
		// first check execution counter
		var modified bool
		var newGrants []ContractExecutionAuthorization_ContractExecutionGrant
		switch {
		case g.GetInfiniteCalls() != nil:
			newGrants = a.GetGrants()
		case g.GetMaxCalls() != nil:
			switch n := g.GetMaxCalls().Remaining; n {
			case 0:
				return nil, false, sdkerrors.ErrUnauthorized.Wrap("no executions allowed")
			case 1: // remove
				newGrants = append(a.Grants[0:i], a.Grants[i+1:]...)
				modified = true
			default:
				obj := ContractExecutionAuthorization_ContractExecutionGrant{
					Contract: g.Contract,
					MaxExecutions: &ContractExecutionAuthorization_ContractExecutionGrant_MaxCalls{
						MaxCalls: &MaxCalls{Remaining: n - 1},
					},
					Filter: g.Filter,
				}
				newGrants = append(append(a.Grants[0:i], obj), a.Grants[i+1:]...)
				modified = true
			}
		default:
			return nil, false, sdkerrors.ErrUnauthorized.Wrapf("unsupported max execution type: %T", g.MaxExecutions)
		}
		// then check permission set
		switch {
		case g.GetAllowAllWildcard() != nil:
		case g.GetAcceptedMessageKeys().GetMessages() != nil:
			gasForDeserialization := gasDeserializationCostPerByte * uint64(len(exec.Msg))
			ctx.GasMeter().ConsumeGas(gasForDeserialization, "contract authorization")

			if err := IsJSONObjectWithTopLevelKey(exec.Msg, g.GetAcceptedMessageKeys().GetMessages()); err != nil {
				return nil, false, sdkerrors.ErrUnauthorized.Wrapf("not an allowed msg: %s", err.Error())
			}
		default:
			return nil, false, sdkerrors.ErrUnauthorized.Wrapf("unsupported filter type: %T", g.Filter)
		}
		return newGrants, modified, nil
	}
	return nil, false, sdkerrors.ErrUnauthorized.Wrap("contract address")
}

// ValidateBasic implements Authorization.ValidateBasic.
func (a ContractExecutionAuthorization) ValidateBasic() error {
	if len(a.Grants) == 0 {
		return ErrEmpty.Wrap("grants")
	}
	for i, v := range a.Grants {
		if err := v.ValidateBasic(); err != nil {
			return sdkerrors.Wrapf(err, "position %d", i)
		}
	}
	// todo (Alex): do we want to allow multiple grants for a contract or enforce a unique constraint
	// example: contractA:doThis:1 , applyGrants:doThat:*  has with different counters for different methods
	// example: contractA:doThis:1 , applyGrants:*:*  would not be great but works as well
	return nil
}

func (g ContractExecutionAuthorization_ContractExecutionGrant) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(g.Contract); err != nil {
		return sdkerrors.Wrap(err, "contract")
	}
	// execution counter
	switch {
	case g.GetInfiniteCalls() != nil:
	case g.GetMaxCalls() != nil:
		if g.GetMaxCalls().Remaining == 0 {
			return ErrEmpty.Wrap("remaining calls")
		}
	default:
		return ErrInvalid.Wrapf("unsupported max execution type: %T", g.MaxExecutions)
	}
	// filter
	switch {
	case g.GetAllowAllWildcard() != nil:
	case g.GetAcceptedMessageKeys().GetMessages() != nil:
		if len(g.GetAcceptedMessageKeys().GetMessages()) == 0 {
			return ErrEmpty.Wrap("messages")
		}
		idx := make(map[string]struct{}, len(g.GetAcceptedMessageKeys().GetMessages()))
		for i, m := range g.GetAcceptedMessageKeys().GetMessages() {
			if m == "" {
				return ErrEmpty.Wrap("message")
			}
			if m != strings.TrimSpace(m) { // TODO (Alex): does this make sense? We should not make assumptions about payload
				return ErrInvalid.Wrapf("message %d contains whitespaces", i)
			}
			if _, exists := idx[m]; exists {
				return ErrDuplicate.Wrap("message")
			}
			idx[m] = struct{}{}
		}
	default:
		return ErrInvalid.Wrapf("unsupported filter type: %T", g.Filter)
	}
	return nil
}
