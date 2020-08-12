package app

import (
	"fmt"

	"github.com/CosmWasm/wasmd/app"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/tendermint/tendermint/crypto/secp256k1"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// This will fail half the time with the second output being 173
// This is due to secp256k1 signatures not being constant size.
// nolint: vet
func ExampleTxSendSize() {
	cdc := app.MakeEncodingConfig().Amino

	var gas uint64 = 1

	priv1 := secp256k1.GenPrivKeySecp256k1([]byte{0})
	addr1 := sdk.AccAddress(priv1.PubKey().Address())
	priv2 := secp256k1.GenPrivKeySecp256k1([]byte{1})
	addr2 := sdk.AccAddress(priv2.PubKey().Address())
	coins := sdk.Coins{sdk.NewCoin("denom", sdk.NewInt(10))}
	msg1 := banktypes.MsgMultiSend{
		Inputs:  []banktypes.Input{banktypes.NewInput(addr1, coins)},
		Outputs: []banktypes.Output{banktypes.NewOutput(addr2, coins)},
	}
	fee := authtypes.NewStdFee(gas, coins)
	signBytes := authtypes.StdSignBytes("example-chain-ID",
		1, 1, 100, fee, []sdk.Msg{&msg1}, "")
	sig, err := priv1.Sign(signBytes)
	if err != nil {
		return
	}
	sigs := []authtypes.StdSignature{{Signature: sig}}
	tx := authtypes.NewStdTx([]sdk.Msg{&msg1}, fee, sigs, "")
	fmt.Println(len(cdc.MustMarshalBinaryBare([]sdk.Msg{&msg1})))
	fmt.Println(len(cdc.MustMarshalBinaryBare(tx)))
	// output: 80
	// 169
}
