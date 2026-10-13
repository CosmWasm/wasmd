package app

import (
	"context"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	abci "github.com/cometbft/cometbft/abci/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	"github.com/cosmos/cosmos-sdk/crypto/keyring"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	"github.com/cosmos/cosmos-sdk/runtime"
	storetypes "github.com/cosmos/cosmos-sdk/store/v2/types"
	"github.com/cosmos/cosmos-sdk/testutil/mock"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/types/tx/signing"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmtypes "github.com/CosmWasm/wasmd/x/wasm/types"
)

func TestCounterGasSimulationCoversDelivery(t *testing.T) {
	for _, counterExists := range []bool{false, true} {
		for _, omitCounterGas := range []bool{false, true} {
			t.Run(fmt.Sprintf("counter-exists=%t/omit-counter-gas=%t", counterExists, omitCounterGas), func(t *testing.T) {
				const chainID = "counter-gas-test"
				encoding := MakeEncodingConfig(t)
				kb := keyring.NewInMemory(encoding.Codec)
				record, _, err := kb.NewMnemonic("sender", keyring.English, hd.CreateHDPath(118, 0, 0).String(), "", hd.Secp256k1)
				require.NoError(t, err)
				pubkey, err := record.GetPubKey()
				require.NoError(t, err)
				sender := sdk.AccAddress(pubkey.Address())
				recipientKey := secp256k1.GenPrivKey().PubKey()
				recipient := sdk.AccAddress(recipientKey.Address())
				validator := mock.NewPV()
				consensusKey, err := validator.GetPubKey()
				require.NoError(t, err)
				valSet := cmttypes.NewValidatorSet([]*cmttypes.Validator{cmttypes.NewValidator(consensusKey, 1)})
				app := SetupWithGenesisValSet(t, valSet, []authtypes.GenesisAccount{
					authtypes.NewBaseAccount(sender, pubkey, 0, 0),
					authtypes.NewBaseAccount(recipient, recipientKey, 1, 0),
				}, chainID, nil,
					banktypes.Balance{Address: sender.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("stake", 10000000))},
					banktypes.Balance{Address: recipient.String(), Coins: sdk.NewCoins(sdk.NewInt64Coin("stake", 10000000))})
				key := app.GetKey(wasmtypes.StoreKey)
				var initial []byte
				if counterExists {
					initial = make([]byte, 12)
					binary.BigEndian.PutUint64(initial, 1)
					binary.BigEndian.PutUint32(initial[8:], 4)
					app.GetContextForFinalizeBlock(nil).MultiStore().GetKVStore(key).Set(wasmtypes.TXCounterPrefix, initial)
					app.SimWriteState()
				}
				_, err = app.Commit()
				require.NoError(t, err)
				ctx := app.GetContextForCheckTx(nil)
				account := app.AccountKeeper.GetAccount(ctx, sender)
				balance := app.BankKeeper.GetBalance(ctx, sender, "stake")
				config := app.TxConfig()
				// Fixed fees and filled signatures isolate counter gas from SDK fee
				// estimation and empty-signature size allowances.
				f := tx.Factory{}.WithTxConfig(config).WithKeybase(kb).WithFromName("sender").WithChainID(chainID).
					WithAccountNumber(account.GetAccountNumber()).WithSequence(account.GetSequence()).
					WithFees("1000stake").WithSignMode(signing.SignMode_SIGN_MODE_DIRECT)
				msg := banktypes.NewMsgSend(sender, recipient, sdk.NewCoins(sdk.NewInt64Coin("stake", 1)))
				buildSigned := func(gas uint64) []byte {
					built, err := f.WithGas(gas).BuildUnsignedTx(msg)
					require.NoError(t, err)
					require.NoError(t, tx.Sign(context.Background(), f.WithGas(gas), "sender", built, true))
					bz, err := config.TxEncoder()(built.GetTx())
					require.NoError(t, err)
					return bz
				}
				info, _, err := app.Simulate(buildSigned(200000))
				require.NoError(t, err)
				// Measure the real execution decorator in an isolated branch, rather
				// than assuming a fixed counter charge.
				counterCtx, _ := ctx.CacheContext()
				counterCtx = counterCtx.WithGasMeter(storetypes.NewInfiniteGasMeter())
				_, err = wasmkeeper.NewCountTXDecorator(runtime.NewKVStoreService(key)).AnteHandle(counterCtx, nil, false,
					func(ctx sdk.Context, _ sdk.Tx, _ bool) (sdk.Context, error) { return ctx, nil })
				require.NoError(t, err)
				counterGas := counterCtx.GasMeter().GasConsumed()
				require.Greater(t, info.GasUsed, counterGas)
				limit := info.GasUsed
				if omitCounterGas {
					// Reproduce the previous measurement that omitted these operations.
					limit -= counterGas
				}
				bz := buildSigned(limit)
				finalInfo, _, err := app.Simulate(bz)
				require.NoError(t, err)
				require.Equal(t, info.GasUsed, finalInfo.GasUsed, "candidate fields must preserve the measured size")
				require.Equal(t, initial, ctx.MultiStore().GetKVStore(key).Get(wasmtypes.TXCounterPrefix))
				require.Equal(t, account.GetSequence(), app.AccountKeeper.GetAccount(ctx, sender).GetSequence())
				require.Equal(t, balance, app.BankKeeper.GetBalance(ctx, sender, "stake"))
				block, err := app.FinalizeBlock(&abci.RequestFinalizeBlock{
					Height: 2, Time: time.Now().UTC(), NextValidatorsHash: valSet.Hash(), Txs: [][]byte{bz},
				})
				require.NoError(t, err)
				require.Len(t, block.TxResults, 1)
				result := block.TxResults[0]
				if omitCounterGas {
					require.EqualValues(t, 11, result.Code, "%+v", result)
				} else {
					require.Zero(t, result.Code, "%+v", result)
					require.LessOrEqual(t, uint64(result.GasUsed), limit)
				}
				t.Logf("counter=%d measured=%d limit=%d execution=%d code=%d", counterGas, finalInfo.GasUsed, limit, result.GasUsed, result.Code)
			})
		}
	}
}
