package wasm_test

import (
	"testing"

	wasmd "github.com/CosmWasm/wasmd/app"

	wasmvmtypes "github.com/CosmWasm/wasmvm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"

	"github.com/CosmWasm/wasmd/x/wasm/ibctesting"
)

func TestIBCReflectContract(t *testing.T) {
	// scenario:
	//  chain A: ibc_reflect_send.wasm
	//  chain B: reflect.wasm + ibc_reflect.wasm
	//
	//  Chain A "ibc_reflect_send" sends a IBC packet "on channel connect" event to chain B "ibc_reflect"
	//  "ibc_reflect" sends a submessage to "reflect" which is returned as submessage.
	ibctesting.DefaultTestingAppInit = wasmd.SetupTestingApp
	var (
		coordinator = ibctesting.NewCoordinator(t, 2, nil, nil)
		chainA      = coordinator.GetChain(ibctesting.GetChainID(0))
		chainB      = coordinator.GetChain(ibctesting.GetChainID(1))
	)
	coordinator.CommitBlock(chainA, chainB)

	initMsg := []byte(`{}`)
	codeID := chainA.StoreCodeFile("./keeper/testdata/ibc_reflect_send.wasm").CodeID
	sendContractAddr := chainA.InstantiateContract(codeID, initMsg)

	reflectID := chainB.StoreCodeFile("./keeper/testdata/reflect.wasm").CodeID
	initMsg = wasmkeeper.IBCReflectInitMsg{
		ReflectCodeID: reflectID,
	}.GetBytes(t)
	codeID = chainB.StoreCodeFile("./keeper/testdata/ibc_reflect.wasm").CodeID

	reflectContractAddr := chainB.InstantiateContract(codeID, initMsg)
	var (
		sourcePortID      = wasmd.ContractInfo(t, chainA, sendContractAddr).IBCPortID
		counterpartPortID = wasmd.ContractInfo(t, chainB, reflectContractAddr).IBCPortID
	)

	path := ibctesting.NewPath(chainA, chainB)
	path.SetChannelOrdered()
	path.EndpointA.ChannelConfig.PortID = sourcePortID
	path.EndpointB.ChannelConfig.PortID = counterpartPortID
	path.EndpointA.ChannelConfig.Version = "ibc-reflect-v1"
	path.EndpointB.ChannelConfig.Version = "ibc-reflect-v1"

	coordinator.Setup(path)

	// ensure the expected packet was prepared, and relay it
	require.Equal(t, 1, len(chainA.PendingSendPackets))
	require.Equal(t, 0, len(chainB.PendingSendPackets))
	err := coordinator.RelayAndAckPendingPackets(path)
	require.NoError(t, err)
	require.Equal(t, 0, len(chainA.PendingSendPackets))
	require.Equal(t, 0, len(chainB.PendingSendPackets))

	// let's query the source contract and make sure it registered an address
	query := ReflectSendQueryMsg{Account: &AccountQuery{ChannelID: path.EndpointA.ChannelID}}
	var account AccountResponse
	err = chainA.SmartQuery(sendContractAddr.String(), query, &account)
	require.NoError(t, err)
	require.NotEmpty(t, account.RemoteAddr)
	require.Empty(t, account.RemoteBalance)

	// close channel
	err = path.CloseChannels()
	require.NoError(t, err)

	// let's query the source contract and make sure it registered an address
	account = AccountResponse{}
	err = chainA.SmartQuery(sendContractAddr.String(), query, &account)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

type ReflectSendQueryMsg struct {
	Admin        *struct{}     `json:"admin,omitempty"`
	ListAccounts *struct{}     `json:"list_accounts,omitempty"`
	Account      *AccountQuery `json:"account,omitempty"`
}

type AccountQuery struct {
	ChannelID string `json:"channel_id"`
}

type AccountResponse struct {
	LastUpdateTime uint64            `json:"last_update_time,string"`
	RemoteAddr     string            `json:"remote_addr"`
	RemoteBalance  wasmvmtypes.Coins `json:"remote_balance"`
}
