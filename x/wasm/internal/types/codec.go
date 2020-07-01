package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// RegisterCodec registers the account types and interface
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(&MsgStoreCode{}, "wasm/store-code", nil)
	cdc.RegisterConcrete(&MsgInstantiateContract{}, "wasm/instantiate", nil)
	cdc.RegisterConcrete(&MsgExecuteContract{}, "wasm/execute", nil)
	cdc.RegisterConcrete(&MsgMigrateContract{}, "wasm/migrate", nil)
	cdc.RegisterConcrete(&MsgUpdateAdmin{}, "wasm/update-contract-admin", nil)
	cdc.RegisterConcrete(&MsgClearAdmin{}, "wasm/clear-contract-admin", nil)
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
	)
}

var (
	amino = codec.New()

	// ModuleCdc references the global x/wasm module codec.

	ModuleCdc = codec.NewHybridCodec(amino, types.NewInterfaceRegistry())
)

func init() {
	RegisterCodec(amino)
	cryptocodec.RegisterCrypto(amino)
	amino.Seal()
}
