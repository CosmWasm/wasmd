package app

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/cosmos/cosmos-sdk/simapp"
	simparams "github.com/cosmos/cosmos-sdk/simapp/params"
)

func MakeEncodingConfig() simparams.EncodingConfig {
	encodingConfig := simapp.MakeEncodingConfig() // todo: this is the simapp !!!
	wasm.RegisterCodec(encodingConfig.Amino)
	wasm.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
