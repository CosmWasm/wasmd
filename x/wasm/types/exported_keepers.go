package types

import (
	types2 "github.com/CosmWasm/wasmvm/types"
	"github.com/cosmos/cosmos-sdk/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitytypes "github.com/cosmos/cosmos-sdk/x/capability/types"
)

// ViewKeeper provides read only operations
type ViewKeeper interface {
	GetContractHistory(ctx types.Context, contractAddr types.AccAddress) []ContractCodeHistoryEntry
	QuerySmart(ctx types.Context, contractAddr types.AccAddress, req []byte) ([]byte, error)
	QueryRaw(ctx types.Context, contractAddress types.AccAddress, key []byte) []byte
	GetContractInfo(ctx types.Context, contractAddress types.AccAddress) *ContractInfo
	IterateContractInfo(ctx types.Context, cb func(types.AccAddress, ContractInfo) bool)
	GetContractState(ctx types.Context, contractAddress types.AccAddress) types.Iterator
	GetCodeInfo(ctx types.Context, codeID uint64) *CodeInfo
	IterateCodeInfos(ctx types.Context, cb func(uint64, CodeInfo) bool)
	GetByteCode(ctx types.Context, codeID uint64) ([]byte, error)
	IsPinnedCode(ctx types.Context, codeID uint64) bool
	HasContractInfo(ctx sdk.Context, contractAddress sdk.AccAddress) bool
}

// MsgOpsKeeper contains mutable operations that are triggered by messages normally
type MsgOpsKeeper interface {
	Create(ctx sdk.Context, creator sdk.AccAddress, wasmCode []byte, source string, builder string, instantiateAccess *AccessConfig) (codeID uint64, err error)
	Instantiate(ctx sdk.Context, codeID uint64, creator, admin sdk.AccAddress, initMsg []byte, label string, deposit sdk.Coins) (sdk.AccAddress, []byte, error)
	Execute(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) (*sdk.Result, error)
	Migrate(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, newCodeID uint64, msg []byte) (*sdk.Result, error)
	UpdateContractAdmin(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, newAdmin sdk.AccAddress) error

	// ClearContractAdmin sets the admin value on the ContractInfo to nil, to disable further migrations/ updates.
	ClearContractAdmin(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress) error
	PinCode(ctx sdk.Context, codeID uint64) error
	UnpinCode(ctx sdk.Context, codeID uint64) error
}

// IBCContractKeeper IBC lifecycle event handler
type IBCContractKeeper interface {
	OnOpenChannel(
		ctx sdk.Context,
		contractAddr sdk.AccAddress,
		channel types2.IBCChannel,
	) error
	OnConnectChannel(
		ctx sdk.Context,
		contractAddr sdk.AccAddress,
		channel types2.IBCChannel,
	) error
	OnCloseChannel(
		ctx sdk.Context,
		contractAddr sdk.AccAddress,
		channel types2.IBCChannel,
	) error
	OnRecvPacket(
		ctx sdk.Context,
		contractAddr sdk.AccAddress,
		packet types2.IBCPacket,
	) ([]byte, error)
	OnAckPacket(
		ctx sdk.Context,
		contractAddr sdk.AccAddress,
		acknowledgement types2.IBCAcknowledgement,
	) error
	OnTimeoutPacket(
		ctx sdk.Context,
		contractAddr sdk.AccAddress,
		packet types2.IBCPacket,
	) error
	// ClaimCapability allows the transfer module to claim a capability
	//that IBC module passes to it
	ClaimCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) error
	// AuthenticateCapability wraps the scopedKeeper's AuthenticateCapability function
	AuthenticateCapability(ctx sdk.Context, cap *capabilitytypes.Capability, name string) bool
}
