package integration

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth"
	authExported "github.com/cosmos/cosmos-sdk/x/auth/exported"
	"github.com/cosmos/cosmos-sdk/x/bank"

	"github.com/tendermint/tendermint/crypto"

	// "github.com/tendermint/tendermint/crypto/ed25519"
	"github.com/tendermint/tendermint/crypto/secp256k1"
)

// CreateTestApp will create a new wasmd application and provide money to every address
// listed there
func CreateTestApp(t *testing.T, accounts []*auth.BaseAccount) *app.WasmApp {
	fmt.Printf("%#v\n", accounts[1])
	genAccounts := make([]authExported.GenesisAccount, len(accounts))
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
	msg := bank.MsgSend{
		FromAddress: accts[0].Address,
		ToAddress:   accts[1].Address,
		Amount:      sdk.NewCoins(sdk.NewInt64Coin("stake", 20)),
	}
	_ = sign(t, wasm, msg, &keys[0], true)
}

func sign(t *testing.T, wasm *app.WasmApp, msg sdk.Msg, signer *signer, expectPass bool) *sdk.Result {
	_, res, _ := SignAndDeliver(t, wasm, []sdk.Msg{msg}, []uint64{signer.acctNum}, []uint64{signer.seq}, expectPass, signer.priv)
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

func genAccountsWithKey(t *testing.T, coins sdk.Coins, n int) ([]*auth.BaseAccount, []signer) {
	accts := make([]*auth.BaseAccount, n)
	keys := make([]signer, n)

	for i := range accts {
		priv, pub, addr := maskKeyPubAddr()
		baseAcct := auth.NewBaseAccountWithAddress(addr)
		err := baseAcct.SetCoins(coins)
		require.NoError(t, err)
		baseAcct.SetPubKey(pub)
		baseAcct.SetAccountNumber(uint64(i + 1))
		baseAcct.SetSequence(1)
		accts[i] = &baseAcct
		keys[i] = signer{
			priv:    priv,
			acctNum: baseAcct.GetAccountNumber(),
			seq:     baseAcct.GetSequence(),
		}
	}

	return accts, keys
}

var maskKeyCounter uint64 = 0

// we need to make this deterministic (same every test run), as encoded address size and thus gas cost,
// depends on the actual bytes (due to ugly CanonicalAddress encoding)
func maskKeyPubAddr() (crypto.PrivKey, crypto.PubKey, sdk.AccAddress) {
	maskKeyCounter++
	seed := make([]byte, 8)
	binary.BigEndian.PutUint64(seed, maskKeyCounter)

	key := secp256k1.GenPrivKeySecp256k1(seed)
	pub := key.PubKey()
	addr := sdk.AccAddress(pub.Address())
	return key, pub, addr
}
