package simulation

import (
	"fmt"
	"math/rand"

	"github.com/CosmWasm/wasmd/x/wasm/internal/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/cosmos/cosmos-sdk/x/simulation"
)

func ParamChanges(r *rand.Rand) []simtypes.ParamChange {
	return []simtypes.ParamChange{
		simulation.NewSimParamChange(types.ModuleName, string(types.ParamStoreKeyUploadAccess),
			func(r *rand.Rand) string {
				param := RandomizeAccessConfig(r)
				return fmt.Sprintf(`{"code_upload_access":"%s"}`, param.String())
			},
		),
		simulation.NewSimParamChange(types.ModuleName, string(types.ParamStoreKeyInstantiateAccess),
			func(r *rand.Rand) string {
				param := RandomizeAccessType(r)
				return fmt.Sprintf(`{"instantiate_default_permission":"%s"}`, param.String())
			},
		),
		simulation.NewSimParamChange(types.ModuleName, string(types.ParamStoreKeyMaxWasmCodeSize),
			func(r *rand.Rand) string {
				param := RandomizeMaxWasmCodeSize(r)
				return fmt.Sprintf(`{"max_wasm_code_size":"%d"`, param)
			},
		),
	}
}

func RandomizeAccessType(r *rand.Rand) types.AccessType {
	return types.AccessType(simtypes.RandIntBetween(r, 1, 3))
}

func RandomizeAccessConfig(r *rand.Rand) types.AccessConfig {
	permissionType := RandomizeAccessType(r)
	address := ""
	if permissionType == types.AccessTypeOnlyAddress {
		account, _ := simtypes.RandomAcc(r, simtypes.RandomAccounts(r, 10))
		address = account.Address.String()
	}
	return types.AccessConfig{
		Permission: permissionType,
		Address:    address,
	}
}

func RandomizeMaxWasmCodeSize(r *rand.Rand) uint64 {
	return uint64(simtypes.RandIntBetween(r, 1, 600) * 1024)
}
