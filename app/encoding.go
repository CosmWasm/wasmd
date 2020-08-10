package app

import (
	simparams "github.com/cosmos/cosmos-sdk/simapp/params"
	"github.com/cosmos/cosmos-sdk/std"
)

func MakeEncodingConfig() simparams.EncodingConfig {
	encodingConfig := simparams.MakeEncodingConfig() // todo: this is the simapp !!!
	std.RegisterCodec(encodingConfig.Amino)
	std.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	ModuleBasics.RegisterCodec(encodingConfig.Amino)
	ModuleBasics.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
