package app

import (
	"github.com/CosmWasm/wasmd/x/wasm"
	"github.com/cosmos/cosmos-sdk/simapp"
	simparams "github.com/cosmos/cosmos-sdk/simapp/params"
)

func MakeEncoding() simparams.EncodingConfig {
	encodingConfig := simapp.MakeEncodingConfig()
	wasm.RegisterCodec(encodingConfig.Amino)
	wasm.RegisterInterfaces(encodingConfig.InterfaceRegistry)
	return encodingConfig
}
