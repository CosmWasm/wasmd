package app

import (
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/std"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

func MakeCodecs() (*std.Codec, *codec.Codec) {
	cdc := std.MakeCodec(ModuleBasics)
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	sdk.RegisterInterfaces(interfaceRegistry)
	ModuleBasics.RegisterInterfaceModules(interfaceRegistry)
	appCodec := std.NewAppCodec(cdc, interfaceRegistry)
	return appCodec, cdc
}
