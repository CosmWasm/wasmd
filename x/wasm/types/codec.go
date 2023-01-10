package types

import (
	"github.com/line/lbm-sdk/codec"
	"github.com/line/lbm-sdk/codec/legacy"
	"github.com/line/lbm-sdk/codec/types"
	cryptocodec "github.com/line/lbm-sdk/crypto/codec"
	sdk "github.com/line/lbm-sdk/types"
	"github.com/line/lbm-sdk/types/msgservice"
	authzcodec "github.com/line/lbm-sdk/x/authz/codec"
	govcodec "github.com/line/lbm-sdk/x/gov/codec"
	govtypes "github.com/line/lbm-sdk/x/gov/types"
)

// RegisterLegacyAminoCodec registers the account types and interface
func RegisterLegacyAminoCodec(cdc *codec.LegacyAmino) { //nolint:staticcheck
	legacy.RegisterAminoMsg(cdc, &MsgStoreCode{}, "wasm/MsgStoreCode")
	legacy.RegisterAminoMsg(cdc, &MsgInstantiateContract{}, "wasm/MsgInstantiateContract")
	legacy.RegisterAminoMsg(cdc, &MsgExecuteContract{}, "wasm/MsgExecuteContract")
	legacy.RegisterAminoMsg(cdc, &MsgMigrateContract{}, "wasm/MsgMigrateContract")
	legacy.RegisterAminoMsg(cdc, &MsgUpdateAdmin{}, "wasm/MsgUpdateAdmin")
	legacy.RegisterAminoMsg(cdc, &MsgClearAdmin{}, "wasm/MsgClearAdmin")
	legacy.RegisterAminoMsg(cdc, &MsgIBCSend{}, "wasm/MsgIBCSend")
	legacy.RegisterAminoMsg(cdc, &MsgIBCCloseChannel{}, "wasm/MsgIBCCloseChannel")

	cdc.RegisterConcrete(&PinCodesProposal{}, "wasm/PinCodesProposal", nil)
	cdc.RegisterConcrete(&UnpinCodesProposal{}, "wasm/UnpinCodesProposal", nil)
	cdc.RegisterConcrete(&StoreCodeProposal{}, "wasm/StoreCodeProposal", nil)
	cdc.RegisterConcrete(&InstantiateContractProposal{}, "wasm/InstantiateContractProposal", nil)
	cdc.RegisterConcrete(&MigrateContractProposal{}, "wasm/MigrateContractProposal", nil)
	cdc.RegisterConcrete(&SudoContractProposal{}, "wasm/SudoContractProposal", nil)
	cdc.RegisterConcrete(&ExecuteContractProposal{}, "wasm/ExecuteContractProposal", nil)
	cdc.RegisterConcrete(&UpdateAdminProposal{}, "wasm/UpdateAdminProposal", nil)
	cdc.RegisterConcrete(&ClearAdminProposal{}, "wasm/ClearAdminProposal", nil)
	cdc.RegisterConcrete(&UpdateInstantiateConfigProposal{}, "wasm/UpdateInstantiateConfigProposal", nil)
}

func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgStoreCode{},
		&MsgInstantiateContract{},
		&MsgExecuteContract{},
		&MsgMigrateContract{},
		&MsgUpdateAdmin{},
		&MsgClearAdmin{},
		&MsgIBCCloseChannel{},
		&MsgIBCSend{},
	)
	registry.RegisterImplementations(
		(*govtypes.Content)(nil),
		&StoreCodeProposal{},
		&InstantiateContractProposal{},
		&MigrateContractProposal{},
		&SudoContractProposal{},
		&ExecuteContractProposal{},
		&UpdateAdminProposal{},
		&ClearAdminProposal{},
		&PinCodesProposal{},
		&UnpinCodesProposal{},
		&UpdateInstantiateConfigProposal{},
	)

	registry.RegisterInterface("ContractInfoExtension", (*ContractInfoExtension)(nil))

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)
}

var (
	amino     = codec.NewLegacyAmino()
	ModuleCdc = codec.NewAminoCodec(amino)
)

func init() {
	RegisterLegacyAminoCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	sdk.RegisterLegacyAminoCodec(amino)

	// Register all Amino interfaces and concrete types on the authz Amino codec so that this can later be
	// used to properly serialize MsgGrant, MsgExec and MsgSubmitProposal instances
	RegisterLegacyAminoCodec(authzcodec.Amino)
	RegisterLegacyAminoCodec(govcodec.Amino)
}
