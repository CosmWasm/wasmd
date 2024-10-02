package wasmd_test

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"os"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/precompile/contracts/wasmd"
	"github.com/CosmWasm/wasmd/precompile/registry"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/cosmos/go-bip39"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/precompile/modules"
	"github.com/evmos/ethermint/x/evm/statedb"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tmtypes "github.com/cometbft/cometbft/proto/tendermint/types"
)

type MockWasmer struct {
}

func (m *MockWasmer) Instantiate(ctx sdk.Context, codeID uint64, creator, admin sdk.AccAddress, initMsg []byte, label string, deposit sdk.Coins) (sdk.AccAddress, []byte, error) {
	addr := sdk.MustAccAddressFromBech32("orai19xtunzaq20unp8squpmfrw8duclac22hd7ves2")
	return addr, []byte("Instantiate"), nil
}

func (m *MockWasmer) Execute(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error) {
	return []byte("Execute"), nil
}

func (m *MockWasmer) QuerySmart(ctx context.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error) {
	return []byte("QuerySmart"), nil
}

func MockAddressPair() (sdk.AccAddress, common.Address) {
	return PrivateKeyToAddresses(MockPrivateKey())
}

func MockPrivateKey() cryptotypes.PrivKey {
	// Generate a new Sei private key
	entropySeed, _ := bip39.NewEntropy(256)
	mnemonic, _ := bip39.NewMnemonic(entropySeed)
	algo := hd.Secp256k1
	derivedPriv, _ := algo.Derive()(mnemonic, "", "")
	return algo.Generate()(derivedPriv)
}

func PrivateKeyToAddresses(privKey cryptotypes.PrivKey) (sdk.AccAddress, common.Address) {
	// Encode the private key to hex (i.e. what wallets do behind the scene when users reveal private keys)
	testPrivHex := hex.EncodeToString(privKey.Bytes())

	// Sign an Ethereum transaction with the hex private key
	key, _ := crypto.HexToECDSA(testPrivHex)
	msg := crypto.Keccak256([]byte("foo"))
	sig, _ := crypto.Sign(msg, key)

	// Recover the public keys from the Ethereum signature
	recoveredPub, _ := crypto.Ecrecover(msg, sig)
	pubKey, _ := crypto.UnmarshalPubkey(recoveredPub)

	return sdk.AccAddress(privKey.PubKey().Address()), crypto.PubkeyToAddress(*pubKey)
}

func TestUnmarshalCosmWasmDeposit(t *testing.T) {
	deposit := wasmd.UnmarshalCosmWasmDeposit([]byte("[]"))
	require.Equal(t, deposit, sdk.NewCoins())
	deposit = wasmd.UnmarshalCosmWasmDeposit([]byte("foobar"))
	require.Equal(t, deposit, sdk.NewCoins())
	deposit = wasmd.UnmarshalCosmWasmDeposit([]byte("{}"))
	require.Equal(t, deposit, sdk.NewCoins())
	deposit = wasmd.UnmarshalCosmWasmDeposit([]byte("[{\"denom\":\"ukava\",\"amount\":\"10\"}, {\"denom\":\"orai\",\"amount\":\"100\"}]"))
	coins := sdk.NewCoins(sdk.NewCoin("ukava", sdkmath.NewInt(10)), sdk.NewCoin("orai", sdkmath.NewInt(100)))
	for _, coin := range coins {
		if coin.Denom == "orai" {
			require.Equal(t, coin.Amount, sdkmath.NewInt(100))
		} else if coin.Denom == "ukava" {
			require.Equal(t, coin.Amount, sdkmath.NewInt(10))
		} else {
			panic("Wrong Unmarshal")
		}
	}
}

// TestContractConstructor ensures we have a valid constructor. This will fail
// if we attempt to define invalid or duplicate function selectors.
func TestContractConstructor(t *testing.T) {
	wasmer := &MockWasmer{}
	precompile, err := wasmd.NewContract(wasmer, wasmer, nil)
	require.NoError(t, err, "expected precompile not error when created")
	assert.NotNil(t, precompile, "expected precompile contract to be defined")
}

func TestExecuteAndQuery(t *testing.T) {

	tApp := app.Setup(t)
	ctx := tApp.NewContextLegacy(true, tmtypes.Header{Height: 1, ChainID: "wasmd-test", Time: time.Now().UTC()})
	tApp.GetWasmKeeper().SetParams(ctx, wasmtypes.DefaultParams())
	mockAddr, mockEVMAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, mockAddr, mockEVMAddr)
	sdk.RegisterDenom("ukava", sdkmath.LegacyNewDec(6))
	amts := sdk.NewCoins(sdk.NewCoin("ukava", sdkmath.NewInt(1000)))
	tApp.GetBankKeeper().MintCoins(ctx, evmtypes.ModuleName, amts)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, mockAddr, amts)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	println("acc addr", mockAddr.String())

	code, err := os.ReadFile("../../cosmwasm/echo/artifacts/echo.wasm")
	require.Nil(t, err)
	codeID, _, err := tApp.ContractKeeper.Create(ctx, mockAddr, code, nil)
	require.Nil(t, err)

	p, _ := modules.GetPrecompileModuleByAddress(registry.WasmdContractAddress)

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	suppliedGas := uint64(10_000_000)

	instantiateMethod := wasmd.ABI.Methods["instantiate"]

	args, err := instantiateMethod.Inputs.Pack(codeID, mockAddr.String(), []byte("{}"), "test", []byte("foo"))
	require.Nil(t, err)
	res, suppliedGas, err := p.Contract.Run(&evm, registry.WasmdContractAddress, registry.WasmdContractAddress,
		append(instantiateMethod.ID, args...),
		suppliedGas,
		false,
		nil,
	)
	require.Nil(t, err)
	rets, _ := instantiateMethod.Outputs.Unpack(res)
	cosmwasmAddr := rets[0].(string)

	// test execute
	executeMethod := wasmd.ABI.Methods["execute"]
	funds := sdk.NewCoins(sdk.NewCoin("ukava", sdkmath.NewInt(10)))
	err = tApp.GetBankKeeper().IsSendEnabledCoins(ctx, funds...)
	require.Nil(t, err)

	fundsBz, _ := json.Marshal(funds)
	args, err = executeMethod.Inputs.Pack(cosmwasmAddr, []byte("{\"echo\":{\"message\":\"test msg\"}}"), fundsBz)
	require.Nil(t, err)

	res, suppliedGas, err = p.Contract.Run(&evm, mockEVMAddr, registry.WasmdContractAddress,
		append(executeMethod.ID, args...),
		suppliedGas,
		false,
		nil,
	)
	require.Nil(t, err)
	rets, _ = executeMethod.Outputs.Unpack(res)
	response := rets[0].([]byte)
	t.Logf("res %s, gas remained %v", response, suppliedGas)

	// check balance after sent funds. Should drop
	balanceAfterExecute := tApp.GetBankKeeper().GetBalance(ctx, mockAddr, "ukava")
	require.Equal(t, balanceAfterExecute, amts[0].Sub(funds[0]))

	// test query
	queryMethod := wasmd.ABI.Methods["query"]

	args, err = queryMethod.Inputs.Pack(cosmwasmAddr, []byte("{\"info\":{}}"))
	require.Nil(t, err)

	res, suppliedGas, err = p.Contract.Run(&evm, mockEVMAddr, registry.WasmdContractAddress,
		append(queryMethod.ID, args...),
		suppliedGas,
		false,
		nil,
	)
	require.Nil(t, err)
	rets, _ = queryMethod.Outputs.Unpack(res)
	response = rets[0].([]byte)
	require.Equal(t, base64.StdEncoding.EncodeToString(response), "eyJtZXNzYWdlIjoicXVlcnkgdGVzdCJ9")
}
