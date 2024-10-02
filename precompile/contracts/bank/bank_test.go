package bank_test

import (
	"encoding/hex"
	"math/big"
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/precompile/contracts/bank"
	"github.com/CosmWasm/wasmd/precompile/registry"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/go-bip39"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/evmos/ethermint/x/evm/statedb"
	"github.com/stretchr/testify/require"

	sdkmath "cosmossdk.io/math"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

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

func TestSend(t *testing.T) {
	denom := "ukava"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	mockAddr, mockEVMAddr := MockAddressPair()
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))
	receiveCosmosAddr, mockReceiverEVMAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, mockAddr, mockEVMAddr)
	tApp.EvmKeeper.SetAddressMapping(ctx, receiveCosmosAddr, mockReceiverEVMAddr)
	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100000)))
	sentCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(10)))
	bankKeeper := tApp.GetBankKeeper()
	accountKeeper := tApp.GetAccountKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, mockAddr, sentCoins)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	p, err := bank.NewContract(tApp.EvmKeeper, bankKeeper, accountKeeper)
	require.Nil(t, err)
	method := bank.ABI.Methods[bank.SendMethod]
	suppliedGas := uint64(10_000_000)

	args, err := method.Inputs.Pack(mockReceiverEVMAddr, denom, sentCoins[0].Amount.BigInt())
	require.Nil(t, err)
	res, _, err := p.Run(&evm, mockEVMAddr, registry.AddrContractAddress,
		append(method.ID, args...),
		suppliedGas,
		false,
		nil,
	)
	require.Nil(t, err)
	output, err := method.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(bool), true)

	balance := bankKeeper.GetBalance(ctx, receiveCosmosAddr, denom)
	require.Equal(t, balance.Amount, sdkmath.NewInt(10))
}

func TestBalance(t *testing.T) {
	denom := "ukava"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	mockAddr, mockEVMAddr := MockAddressPair()
	tApp.EvmKeeper.SetAddressMapping(ctx, mockAddr, mockEVMAddr)
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))
	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100000)))
	sentCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(10)))
	bankKeeper := tApp.GetBankKeeper()
	accountKeeper := tApp.GetAccountKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SendCoinsFromModuleToAccount(ctx, evmtypes.ModuleName, mockAddr, sentCoins)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	p, err := bank.NewContract(tApp.EvmKeeper, bankKeeper, accountKeeper)
	require.Nil(t, err)
	method := bank.ABI.Methods[bank.BalanceMethod]
	suppliedGas := uint64(10_000_000)

	args, err := method.Inputs.Pack(mockEVMAddr, denom)
	require.Nil(t, err)
	res, _, err := p.Run(&evm, registry.AddrContractAddress, registry.AddrContractAddress,
		append(method.ID, args...),
		suppliedGas,
		false,
		nil,
	)
	require.Nil(t, err)
	output, err := method.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(*big.Int), big.NewInt(sentCoins[0].Amount.Int64()))
}

func TestSupply(t *testing.T) {
	denom := "ukava"
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)
	sdk.RegisterDenom(denom, sdkmath.LegacyNewDec(6))
	mintCoins := sdk.NewCoins(sdk.NewCoin(denom, sdkmath.NewInt(100000)))
	bankKeeper := tApp.GetBankKeeper()
	accountKeeper := tApp.GetAccountKeeper()
	err := bankKeeper.MintCoins(ctx, evmtypes.ModuleName, mintCoins)
	require.NoError(t, err)
	tApp.GetBankKeeper().SetParams(ctx, banktypes.DefaultParams())

	evm := vm.EVM{
		StateDB: statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
	}
	p, err := bank.NewContract(tApp.EvmKeeper, bankKeeper, accountKeeper)
	require.Nil(t, err)
	method := bank.ABI.Methods[bank.SupplyMethod]
	suppliedGas := uint64(10_000_000)

	args, err := method.Inputs.Pack(denom)
	require.Nil(t, err)
	res, _, err := p.Run(&evm, registry.AddrContractAddress, registry.AddrContractAddress,
		append(method.ID, args...),
		suppliedGas,
		false,
		nil,
	)
	require.Nil(t, err)
	output, err := method.Outputs.Unpack(res)
	require.Nil(t, err)
	require.Equal(t, 1, len(output))
	require.Equal(t, output[0].(*big.Int), big.NewInt(mintCoins[0].Amount.Int64()))
}
