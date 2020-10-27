package integration

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/tendermint/tendermint/crypto"
)

// CreateTestApp will create a new wasmd application and provide money to every address
// listed there
func CreateTestApp(t *testing.T, accounts []*authtypes.BaseAccount) *app.WasmApp {
	fmt.Printf("%#v\n", accounts[1])
	genAccounts := make([]authtypes.GenesisAccount, len(accounts))
	for i, acct := range accounts {
		genAccounts[i] = acct
	}
	wasmd := SetupWithGenesisAccounts(genAccounts)
	return wasmd
}

func TestSendWithApp(t *testing.T) {
	// TODO: figure out how to get valid sigs
	t.Skip("Disabled until I get valid sigs")

	// TODO: how to change default coin?
	coin := sdk.NewCoins(sdk.NewInt64Coin("stake", 123456))
	accts, keys := genAccountsWithKey(t, coin, 2)
	wasm := CreateTestApp(t, accts)

	// TODO: check account balance first
	msg := banktypes.MsgSend{
		FromAddress: accts[0].Address,
		ToAddress:   accts[1].Address,
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 20)),
	}
	_ = sign(t, wasm, &msg, &keys[0], true)
}

func sign(t *testing.T, wasm *app.WasmApp, msg sdk.Msg, signer *signer, expectPass bool) *sdk.Result {
	txGen := app.MakeEncodingConfig().TxConfig
	_, res, _ := SignAndDeliver(t, txGen, wasm, []sdk.Msg{msg}, []uint64{signer.acctNum}, []uint64{signer.seq}, expectPass, signer.priv)
	if expectPass {
		signer.seq++
	}
	return res
}

type signer struct {
	priv    crypto.PrivKey
	seq     uint64
	acctNum uint64
}

func genAccountsWithKey(t *testing.T, coins sdk.Coins, n int) ([]*authtypes.BaseAccount, []signer) {
	//accts := make([]*authtypes.BaseAccount, n)
	//keys := make([]signer, n)
	//
	//for i := range accts {
	//	priv, pub, addr := maskKeyPubAddr()
	//	baseAcct := authtypes.NewBaseAccountWithAddress(addr)
	//	err := baseAcct.SetCoins(coins)
	//	require.NoError(t, err)
	//	baseAcct.SetPubKey(pub)
	//	baseAcct.SetAccountNumber(uint64(i + 1))
	//	baseAcct.SetSequence(1)
	//	accts[i] = &baseAcct
	//	keys[i] = signer{
	//		priv:    priv,
	//		acctNum: baseAcct.GetAccountNumber(),
	//		seq:     baseAcct.GetSequence(),
	//	}
	//}
	//
	//return accts, keys
	t.Fatal("not implemented")
	return nil, nil
}

var maskKeyCounter uint64 = 0

// we need to make this deterministic (same every test run), as encoded address size and thus gas cost,
// depends on the actual bytes (due to ugly CanonicalAddress encoding)
func maskKeyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	maskKeyCounter++
	seed := make([]byte, 8)
	binary.BigEndian.PutUint64(seed, maskKeyCounter)

	key := secp256k1.GenPrivKeyFromSecret(seed)
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}
