package keeper

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	wasmkeeper "github.com/Finschia/wasmd/x/wasm/keeper"
)

func mustMarshal(t *testing.T, r interface{}) []byte {
	t.Helper()
	bz, err := json.Marshal(r)
	require.NoError(t, err)
	return bz
}

func TestInactivateContract(t *testing.T) {
	parentCtx, keepers := CreateTestInput(t, false, AvailableCapabilities)
	example := InstantiateHackatomExampleContract(t, parentCtx, keepers)
	otherContract := StoreHackatomExampleContract(t, parentCtx, keepers)
	newVerifier := RandomAccountAddress(t)
	newAdmin := RandomAccountAddress(t)
	migrateMsg := []byte(fmt.Sprintf("{\"verifier\":\"%s\"}", newVerifier.String()))

	contractKeeper := NewPermissionedKeeper(*wasmkeeper.NewDefaultPermissionKeeper(keepers.WasmKeeper), keepers.WasmKeeper)

	var err error
	// deactivate state
	{
		// check execute
		_, err := contractKeeper.Execute(parentCtx, example.Contract, example.VerifierAddr, []byte(`{"release":{}}`), nil)
		require.NoError(t, err)

		// check Migrate
		_, err = contractKeeper.Migrate(parentCtx, example.Contract, example.CreatorAddr, otherContract.CodeID, migrateMsg)
		require.NoError(t, err)

		// check update contract admin
		err = contractKeeper.UpdateContractAdmin(parentCtx, example.Contract, example.CreatorAddr, newAdmin)
		require.NoError(t, err)

		// check clear contract admin
		err = contractKeeper.ClearContractAdmin(parentCtx, example.Contract, newAdmin)
		require.NoError(t, err)
	}

	// set deactivate
	err = contractKeeper.DeactivateContract(parentCtx, example.Contract)
	require.NoError(t, err)

	// deactivate state
	{
		// check execute
		_, err = contractKeeper.Execute(parentCtx, example.Contract, newVerifier, []byte(`{"release":{}}`), nil)
		require.Error(t, err)

		// check migrate
		_, err = contractKeeper.Migrate(parentCtx, example.Contract, example.CreatorAddr, otherContract.CodeID, migrateMsg)
		require.Error(t, err)

		// check update contract admin
		err = contractKeeper.UpdateContractAdmin(parentCtx, example.Contract, example.CreatorAddr, newAdmin)
		require.Error(t, err)

		// check clear contract admin
		err = contractKeeper.ClearContractAdmin(parentCtx, example.Contract, newAdmin)
		require.Error(t, err)
	}

	// set activate
	err = contractKeeper.ActivateContract(parentCtx, example.Contract)
	require.NoError(t, err)

	// activate state
	{
		// check execute
		_, err = contractKeeper.Execute(parentCtx, example.Contract, newVerifier, []byte(`{"release":{}}`), nil)
		require.NoError(t, err)
	}
}
