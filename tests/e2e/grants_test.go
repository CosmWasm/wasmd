package e2e_test

import (
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/authz"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/CosmWasm/wasmd/x/wasm/ibctesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestGrants(t *testing.T) {
	// given a contract by address A
	// and   grant for address B by A created
	// When  B sends an execute with tokens from A
	// then
	// - balance A reduced
	// - balance B not touched

	chain := ibctesting.NewCoordinator(t, 1).GetChain(ibctesting.GetChainID(0))
	codeID := chain.StoreCodeFile("../../x/wasm/keeper/testdata/reflect_1_1.wasm").CodeID
	contractAddr := chain.InstantiateContract(codeID, []byte(`{}`))
	require.NotEmpty(t, contractAddr)

	granteePrivKey := secp256k1.GenPrivKey()
	granteeAddr := granteePrivKey.PubKey().Bytes()
	chain.Fund(granteeAddr, sdk.NewInt(1_000_000))
	assert.Equal(t, sdk.NewInt(1_000_000), chain.Balance(granteeAddr, sdk.DefaultBondDenom).Amount)
	// setup grant
	grant, err := types.NewContractGrant(contractAddr, types.NewMaxCallsLimit(1), types.NewAllowAllMessagesFilter())
	require.NoError(t, err)
	a := types.NewContractExecutionAuthorization(*grant)
	grantMsg, err := authz.NewMsgGrant(chain.SenderAccount.GetAddress(), granteeAddr, a, time.Now().Add(time.Hour))
	require.NoError(t, err)
	_, err = chain.SendMsgs(grantMsg)
	require.NoError(t, err)

	// todo: verify execution of grant
}
