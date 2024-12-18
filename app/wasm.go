package app

// AllCapabilities returns all capabilities available with the current wasmvm
// See https://github.com/CosmWasm/cosmwasm/blob/main/docs/CAPABILITIES-BUILT-IN.md
// This functionality is going to be moved upstream: https://github.com/CosmWasm/wasmvm/issues/425
func AllCapabilities() []string {
	// This branch contains the 0.33.x version line. It was updated from wasmvm v1.3.0 to v1.5.x
	// to be able to provide security updates, but the new features introduced in between were not
	// implemented here in x/wasm. Therefore this deliberately does not include "cosmwasm_1_4",
	// so contracts requiring those features cannot be uploaded.
	return []string{
		"iterator",
		"staking",
		"stargate",
		"cosmwasm_1_1",
		"cosmwasm_1_2",
		"cosmwasm_1_3",
	}
}
