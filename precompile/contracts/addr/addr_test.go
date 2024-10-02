package addr_test

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"math/big"
	"testing"

	"github.com/CosmWasm/wasmd/app"
	"github.com/CosmWasm/wasmd/precompile/contracts/addr"
	"github.com/cosmos/cosmos-sdk/crypto/hd"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/go-bip39"
	"github.com/ethereum/go-ethereum/common"
	"github.com/evmos/ethermint/x/evm/statedb"

	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
)

const suppliedGas = uint64(10_000_000)

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

func TestGetCosmosAddr(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	method := addr.ABI.Methods[addr.GetCosmosAddressMethod]

	targetPrivKey := MockPrivateKey()
	targetCosmosAddress, targetEvmAddress := PrivateKeyToAddresses(targetPrivKey)
	targetCosmosAddressNoMapping := sdk.AccAddress(targetEvmAddress.Bytes())

	evm := vm.EVM{
		StateDB:   statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
		TxContext: vm.TxContext{Origin: targetEvmAddress},
	}

	happyPathOutputNoMapping, _ := method.Outputs.Pack(targetCosmosAddressNoMapping.String())
	happyPathOutput, _ := method.Outputs.Pack(targetCosmosAddress.String())

	type args struct {
		evm      *vm.EVM
		caller   common.Address
		value    *big.Int
		readOnly bool
		hookFn   func()
	}
	tests := []struct {
		name       string
		args       args
		wantRet    []byte
		wantErr    bool
		wantErrMsg string
		wrongRet   bool
	}{
		{
			name: "happy path - no evm mapping",
			args: args{
				evm:    &evm,
				caller: targetEvmAddress,
				value:  big.NewInt(0),
				hookFn: func() {},
			},
			wantRet: happyPathOutputNoMapping,
			wantErr: false,
		},
		{
			name: "happy path - with evm mapping",
			args: args{
				evm:    &evm,
				caller: targetEvmAddress,
				value:  big.NewInt(0),
				hookFn: func() {
					tApp.EvmKeeper.SetAddressMapping(ctx, targetCosmosAddress, targetEvmAddress)
				},
			},
			wantRet: happyPathOutput,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the precompile and inputs
			p, err := addr.NewContract(tApp.EvmKeeper)
			require.Nil(t, err)
			inputs, err := method.Inputs.Pack(tt.args.caller)
			require.Nil(t, err)

			// call hook before testing
			tt.args.hookFn()

			// Make the call to associate.
			ret, _, err := p.Run(tt.args.evm, tt.args.caller, tt.args.caller,
				append(method.ID, inputs...),
				suppliedGas,
				tt.args.readOnly,
				tt.args.value,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v %v", err, tt.wantErr, string(ret))
				return
			}
			if err != nil {
				require.Equal(t, tt.wantErrMsg, err.Error())
			} else if tt.wrongRet {
				// tt.wrongRet is set if we expect a return value that's different from the happy path. This means that the wrong addresses were associated.
				require.NotEqual(t, tt.wantRet, ret)
			} else {
				require.Equal(t, tt.wantRet, ret)
			}
		})
	}
}

func TestGetEvmAddr(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	method := addr.ABI.Methods[addr.GetEvmAddressMethod]

	targetPrivKey := MockPrivateKey()
	targetCosmosAddress, targetEvmAddress := PrivateKeyToAddresses(targetPrivKey)

	evm := vm.EVM{
		StateDB:   statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
		TxContext: vm.TxContext{Origin: targetEvmAddress},
	}

	happyPathOutput, _ := method.Outputs.Pack(targetEvmAddress)

	type args struct {
		evm      *vm.EVM
		caller   common.Address
		value    *big.Int
		readOnly bool
		hookFn   func()
	}
	tests := []struct {
		name       string
		args       args
		wantRet    []byte
		wantErr    bool
		wantErrMsg string
		wrongRet   bool
	}{
		{
			name: "happy path - no evm mapping",
			args: args{
				evm:    &evm,
				caller: targetEvmAddress,
				value:  big.NewInt(0),
				hookFn: func() {},
			},
			wantErrMsg: fmt.Errorf("cosmos address %s is not associated\n", targetCosmosAddress).Error(),
			wantErr:    true,
		},
		{
			name: "happy path - with evm mapping",
			args: args{
				evm:    &evm,
				caller: targetEvmAddress,
				value:  big.NewInt(0),
				hookFn: func() {
					tApp.EvmKeeper.SetAddressMapping(ctx, targetCosmosAddress, targetEvmAddress)
				},
			},
			wantRet: happyPathOutput,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the precompile and inputs
			p, err := addr.NewContract(tApp.EvmKeeper)
			require.Nil(t, err)
			inputs, err := method.Inputs.Pack(targetCosmosAddress.String())
			require.Nil(t, err)

			// call hook before testing
			tt.args.hookFn()

			// Make the call to associate.
			ret, _, err := p.Run(tt.args.evm, tt.args.caller, tt.args.caller,
				append(method.ID, inputs...),
				suppliedGas,
				tt.args.readOnly,
				tt.args.value,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v %v", err, tt.wantErr, string(ret))
				return
			}
			if err != nil {
				require.Equal(t, tt.wantErrMsg, err.Error())
			} else if tt.wrongRet {
				// tt.wrongRet is set if we expect a return value that's different from the happy path. This means that the wrong addresses were associated.
				require.NotEqual(t, tt.wantRet, ret)
			} else {
				require.Equal(t, tt.wantRet, ret)
			}
		})
	}
}

func TestAssociatePubKey(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	method := addr.ABI.Methods[addr.AssociatePubKeyMethod]

	// Target refers to the address that the caller is trying to associate.
	targetPrivKey := MockPrivateKey()
	targetPubKey := targetPrivKey.PubKey()
	targetPubKeyHex := hex.EncodeToString(targetPubKey.Bytes())
	targetCosmosAddress, targetEvmAddress := PrivateKeyToAddresses(targetPrivKey)

	// Caller refers to the party calling the precompile.
	callerPrivKey := MockPrivateKey()
	_, callerEvmAddress := PrivateKeyToAddresses(callerPrivKey)

	evm := vm.EVM{
		StateDB:   statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
		TxContext: vm.TxContext{Origin: callerEvmAddress},
	}

	happyPathOutput, _ := method.Outputs.Pack(targetCosmosAddress.String(), targetEvmAddress)

	type args struct {
		evm      *vm.EVM
		caller   common.Address
		pubKey   string
		value    *big.Int
		readOnly bool
	}
	tests := []struct {
		name       string
		args       args
		wantRet    []byte
		wantErr    bool
		wantErrMsg string
		wrongRet   bool
	}{
		{
			name: "fails if payable",
			args: args{
				evm:    &evm,
				caller: callerEvmAddress,
				pubKey: targetPubKeyHex,
				value:  big.NewInt(10),
			},
			wantErr:    true,
			wantErrMsg: "sending funds to a non-payable function",
		},
		{
			name: "fails on static call",
			args: args{
				evm:      &evm,
				caller:   callerEvmAddress,
				pubKey:   targetPubKeyHex,
				value:    big.NewInt(10),
				readOnly: true,
			},
			wantErr:    true,
			wantErrMsg: "cannot call associate pub key precompile from staticcall",
		},
		{
			name: "fails if input is appended with 0x",
			args: args{
				evm:    &evm,
				caller: callerEvmAddress,
				pubKey: fmt.Sprintf("0x%v", targetPubKeyHex),
				value:  big.NewInt(0),
			},
			wantErr:    true,
			wantErrMsg: "encoding/hex: invalid byte: U+0078 'x'",
		},
		{
			name: "fails if caller address does not match with public key",
			args: args{
				evm:    &evm,
				caller: callerEvmAddress,
				pubKey: targetPubKeyHex,
				value:  big.NewInt(0),
			},
			wantErrMsg: fmt.Errorf("Caller address %s does not match with EVM address %s computed from the public key %s\n", callerEvmAddress.Hex(), targetEvmAddress.Hex(), base64.StdEncoding.EncodeToString(targetPubKey.Bytes())).Error(),
			wantErr:    true,
		},
		{
			name: "happy path - associates addresses if signature is correct",
			args: args{
				evm:    &evm,
				caller: targetEvmAddress,
				pubKey: targetPubKeyHex,
				value:  big.NewInt(0),
			},
			wantRet: happyPathOutput,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the precompile and inputs
			p, err := addr.NewContract(tApp.EvmKeeper)
			require.Nil(t, err)
			inputs, err := method.Inputs.Pack(tt.args.pubKey)
			require.Nil(t, err)

			// Make the call to associate.
			ret, _, err := p.Run(tt.args.evm, tt.args.caller, tt.args.caller,
				append(method.ID, inputs...),
				suppliedGas,
				tt.args.readOnly,
				tt.args.value,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v %v", err, tt.wantErr, string(ret))
				return
			}
			if err != nil {
				require.Equal(t, tt.wantErrMsg, err.Error())
			} else if tt.wrongRet {
				// tt.wrongRet is set if we expect a return value that's different from the happy path. This means that the wrong addresses were associated.
				require.NotEqual(t, tt.wantRet, ret)
			} else {
				require.Equal(t, tt.wantRet, ret)
				mappedCosmosAddress := tApp.EvmKeeper.GetCosmosAddressMapping(ctx, targetEvmAddress)
				require.Equal(t, targetCosmosAddress, mappedCosmosAddress)
				mappedEvmAddress, err := tApp.EvmKeeper.GetEvmAddressMapping(ctx, targetCosmosAddress)
				require.NoError(t, err)
				require.Equal(t, &targetEvmAddress, mappedEvmAddress)
			}
		})
	}
}

func TestAssociate(t *testing.T) {
	tApp := app.Setup(t)
	ctx := tApp.NewContext(true)

	method := addr.ABI.Methods[addr.AssociateMethod]

	// Target refers to the address that the caller is trying to associate.
	targetPrivKey := MockPrivateKey()
	targetPrivHex := hex.EncodeToString(targetPrivKey.Bytes())
	targetCosmosAddress, targetEvmAddress := PrivateKeyToAddresses(targetPrivKey)
	targetKey, _ := crypto.HexToECDSA(targetPrivHex)

	// Create the inputs
	emptyData := make([]byte, 32)
	prefixedMessage := fmt.Sprintf("\x19Ethereum Signed Message:\n%d", len(emptyData)) + string(emptyData)
	hash := crypto.Keccak256Hash([]byte(prefixedMessage))
	sig, err := crypto.Sign(hash.Bytes(), targetKey)
	require.Nil(t, err)

	r := fmt.Sprintf("0x%v", new(big.Int).SetBytes(sig[:32]).Text(16))
	s := fmt.Sprintf("0x%v", new(big.Int).SetBytes(sig[32:64]).Text(16))
	v := fmt.Sprintf("0x%v", new(big.Int).SetBytes([]byte{sig[64]}).Text(16))

	// Caller refers to the party calling the precompile.
	callerPrivKey := MockPrivateKey()
	_, callerEvmAddress := PrivateKeyToAddresses(callerPrivKey)

	evm := vm.EVM{
		StateDB:   statedb.New(ctx, tApp.EvmKeeper, statedb.NewEmptyTxConfig(common.BytesToHash(ctx.HeaderHash()))),
		TxContext: vm.TxContext{Origin: callerEvmAddress},
	}

	happyPathOutput, _ := method.Outputs.Pack(targetCosmosAddress.String(), targetEvmAddress)

	type args struct {
		evm      *vm.EVM
		caller   common.Address
		v        string
		r        string
		s        string
		msg      string
		value    *big.Int
		readOnly bool
	}
	tests := []struct {
		name       string
		args       args
		wantRet    []byte
		wantErr    bool
		wantErrMsg string
		wrongRet   bool
	}{
		{
			name: "fails if payable",
			args: args{
				evm:    &evm,
				caller: callerEvmAddress,
				v:      v,
				r:      r,
				s:      s,
				msg:    prefixedMessage,
				value:  big.NewInt(10),
			},
			wantErr:    true,
			wantErrMsg: "sending funds to a non-payable function",
		},
		{
			name: "fails on static calls",
			args: args{
				evm:      &evm,
				caller:   callerEvmAddress,
				v:        v,
				r:        r,
				s:        s,
				msg:      prefixedMessage,
				value:    big.NewInt(10),
				readOnly: true,
			},
			wantErr:    true,
			wantErrMsg: "cannot call associate precompile from staticcall",
		},
		{
			name: "fails if input is not hex",
			args: args{
				evm:    &evm,
				caller: callerEvmAddress,
				v:      "nothex",
				r:      r,
				s:      s,
				msg:    prefixedMessage,
				value:  big.NewInt(0),
			},
			wantErr:    true,
			wantErrMsg: "encoding/hex: invalid byte: U+006E 'n'",
		},
		{
			name: "associates wrong address if invalid signature (different message)",
			args: args{
				evm:    &evm,
				caller: callerEvmAddress,
				v:      v,
				r:      r,
				s:      s, // Pass in r instead of s here for invalid value
				msg:    prefixedMessage,
				value:  big.NewInt(0),
			},
			wantErrMsg: fmt.Errorf("Caller address %s does not match with EVM address %s computed from the public key %s\n", callerEvmAddress.Hex(), targetEvmAddress.Hex(), base64.StdEncoding.EncodeToString(targetPrivKey.PubKey().Bytes())).Error(),
			wantErr:    true,
		},
		{
			name: "happy path - associates addresses if signature is correct",
			args: args{
				evm:    &evm,
				caller: targetEvmAddress,
				v:      v,
				r:      r,
				s:      s,
				msg:    prefixedMessage,
				value:  big.NewInt(0),
			},
			wantRet: happyPathOutput,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the precompile and inputs
			p, _ := addr.NewContract(tApp.EvmKeeper)
			require.Nil(t, err)
			inputs, err := method.Inputs.Pack(tt.args.v, tt.args.r, tt.args.s, tt.args.msg)
			require.Nil(t, err)

			// Make the call to associate.
			ret, _, err := p.Run(tt.args.evm, tt.args.caller, tt.args.caller,
				append(method.ID, inputs...),
				suppliedGas,
				tt.args.readOnly,
				tt.args.value,
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("Run() error = %v, wantErr %v %v", err, tt.wantErr, string(ret))
				return
			}
			if err != nil {
				require.Equal(t, tt.wantErrMsg, err.Error())
			} else if tt.wrongRet {
				// tt.wrongRet is set if we expect a return value that's different from the happy path. This means that the wrong addresses were associated.
				require.NotEqual(t, tt.wantRet, ret)
			} else {
				require.Equal(t, tt.wantRet, ret)
				require.Equal(t, tt.wantRet, ret)
				mappedCosmosAddress := tApp.EvmKeeper.GetCosmosAddressMapping(ctx, targetEvmAddress)
				require.Equal(t, targetCosmosAddress, mappedCosmosAddress)
				mappedEvmAddress, err := tApp.EvmKeeper.GetEvmAddressMapping(ctx, targetCosmosAddress)
				require.NoError(t, err)
				require.Equal(t, &targetEvmAddress, mappedEvmAddress)
			}
		})
	}
}
