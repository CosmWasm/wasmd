package types

import (
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/codec/types"
	cryptocodec "github.com/cosmos/cosmos-sdk/crypto/codec"
	// "github.com/cosmos/cosmos-sdk/x/supply/exported"
)

// RegisterCodec registers the account types and interface
func RegisterCodec(cdc *codec.Codec) {
	cdc.RegisterConcrete(&MsgStoreCode{}, "wasm/store-code", nil)
	cdc.RegisterConcrete(&MsgInstantiateContract{}, "wasm/instantiate", nil)
	cdc.RegisterConcrete(&MsgExecuteContract{}, "wasm/execute", nil)
	cdc.RegisterConcrete(&MsgMigrateContract{}, "wasm/migrate", nil)
	cdc.RegisterConcrete(&MsgUpdateAdministrator{}, "wasm/update-contract-admin", nil)
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
