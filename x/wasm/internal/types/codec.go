package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
)

// RegisterCodec registers the account types and interface
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(&MsgStoreCode{}, "wasm/store-code", nil)
	cdc.RegisterConcrete(&MsgInstantiateContract{}, "wasm/instantiate", nil)
	cdc.RegisterConcrete(&MsgExecuteContract{}, "wasm/execute", nil)
	cdc.RegisterConcrete(&MsgMigrateContract{}, "wasm/migrate", nil)
	cdc.RegisterConcrete(&MsgUpdateAdmin{}, "wasm/update-contract-admin", nil)
	cdc.RegisterConcrete(&MsgClearAdmin{}, "wasm/clear-contract-admin", nil)

	cdc.RegisterConcrete(StoreCodeProposal{}, "wasm/store-proposal", nil)
	cdc.RegisterConcrete(InstantiateContractProposal{}, "wasm/instantiate-proposal", nil)
	cdc.RegisterConcrete(MigrateContractProposal{}, "wasm/migrate-proposal", nil)
	cdc.RegisterConcrete(UpdateAdminProposal{}, "wasm/update-admin-proposal", nil)
	cdc.RegisterConcrete(ClearAdminProposal{}, "wasm/clear-admin-proposal", nil)
}

// ModuleCdc generic sealed codec to be used throughout module
var ModuleCdc *codec.Codec

func init() {
	cdc := codec.New()
	RegisterCodec(cdc)
	codec.RegisterCrypto(cdc)
	ModuleCdc = cdc.Seal()
}
