package testdata

import (
	_ "embed"
)

//go:embed reflect.wasm
var reflectContract []byte

func ReflectContractWasm() []byte {
	return reflectContract
}
