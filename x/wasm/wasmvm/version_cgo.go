package cosmwasm

import (
	"github.com/CosmWasm/wasmd/x/wasm/wasmvm/internal/api"
)

func libwasmvmVersionImpl() (string, error) {
	return api.LibwasmvmVersion()
}
