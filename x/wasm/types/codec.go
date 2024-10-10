package types

import (
	"cosmossdk.io/core/registry"
	"cosmossdk.io/x/authz"
	"cosmossdk.io/x/gov/types/v1beta1"
	"github.com/cosmos/cosmos-sdk/codec/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/msgservice"
)

// RegisterLegacyAminoCodec registers the concrete types and interface
func RegisterLegacyAminoCodec(registrar registry.AminoRegistrar) {
	registrar.RegisterConcrete(&MsgStoreCode{}, "wasm/MsgStoreCode")
	registrar.RegisterConcrete(&MsgInstantiateContract{}, "wasm/MsgInstantiateContract")
	registrar.RegisterConcrete(&MsgInstantiateContract2{}, "wasm/MsgInstantiateContract2")
	registrar.RegisterConcrete(&MsgExecuteContract{}, "wasm/MsgExecuteContract")
	registrar.RegisterConcrete(&MsgMigrateContract{}, "wasm/MsgMigrateContract")
	registrar.RegisterConcrete(&MsgUpdateAdmin{}, "wasm/MsgUpdateAdmin")
	registrar.RegisterConcrete(&MsgClearAdmin{}, "wasm/MsgClearAdmin")
	registrar.RegisterConcrete(&MsgUpdateInstantiateConfig{}, "wasm/MsgUpdateInstantiateConfig")
	registrar.RegisterConcrete(&MsgUpdateParams{}, "wasm/MsgUpdateParams")
	registrar.RegisterConcrete(&MsgSudoContract{}, "wasm/MsgSudoContract")
	registrar.RegisterConcrete(&MsgPinCodes{}, "wasm/MsgPinCodes")
	registrar.RegisterConcrete(&MsgUnpinCodes{}, "wasm/MsgUnpinCodes")
	registrar.RegisterConcrete(&MsgStoreAndInstantiateContract{}, "wasm/MsgStoreAndInstantiateContract")
	registrar.RegisterConcrete(&MsgAddCodeUploadParamsAddresses{}, "wasm/MsgAddCodeUploadParamsAddresses")
	registrar.RegisterConcrete(&MsgRemoveCodeUploadParamsAddresses{}, "wasm/MsgRemoveCodeUploadParamsAddresses")
	registrar.RegisterConcrete(&MsgStoreAndMigrateContract{}, "wasm/MsgStoreAndMigrateContract")
	registrar.RegisterConcrete(&MsgUpdateContractLabel{}, "wasm/MsgUpdateContractLabel")

	registrar.RegisterInterface((*ContractInfoExtension)(nil), nil)

	registrar.RegisterInterface((*ContractAuthzFilterX)(nil), nil)
	registrar.RegisterConcrete(&AllowAllMessagesFilter{}, "wasm/AllowAllMessagesFilter")
	registrar.RegisterConcrete(&AcceptedMessageKeysFilter{}, "wasm/AcceptedMessageKeysFilter")
	registrar.RegisterConcrete(&AcceptedMessagesFilter{}, "wasm/AcceptedMessagesFilter")

	registrar.RegisterInterface((*ContractAuthzLimitX)(nil), nil)
	registrar.RegisterConcrete(&MaxCallsLimit{}, "wasm/MaxCallsLimit")
	registrar.RegisterConcrete(&MaxFundsLimit{}, "wasm/MaxFundsLimit")
	registrar.RegisterConcrete(&CombinedLimit{}, "wasm/CombinedLimit")

	registrar.RegisterConcrete(&StoreCodeAuthorization{}, "wasm/StoreCodeAuthorization")
	registrar.RegisterConcrete(&ContractExecutionAuthorization{}, "wasm/ContractExecutionAuthorization")
	registrar.RegisterConcrete(&ContractMigrationAuthorization{}, "wasm/ContractMigrationAuthorization")

	// legacy gov v1beta1 types that may be used for unmarshalling stored gov data
	registrar.RegisterConcrete(&PinCodesProposal{}, "wasm/PinCodesProposal")
	registrar.RegisterConcrete(&UnpinCodesProposal{}, "wasm/UnpinCodesProposal")
	registrar.RegisterConcrete(&StoreCodeProposal{}, "wasm/StoreCodeProposal")
	registrar.RegisterConcrete(&InstantiateContractProposal{}, "wasm/InstantiateContractProposal")
	registrar.RegisterConcrete(&InstantiateContract2Proposal{}, "wasm/InstantiateContract2Proposal")
	registrar.RegisterConcrete(&MigrateContractProposal{}, "wasm/MigrateContractProposal")
	registrar.RegisterConcrete(&SudoContractProposal{}, "wasm/SudoContractProposal")
	registrar.RegisterConcrete(&ExecuteContractProposal{}, "wasm/ExecuteContractProposal")
	registrar.RegisterConcrete(&UpdateAdminProposal{}, "wasm/UpdateAdminProposal")
	registrar.RegisterConcrete(&ClearAdminProposal{}, "wasm/ClearAdminProposal")
	registrar.RegisterConcrete(&UpdateInstantiateConfigProposal{}, "wasm/UpdateInstantiateConfigProposal")
	registrar.RegisterConcrete(&StoreAndInstantiateContractProposal{}, "wasm/StoreAndInstantiateContractProposal")
}

// RegisterInterfaces registers the concrete proto types and interfaces with the SDK interface registry
func RegisterInterfaces(registry types.InterfaceRegistry) {
	registry.RegisterImplementations(
		(*sdk.Msg)(nil),
		&MsgStoreCode{},
		&MsgInstantiateContract{},
		&MsgInstantiateContract2{},
		&MsgExecuteContract{},
		&MsgMigrateContract{},
		&MsgUpdateAdmin{},
		&MsgClearAdmin{},
		&MsgIBCCloseChannel{},
		&MsgIBCSend{},
		&MsgUpdateInstantiateConfig{},
		&MsgUpdateParams{},
		&MsgSudoContract{},
		&MsgPinCodes{},
		&MsgUnpinCodes{},
		&MsgStoreAndInstantiateContract{},
		&MsgAddCodeUploadParamsAddresses{},
		&MsgRemoveCodeUploadParamsAddresses{},
		&MsgStoreAndMigrateContract{},
		&MsgUpdateContractLabel{},
	)
	registry.RegisterInterface("cosmwasm.wasm.v1.ContractInfoExtension", (*ContractInfoExtension)(nil))

	registry.RegisterInterface("cosmwasm.wasm.v1.ContractAuthzFilterX", (*ContractAuthzFilterX)(nil))
	registry.RegisterImplementations(
		(*ContractAuthzFilterX)(nil),
		&AllowAllMessagesFilter{},
		&AcceptedMessageKeysFilter{},
		&AcceptedMessagesFilter{},
	)

	registry.RegisterInterface("cosmwasm.wasm.v1.ContractAuthzLimitX", (*ContractAuthzLimitX)(nil))
	registry.RegisterImplementations(
		(*ContractAuthzLimitX)(nil),
		&MaxCallsLimit{},
		&MaxFundsLimit{},
		&CombinedLimit{},
	)

	registry.RegisterImplementations(
		(*authz.Authorization)(nil),
		&StoreCodeAuthorization{},
		&ContractExecutionAuthorization{},
		&ContractMigrationAuthorization{},
	)

	msgservice.RegisterMsgServiceDesc(registry, &_Msg_serviceDesc)

	// legacy gov v1beta1 types that may be used for unmarshalling stored gov data
	registry.RegisterImplementations(
		(*v1beta1.Content)(nil),
		&StoreCodeProposal{},
		&InstantiateContractProposal{},
		&InstantiateContract2Proposal{},
		&MigrateContractProposal{},
		&SudoContractProposal{},
		&ExecuteContractProposal{},
		&UpdateAdminProposal{},
		&ClearAdminProposal{},
		&PinCodesProposal{},
		&UnpinCodesProposal{},
		&UpdateInstantiateConfigProposal{},
		&StoreAndInstantiateContractProposal{},
	)
}
