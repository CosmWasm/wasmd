package ibctesting

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"cosmossdk.io/math"
	"github.com/cometbft/cometbft/libs/rand"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/gogoproto/proto"

	"github.com/CosmWasm/wasmd/app"
	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	"github.com/CosmWasm/wasmd/x/wasm/types"
	abci "github.com/cometbft/cometbft/abci/types"
	clienttypes "github.com/cosmos/ibc-go/v9/modules/core/02-client/types"
	channeltypes "github.com/cosmos/ibc-go/v9/modules/core/04-channel/types"
	host "github.com/cosmos/ibc-go/v9/modules/core/24-host"
	ibctesting "github.com/cosmos/ibc-go/v9/testing"
	ibctestingtypes "github.com/cosmos/ibc-go/v9/testing/types"
)

type WasmTestApp struct {
	*app.WasmApp
}

func (app WasmTestApp) GetStakingKeeper() ibctestingtypes.StakingKeeper {
	return app.WasmApp.GetStakingKeeper()
}

func (app WasmTestApp) GetTxConfig() client.TxConfig {
	return app.TxConfig()
}

type WasmTestChain struct {
	*ibctesting.TestChain

	PendingSendPackets []channeltypes.Packet
}

func NewWasmTestChain(chain *ibctesting.TestChain) WasmTestChain {
	return WasmTestChain{TestChain: chain, PendingSendPackets: []channeltypes.Packet{}}
}

func (chain *WasmTestChain) CaptureIBCEvents(result *abci.ExecTxResult) {
	toSend, _ := ibctesting.ParsePacketsFromEvents(channeltypes.EventTypeSendPacket, result.Events)
	// require.NoError(chain, err)
	if len(toSend) > 0 {
		// Keep a queue on the chain that we can relay in tests
		chain.PendingSendPackets = append(chain.PendingSendPackets, toSend...)
	}
}

func (chain *WasmTestChain) SendMsgs(msgs ...sdk.Msg) (*abci.ExecTxResult, error) {
	result, err := chain.TestChain.SendMsgs(msgs...)
	chain.CaptureIBCEvents(result)
	return result, err
}

func (chain WasmTestChain) GetWasmApp() *app.WasmApp {
	return chain.App.(WasmTestApp).WasmApp
}

func (chain WasmTestChain) StoreCodeFile(filename string) types.MsgStoreCodeResponse {
	wasmCode, err := os.ReadFile(filename)
	require.NoError(chain.TB, err)
	if strings.HasSuffix(filename, "wasm") { // compress for gas limit
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		_, err := gz.Write(wasmCode)
		require.NoError(chain.TB, err)
		err = gz.Close()
		require.NoError(chain.TB, err)
		wasmCode = buf.Bytes()
	}
	return chain.StoreCode(wasmCode)
}

func (chain WasmTestChain) StoreCode(byteCode []byte) types.MsgStoreCodeResponse {
	storeMsg := &types.MsgStoreCode{
		Sender:       chain.SenderAccount.GetAddress().String(),
		WASMByteCode: byteCode,
	}
	r, err := chain.SendMsgs(storeMsg)
	require.NoError(chain.TB, err)

	var pInstResp types.MsgStoreCodeResponse
	chain.UnwrapExecTXResult(r, &pInstResp)

	require.NotEmpty(chain.TB, pInstResp.CodeID)
	require.NotEmpty(chain.TB, pInstResp.Checksum)
	return pInstResp
}

// UnwrapExecTXResult is a helper to unpack execution result from proto any type
func (chain WasmTestChain) UnwrapExecTXResult(r *abci.ExecTxResult, target proto.Message) {
	var wrappedRsp sdk.TxMsgData
	require.NoError(chain.TB, chain.App.AppCodec().Unmarshal(r.Data, &wrappedRsp))

	// unmarshal protobuf response from data
	require.Len(chain.TB, wrappedRsp.MsgResponses, 1)
	require.NoError(chain.TB, proto.Unmarshal(wrappedRsp.MsgResponses[0].Value, target))
}

func (chain WasmTestChain) InstantiateContract(codeID uint64, initMsg []byte) sdk.AccAddress {
	instantiateMsg := &types.MsgInstantiateContract{
		Sender: chain.SenderAccount.GetAddress().String(),
		Admin:  chain.SenderAccount.GetAddress().String(),
		CodeID: codeID,
		Label:  "ibc-test",
		Msg:    initMsg,
		Funds:  sdk.Coins{ibctesting.TestCoin},
	}

	r, err := chain.SendMsgs(instantiateMsg)
	require.NoError(chain.TB, err)

	var pExecResp types.MsgInstantiateContractResponse
	chain.UnwrapExecTXResult(r, &pExecResp)

	a, err := sdk.AccAddressFromBech32(pExecResp.Address)
	require.NoError(chain.TB, err)
	return a
}

// SeedNewContractInstance stores some wasm code and instantiates a new contract on this chain.
// This method can be called to prepare the store with some valid CodeInfo and ContractInfo. The returned
// Address is the contract address for this instance. Test should make use of this data and/or use NewIBCContractMockWasmEngine
// for using a contract mock in Go.
func (chain *WasmTestChain) SeedNewContractInstance() sdk.AccAddress {
	pInstResp := chain.StoreCode(append(wasmIdent, rand.Bytes(10)...))
	codeID := pInstResp.CodeID

	anyAddressStr := chain.SenderAccount.GetAddress().String()
	initMsg := []byte(fmt.Sprintf(`{"verifier": %q, "beneficiary": %q}`, anyAddressStr, anyAddressStr))
	return chain.InstantiateContract(codeID, initMsg)
}

func (chain WasmTestChain) ContractInfo(contractAddr sdk.AccAddress) *types.ContractInfo {
	return chain.App.(WasmTestApp).GetWasmKeeper().GetContractInfo(chain.GetContext(), contractAddr)
}

// Fund an address with the given amount in default denom
func (chain WasmTestChain) Fund(addr sdk.AccAddress, amount math.Int) {
	_, err := chain.SendMsgs(&banktypes.MsgSend{
		FromAddress: chain.SenderAccount.GetAddress().String(),
		ToAddress:   addr.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amount)),
	})
	require.NoError(chain.TB, err)
}

// GetTimeoutHeight is a convenience function which returns a IBC packet timeout height
// to be used for testing. It returns the current IBC height + 100 blocks
func (chain WasmTestChain) GetTimeoutHeight() clienttypes.Height {
	return clienttypes.NewHeight(clienttypes.ParseChainID(chain.ChainID), uint64(chain.GetContext().BlockHeight())+100)
}

func (chain WasmTestChain) Balance(acc sdk.AccAddress, denom string) sdk.Coin {
	return chain.App.(WasmTestApp).GetBankKeeper().GetBalance(chain.GetContext(), acc, denom)
}

func (chain WasmTestChain) AllBalances(acc sdk.AccAddress) sdk.Coins {
	return chain.App.(WasmTestApp).GetBankKeeper().GetAllBalances(chain.GetContext(), acc)
}

// SendNonDefaultSenderMsgs is the same as SendMsgs but with a custom signer/account
func (chain *WasmTestChain) SendNonDefaultSenderMsgs(senderPrivKey cryptotypes.PrivKey, msgs ...sdk.Msg) (*abci.ExecTxResult, error) {
	require.NotEqual(chain.TB, chain.SenderPrivKey, senderPrivKey, "use SendMsgs method")

	addr := sdk.AccAddress(senderPrivKey.PubKey().Address().Bytes())
	account := chain.GetWasmApp().GetAccountKeeper().GetAccount(chain.GetContext(), addr)
	prevAccount := chain.SenderAccount
	prevSenderPrivKey := chain.SenderPrivKey
	chain.SenderAccount = account
	chain.SenderPrivKey = senderPrivKey

	require.NotNil(chain.TB, account)
	result, err := chain.SendMsgs(msgs...)

	chain.SenderAccount = prevAccount
	chain.SenderPrivKey = prevSenderPrivKey

	return result, err
}

// SmartQuery This will serialize the query message and submit it to the contract.
// The response is parsed into the provided interface.
// Usage: SmartQuery(addr, QueryMsg{Foo: 1}, &response)
func (chain *WasmTestChain) SmartQuery(contractAddr string, queryMsg, response interface{}) error {
	msg, err := json.Marshal(queryMsg)
	if err != nil {
		return err
	}

	req := types.QuerySmartContractStateRequest{
		Address:   contractAddr,
		QueryData: msg,
	}
	reqBin, err := proto.Marshal(&req)
	if err != nil {
		return err
	}

	res, err := chain.App.Query(context.TODO(), &abci.RequestQuery{
		Path: "/cosmwasm.wasm.v1.Query/SmartContractState",
		Data: reqBin,
	})
	require.NoError(chain.TB, err)

	if res.Code != 0 {
		return fmt.Errorf("smart query failed: (%d) %s", res.Code, res.Log)
	}

	// unpack protobuf
	var resp types.QuerySmartContractStateResponse
	err = proto.Unmarshal(res.Value, &resp)
	if err != nil {
		return err
	}
	// unpack json content
	return json.Unmarshal(resp.Data, response)
}

// RelayAndAckPendingPackets sends pending packages from path.EndpointA to the counterparty chain and acks
func RelayAndAckPendingPackets(chainA *WasmTestChain, chainB *WasmTestChain, path *ibctesting.Path) error {
	// get all the packet to relay src->dest
	src := path.EndpointA
	require.NoError(chainA, src.UpdateClient())
	chainA.Logf("Relay: %d Packets A->B, %d Packets B->A\n", len(chainA.PendingSendPackets), len(chainB.PendingSendPackets))
	for _, v := range chainA.PendingSendPackets {
		result, _, err := path.RelayPacketWithResults(v)
		if err != nil {
			return err
		}
		chainA.CaptureIBCEvents(result)
		chainA.PendingSendPackets = chainA.PendingSendPackets[1:]
	}

	src = path.EndpointB
	require.NoError(chainB, src.UpdateClient())
	for _, v := range chainB.PendingSendPackets {
		result, _, err := path.RelayPacketWithResults(v)
		if err != nil {
			return err
		}
		chainA.CaptureIBCEvents(result)
		chainB.PendingSendPackets = chainB.PendingSendPackets[1:]
	}
	return nil
}

// TimeoutPendingPackets returns the package to source chain to let the IBC app revert any operation.
// from A to B
func TimeoutPendingPackets(coord *ibctesting.Coordinator, chainSrc *WasmTestChain, path *ibctesting.Path) error {
	src := path.EndpointA
	dest := path.EndpointB

	toSend := chainSrc.PendingSendPackets
	coord.Logf("Timeout %d Packets A->B\n", len(toSend))
	require.NoError(coord, src.UpdateClient())

	// Increment time and commit block so that 5 second delay period passes between send and receive
	coord.IncrementTime()
	coord.CommitBlock(src.Chain, dest.Chain)
	for _, packet := range toSend {
		// get proof of packet unreceived on dest
		packetKey := host.PacketReceiptKey(packet.GetDestPort(), packet.GetDestChannel(), packet.GetSequence())
		proofUnreceived, proofHeight := dest.QueryProof(packetKey)
		timeoutMsg := channeltypes.NewMsgTimeout(packet, packet.Sequence, proofUnreceived, proofHeight, src.Chain.SenderAccount.GetAddress().String())
		_, err := chainSrc.SendMsgs(timeoutMsg)
		if err != nil {
			return err
		}
	}
	chainSrc.PendingSendPackets = nil
	return nil
}

// CloseChannel close channel on both sides
func CloseChannel(coord *ibctesting.Coordinator, path *ibctesting.Path) {
	err := path.EndpointA.ChanCloseInit()
	require.NoError(coord, err)
	coord.IncrementTime()
	err = path.EndpointB.UpdateClient()
	require.NoError(coord, err)
	channelKey := host.ChannelKey(path.EndpointB.Counterparty.ChannelConfig.PortID, path.EndpointB.Counterparty.ChannelID)
	proof, proofHeight := path.EndpointB.Counterparty.QueryProof(channelKey)
	msg := channeltypes.NewMsgChannelCloseConfirm(
		path.EndpointB.ChannelConfig.PortID, path.EndpointB.ChannelID,
		proof, proofHeight,
		path.EndpointB.Chain.SenderAccount.GetAddress().String(), 0,
	)
	_, err = path.EndpointB.Chain.SendMsgs(msg)
	require.NoError(coord, err)
}

// ChainAppFactory abstract factory method that usually implemented by app.SetupWithGenesisValSet
type ChainAppFactory2 func(t *testing.T, valSet *cmttypes.ValidatorSet, genAccs []authtypes.GenesisAccount, chainID string, opts []wasmkeeper.Option, balances ...banktypes.Balance) WasmTestApp

// DefaultWasmAppFactory instantiates and sets up the default wasmd app
func DefaultWasmAppFactory2(t *testing.T, valSet *cmttypes.ValidatorSet, genAccs []authtypes.GenesisAccount, chainID string, opts []wasmkeeper.Option, balances ...banktypes.Balance) WasmTestApp {
	return WasmTestApp{app.SetupWithGenesisValSet(t, valSet, genAccs, chainID, opts, balances...)}
}

// NewDefaultTestChain initializes a new test chain with a default of 4 validators
// Use this function if the tests do not need custom control over the validator set
func NewDefaultTestChain2(t *testing.T, coord *ibctesting.Coordinator, chainID string, opts ...wasmkeeper.Option) *ibctesting.TestChain {
	return NewTestChain2(t, coord, DefaultWasmAppFactory2, chainID, opts...)
}

// NewTestChain initializes a new test chain with a default of 4 validators
// Use this function if the tests do not need custom control over the validator set
func NewTestChain2(t *testing.T, coord *ibctesting.Coordinator, appFactory ChainAppFactory2, chainID string, opts ...wasmkeeper.Option) *ibctesting.TestChain {
	// generate validators private/public key
	var (
		validatorsPerChain = 4
		validators         = make([]*cmttypes.Validator, 0, validatorsPerChain)
		signersByAddress   = make(map[string]cmttypes.PrivValidator, validatorsPerChain)
	)

	for i := 0; i < validatorsPerChain; i++ {
		_, privVal := cmttypes.RandValidator(false, 100)
		pubKey, err := privVal.GetPubKey()
		require.NoError(t, err)
		validators = append(validators, cmttypes.NewValidator(pubKey, 1))
		signersByAddress[pubKey.Address().String()] = privVal
	}

	// construct validator set;
	// Note that the validators are sorted by voting power
	// or, if equal, by address lexical order
	valSet := cmttypes.NewValidatorSet(validators)

	return NewTestChainWithValSet2(t, coord, appFactory, chainID, valSet, signersByAddress, opts...)
}

// NewTestChainWithValSet initializes a new TestChain instance with the given validator set
// and signer array. It also initializes 10 Sender accounts with a balance of 10000000000000000000 coins of
// bond denom to use for tests.
//
// The first block height is committed to state in order to allow for client creations on
// counterparty chains. The TestChain will return with a block height starting at 2.
//
// Time management is handled by the Coordinator in order to ensure synchrony between chains.
// Each update of any chain increments the block header time for all chains by 5 seconds.
//
// NOTE: to use a custom sender privkey and account for testing purposes, replace and modify this
// constructor function.
//
// CONTRACT: Validator array must be provided in the order expected by Tendermint.
// i.e. sorted first by power and then lexicographically by address.
func NewTestChainWithValSet2(t *testing.T, coord *ibctesting.Coordinator, appFactory ChainAppFactory2, chainID string, valSet *cmttypes.ValidatorSet, signers map[string]cmttypes.PrivValidator, opts ...wasmkeeper.Option) *ibctesting.TestChain {
	genAccs := []authtypes.GenesisAccount{}
	genBals := []banktypes.Balance{}
	senderAccs := []ibctesting.SenderAccount{}

	// generate genesis accounts
	for i := 0; i < MaxAccounts; i++ {
		senderPrivKey := secp256k1.GenPrivKey()
		acc := authtypes.NewBaseAccount(senderPrivKey.PubKey().Address().Bytes(), senderPrivKey.PubKey(), uint64(i), 0)
		amount, ok := math.NewIntFromString("10000000000000000000")
		require.True(t, ok)

		// add sender account
		balance := banktypes.Balance{
			Address: acc.GetAddress().String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(sdk.DefaultBondDenom, amount)),
		}

		genAccs = append(genAccs, acc)
		genBals = append(genBals, balance)

		senderAcc := ibctesting.SenderAccount{
			SenderAccount: acc,
			SenderPrivKey: senderPrivKey,
		}

		senderAccs = append(senderAccs, senderAcc)
	}

	wasmApp := appFactory(t, valSet, genAccs, chainID, opts, genBals...)

	// create current header and call begin block
	header := cmtproto.Header{
		ChainID: chainID,
		Height:  1,
		Time:    coord.CurrentTime.UTC(),
	}

	txConfig := wasmApp.GetTxConfig()

	// create an account to send transactions from
	chain := &ibctesting.TestChain{
		TB:             t,
		Coordinator:    coord,
		ChainID:        chainID,
		App:            wasmApp,
		ProposedHeader: header,
		TxConfig:       txConfig,
		Codec:          wasmApp.AppCodec(),
		Vals:           valSet,
		NextVals:       valSet,
		Signers:        signers,
		SenderPrivKey:  senderAccs[0].SenderPrivKey,
		SenderAccount:  senderAccs[0].SenderAccount,
		SenderAccounts: senderAccs,
	}

	coord.CommitBlock(chain)

	return chain
}

// NewCoordinator initializes Coordinator with N TestChain's
func NewCoordinator2(t *testing.T, n int, opts ...[]wasmkeeper.Option) *ibctesting.Coordinator {
	t.Helper()
	chains := make(map[string]*ibctesting.TestChain)
	coord := &ibctesting.Coordinator{
		T:           t,
		CurrentTime: globalStartTime,
	}

	for i := 1; i <= n; i++ {
		chainID := ibctesting.GetChainID(i)
		var x []wasmkeeper.Option
		if len(opts) > (i - 1) {
			x = opts[i-1]
		}
		chains[chainID] = NewDefaultTestChain2(t, coord, chainID, x...)
	}
	coord.Chains = chains

	return coord
}
