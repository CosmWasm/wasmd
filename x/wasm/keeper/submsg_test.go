package keeper

import (
	"encoding/json"
	"fmt"
	"strconv"
	"testing"

	wasmvm "github.com/CosmWasm/wasmvm/v2"
	wasmvmtypes "github.com/CosmWasm/wasmvm/v2/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/CosmWasm/wasmd/x/wasm/keeper/testdata"
	"github.com/CosmWasm/wasmd/x/wasm/keeper/wasmtesting"
	"github.com/CosmWasm/wasmd/x/wasm/types"
)

// test handing of submessages, very closely related to the reflect_test

// Try a simple send, no gas limit to for a sanity check before trying table tests
func TestDispatchSubMsgSuccessCase(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, ReflectCapabilities)
	accKeeper, keeper, bankKeeper := keepers.AccountKeeper, keepers.WasmKeeper, keepers.BankKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	contractStart := sdk.NewCoins(sdk.NewInt64Coin("denom", 40000))

	creator := keepers.Faucet.NewFundedRandomAccount(ctx, deposit...)
	creatorBalance := deposit.Sub(contractStart...)
	_, fred := keyPubAddr()

	// upload code
	codeID, _, err := keepers.ContractKeeper.Create(ctx, creator, testdata.ReflectContractWasm(), nil)
	require.NoError(t, err)
	require.Equal(t, uint64(1), codeID)

	// creator instantiates a contract and gives it tokens
	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, codeID, creator, nil, []byte("{}"), "reflect contract 1", contractStart)
	require.NoError(t, err)
	require.NotEmpty(t, contractAddr)

	// check some account values
	checkAccount(t, ctx, accKeeper, bankKeeper, contractAddr, contractStart)
	checkAccount(t, ctx, accKeeper, bankKeeper, creator, creatorBalance)
	checkAccount(t, ctx, accKeeper, bankKeeper, fred, nil)

	// creator can send contract's tokens to fred (using SendMsg)
	msg := wasmvmtypes.CosmosMsg{
		Bank: &wasmvmtypes.BankMsg{
			Send: &wasmvmtypes.SendMsg{
				ToAddress: fred.String(),
				Amount: []wasmvmtypes.Coin{{
					Denom:  "denom",
					Amount: "15000",
				}},
			},
		},
	}
	reflectSend := testdata.ReflectHandleMsg{
		ReflectSubMsg: &testdata.ReflectSubPayload{
			Msgs: []wasmvmtypes.SubMsg{{
				ID:      7,
				Msg:     msg,
				ReplyOn: wasmvmtypes.ReplyAlways,
			}},
		},
	}
	reflectSendBz, err := json.Marshal(reflectSend)
	require.NoError(t, err)
	_, err = keepers.ContractKeeper.Execute(ctx, contractAddr, creator, reflectSendBz, nil)
	require.NoError(t, err)

	// fred got coins
	checkAccount(t, ctx, accKeeper, bankKeeper, fred, sdk.NewCoins(sdk.NewInt64Coin("denom", 15000)))
	// contract lost them
	checkAccount(t, ctx, accKeeper, bankKeeper, contractAddr, sdk.NewCoins(sdk.NewInt64Coin("denom", 25000)))
	checkAccount(t, ctx, accKeeper, bankKeeper, creator, creatorBalance)

	// query the reflect state to ensure the result was stored
	query := testdata.ReflectQueryMsg{
		SubMsgResult: &testdata.SubCall{ID: 7},
	}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)
	queryRes, err := keeper.QuerySmart(ctx, contractAddr, queryBz)
	require.NoError(t, err)

	var res wasmvmtypes.Reply
	err = json.Unmarshal(queryRes, &res)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), res.ID)
	assert.Empty(t, res.Result.Err)
	require.NotNil(t, res.Result.Ok)
	sub := res.Result.Ok
	assert.Empty(t, sub.Data)
	// as of v0.28.0 we strip out all events that don't come from wasm contracts. can't trust the sdk.
	require.Len(t, sub.Events, 0)
}

func setupSubMsgTest(t *testing.T) (sdk.Context, TestKeepers, sdk.AccAddress, []sdk.Coin) {
	ctx, keepers := CreateTestInput(t, false, ReflectCapabilities)
	ctx = ctx.WithGasMeter(storetypes.NewInfiniteGasMeter())
	ctx = ctx.WithBlockGasMeter(storetypes.NewInfiniteGasMeter())

	contractStart := sdk.NewCoins(sdk.NewInt64Coin("funds", 1_000_000))
	uploader := keepers.Faucet.NewFundedRandomAccount(ctx, contractStart.Add(contractStart...)...)

	return ctx, keepers, uploader, contractStart
}

func TestDispatchSubMsgValidSend(t *testing.T) {
	ctx, keepers, uploader, contractStart := setupSubMsgTest(t)

	// upload code and instantiate
	reflectID, _, err := keepers.ContractKeeper.Create(ctx, uploader, testdata.ReflectContractWasm(), nil)
	require.NoError(t, err)

	creator := keepers.Faucet.NewFundedRandomAccount(ctx, contractStart...)
	_, empty := keyPubAddr()

	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, reflectID, creator, nil, []byte("{}"), "valid send test", contractStart)
	require.NoError(t, err)

	// test valid send
	msg := validBankSend(contractAddr.String(), empty.String(), "funds", 500_000)
	resp := executeSubmsg(t, ctx, keepers, contractAddr, creator, msg, 5, nil)

	require.Empty(t, resp.Result.Err)
	require.NotNil(t, resp.Result.Ok)
	require.Empty(t, resp.Result.Ok.Events)
	assertGasUsed(t, ctx, 110_000, 112_000)
}

func TestDispatchSubMsgInvalidSend(t *testing.T) {
	ctx, keepers, uploader, contractStart := setupSubMsgTest(t)

	// upload code and instantiate
	reflectID, _, err := keepers.ContractKeeper.Create(ctx, uploader, testdata.ReflectContractWasm(), nil)
	require.NoError(t, err)

	creator := keepers.Faucet.NewFundedRandomAccount(ctx, contractStart...)
	_, empty := keyPubAddr()

	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, reflectID, creator, nil, []byte("{}"), "invalid send test", contractStart)
	require.NoError(t, err)

	// test invalid send
	msg := invalidBankSend(contractAddr.String(), empty.String(), "funds", 2_000_000)
	resp := executeSubmsg(t, ctx, keepers, contractAddr, creator, msg, 6, nil)

	require.NotEmpty(t, resp.Result.Err)
	require.Nil(t, resp.Result.Ok)
	require.Contains(t, resp.Result.Err, "codespace: sdk, code: 5")
	assertGasUsed(t, ctx, 78_000, 81_000)
}

func TestDispatchSubMsgWithGasLimit(t *testing.T) {
	ctx, keepers, uploader, contractStart := setupSubMsgTest(t)
	subGasLimit := uint64(300_000)

	// upload code and instantiate
	reflectID, _, err := keepers.ContractKeeper.Create(ctx, uploader, testdata.ReflectContractWasm(), nil)
	require.NoError(t, err)

	creator := keepers.Faucet.NewFundedRandomAccount(ctx, contractStart...)
	_, empty := keyPubAddr()

	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, reflectID, creator, nil, []byte("{}"), "gas limit test", contractStart)
	require.NoError(t, err)

	// test with gas limit
	msg := validBankSend(contractAddr.String(), empty.String(), "funds", 500_000)
	resp := executeSubmsg(t, ctx, keepers, contractAddr, creator, msg, 15, &subGasLimit)

	require.Empty(t, resp.Result.Err)
	require.NotNil(t, resp.Result.Ok)
	require.Empty(t, resp.Result.Ok.Events)
	assertGasUsed(t, ctx, 110_000, 112_000)
}

func validBankSend(_, emptyAccount, denom string, amount int) wasmvmtypes.CosmosMsg {
	return wasmvmtypes.CosmosMsg{
		Bank: &wasmvmtypes.BankMsg{
			Send: &wasmvmtypes.SendMsg{
				ToAddress: emptyAccount,
				Amount: []wasmvmtypes.Coin{{
					Denom:  denom,
					Amount: strconv.Itoa(amount),
				}},
			},
		},
	}
}

func invalidBankSend(contract, emptyAccount, denom string, amount int) wasmvmtypes.CosmosMsg {
	return validBankSend(contract, emptyAccount, denom, amount)
}

func executeSubmsg(t *testing.T, ctx sdk.Context, keepers TestKeepers, contractAddr, creator sdk.AccAddress, msg wasmvmtypes.CosmosMsg, submsgID uint64, gasLimit *uint64) wasmvmtypes.Reply {
	reflectSend := testdata.ReflectHandleMsg{
		ReflectSubMsg: &testdata.ReflectSubPayload{
			Msgs: []wasmvmtypes.SubMsg{{
				ID:       submsgID,
				Msg:      msg,
				GasLimit: gasLimit,
				ReplyOn:  wasmvmtypes.ReplyAlways,
			}},
		},
	}
	reflectSendBz, err := json.Marshal(reflectSend)
	require.NoError(t, err)

	_, err = keepers.ContractKeeper.Execute(ctx, contractAddr, creator, reflectSendBz, nil)
	require.NoError(t, err)

	// query the reply
	query := testdata.ReflectQueryMsg{
		SubMsgResult: &testdata.SubCall{ID: submsgID},
	}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)
	queryRes, err := keepers.WasmKeeper.QuerySmart(ctx, contractAddr, queryBz)
	require.NoError(t, err)

	var res wasmvmtypes.Reply
	err = json.Unmarshal(queryRes, &res)
	require.NoError(t, err)
	assert.Equal(t, submsgID, res.ID)

	return res
}

func assertGasUsed(t *testing.T, ctx sdk.Context, minGas, maxGas uint64) {
	gasUsed := ctx.GasMeter().GasConsumed()
	assert.True(t, gasUsed >= minGas, "Used %d gas (less than expected %d)", gasUsed, minGas)
	assert.True(t, gasUsed <= maxGas, "Used %d gas (more than expected %d)", gasUsed, maxGas)
}

// Test an error case, where the Encoded doesn't return any sdk.Msg and we trigger(ed) a null pointer exception.
// This occurs with the IBC encoder. Test this.
func TestDispatchSubMsgEncodeToNoSdkMsg(t *testing.T) {
	// fake out the bank handle to return success with no data
	nilEncoder := func(sender sdk.AccAddress, msg *wasmvmtypes.BankMsg) ([]sdk.Msg, error) {
		return nil, nil
	}
	customEncoders := &MessageEncoders{
		Bank: nilEncoder,
	}

	ctx, keepers := CreateTestInput(t, false, ReflectCapabilities, WithMessageHandler(NewSDKMessageHandler(MakeTestCodec(t), nil, customEncoders)))
	keeper := keepers.WasmKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	contractStart := sdk.NewCoins(sdk.NewInt64Coin("denom", 40000))

	creator := keepers.Faucet.NewFundedRandomAccount(ctx, deposit...)
	_, fred := keyPubAddr()

	// upload code
	codeID, _, err := keepers.ContractKeeper.Create(ctx, creator, testdata.ReflectContractWasm(), nil)
	require.NoError(t, err)

	// creator instantiates a contract and gives it tokens
	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, codeID, creator, nil, []byte("{}"), "reflect contract 1", contractStart)
	require.NoError(t, err)
	require.NotEmpty(t, contractAddr)

	// creator can send contract's tokens to fred (using SendMsg)
	msg := wasmvmtypes.CosmosMsg{
		Bank: &wasmvmtypes.BankMsg{
			Send: &wasmvmtypes.SendMsg{
				ToAddress: fred.String(),
				Amount: []wasmvmtypes.Coin{{
					Denom:  "denom",
					Amount: "15000",
				}},
			},
		},
	}
	reflectSend := testdata.ReflectHandleMsg{
		ReflectSubMsg: &testdata.ReflectSubPayload{
			Msgs: []wasmvmtypes.SubMsg{{
				ID:      7,
				Msg:     msg,
				ReplyOn: wasmvmtypes.ReplyAlways,
			}},
		},
	}
	reflectSendBz, err := json.Marshal(reflectSend)
	require.NoError(t, err)
	_, err = keepers.ContractKeeper.Execute(ctx, contractAddr, creator, reflectSendBz, nil)
	require.NoError(t, err)

	// query the reflect state to ensure the result was stored
	query := testdata.ReflectQueryMsg{
		SubMsgResult: &testdata.SubCall{ID: 7},
	}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)
	queryRes, err := keeper.QuerySmart(ctx, contractAddr, queryBz)
	require.NoError(t, err)

	var res wasmvmtypes.Reply
	err = json.Unmarshal(queryRes, &res)
	require.NoError(t, err)
	assert.Equal(t, uint64(7), res.ID)
	assert.Empty(t, res.Result.Err)
	require.NotNil(t, res.Result.Ok)
	sub := res.Result.Ok
	assert.Empty(t, sub.Data)
	require.Len(t, sub.Events, 0)
}

// Try a simple send, no gas limit to for a sanity check before trying table tests
func TestDispatchSubMsgConditionalReplyOn(t *testing.T) {
	ctx, keepers := CreateTestInput(t, false, ReflectCapabilities)
	keeper := keepers.WasmKeeper

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
	contractStart := sdk.NewCoins(sdk.NewInt64Coin("denom", 40000))

	creator := keepers.Faucet.NewFundedRandomAccount(ctx, deposit...)
	_, fred := keyPubAddr()

	// upload code
	codeID, _, err := keepers.ContractKeeper.Create(ctx, creator, testdata.ReflectContractWasm(), nil)
	require.NoError(t, err)

	// creator instantiates a contract and gives it tokens
	contractAddr, _, err := keepers.ContractKeeper.Instantiate(ctx, codeID, creator, nil, []byte("{}"), "reflect contract 1", contractStart)
	require.NoError(t, err)

	goodSend := wasmvmtypes.CosmosMsg{
		Bank: &wasmvmtypes.BankMsg{
			Send: &wasmvmtypes.SendMsg{
				ToAddress: fred.String(),
				Amount: []wasmvmtypes.Coin{{
					Denom:  "denom",
					Amount: "1000",
				}},
			},
		},
	}
	failSend := wasmvmtypes.CosmosMsg{
		Bank: &wasmvmtypes.BankMsg{
			Send: &wasmvmtypes.SendMsg{
				ToAddress: fred.String(),
				Amount: []wasmvmtypes.Coin{{
					Denom:  "no-such-token",
					Amount: "777777",
				}},
			},
		},
	}

	cases := map[string]struct {
		// true for wasmvmtypes.ReplySuccess, false for wasmvmtypes.ReplyError
		replyOnSuccess bool
		msg            wasmvmtypes.CosmosMsg
		// true if the call should return an error (it wasn't handled)
		expectError bool
		// true if the reflect contract wrote the response (success or error) - it was captured
		writeResult bool
	}{
		"all good, reply success": {
			replyOnSuccess: true,
			msg:            goodSend,
			expectError:    false,
			writeResult:    true,
		},
		"all good, reply error": {
			replyOnSuccess: false,
			msg:            goodSend,
			expectError:    false,
			writeResult:    false,
		},
		"bad msg, reply success": {
			replyOnSuccess: true,
			msg:            failSend,
			expectError:    true,
			writeResult:    false,
		},
		"bad msg, reply error": {
			replyOnSuccess: false,
			msg:            failSend,
			expectError:    false,
			writeResult:    true,
		},
	}

	var id uint64
	for name, tc := range cases {
		id++
		t.Run(name, func(t *testing.T) {
			subMsg := wasmvmtypes.SubMsg{
				ID:      id,
				Msg:     tc.msg,
				ReplyOn: wasmvmtypes.ReplySuccess,
			}
			if !tc.replyOnSuccess {
				subMsg.ReplyOn = wasmvmtypes.ReplyError
			}

			reflectSend := testdata.ReflectHandleMsg{
				ReflectSubMsg: &testdata.ReflectSubPayload{
					Msgs: []wasmvmtypes.SubMsg{subMsg},
				},
			}
			reflectSendBz, err := json.Marshal(reflectSend)
			require.NoError(t, err)
			_, err = keepers.ContractKeeper.Execute(ctx, contractAddr, creator, reflectSendBz, nil)

			if tc.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// query the reflect state to check if the result was stored
			query := testdata.ReflectQueryMsg{
				SubMsgResult: &testdata.SubCall{ID: id},
			}
			queryBz, err := json.Marshal(query)
			require.NoError(t, err)
			queryRes, err := keeper.QuerySmart(ctx, contractAddr, queryBz)
			if tc.writeResult {
				// we got some data for this call
				require.NoError(t, err)
				var res wasmvmtypes.Reply
				err = json.Unmarshal(queryRes, &res)
				require.NoError(t, err)
				require.Equal(t, id, res.ID)
			} else {
				// nothing should be there -> error
				require.Error(t, err)
			}
		})
	}
}

func TestInstantiateGovSubMsgAuthzPropagated(t *testing.T) {
	mockWasmVM := &wasmtesting.MockWasmEngine{}
	wasmtesting.MakeInstantiable(mockWasmVM)
	var instanceLevel int
	// mock wasvm to return new instantiate msgs with the response
	mockWasmVM.InstantiateFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, info wasmvmtypes.MessageInfo, initMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
		if instanceLevel == 2 {
			return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{}}, 0, nil
		}
		instanceLevel++
		submsgPayload := fmt.Sprintf(`{"sub":%d}`, instanceLevel)
		return &wasmvmtypes.ContractResult{
			Ok: &wasmvmtypes.Response{
				Messages: []wasmvmtypes.SubMsg{
					{
						ReplyOn: wasmvmtypes.ReplyNever,
						Msg: wasmvmtypes.CosmosMsg{
							Wasm: &wasmvmtypes.WasmMsg{Instantiate: &wasmvmtypes.InstantiateMsg{
								CodeID: 1, Msg: []byte(submsgPayload), Label: "from sub-msg",
							}},
						},
					},
				},
			},
		}, 0, nil
	}

	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities, WithWasmEngine(mockWasmVM))
	k := keepers.WasmKeeper

	// make chain restricted so that nobody can create instances
	newParams := types.DefaultParams()
	newParams.InstantiateDefaultPermission = types.AccessTypeNobody
	require.NoError(t, k.SetParams(ctx, newParams))

	example1 := StoreRandomContract(t, ctx, keepers, mockWasmVM)

	specs := map[string]struct {
		policy types.AuthorizationPolicy
		expErr *errorsmod.Error
	}{
		"default policy - rejected": {
			policy: DefaultAuthorizationPolicy{},
			expErr: sdkerrors.ErrUnauthorized,
		},
		"propagating gov policy - accepted": {
			policy: newGovAuthorizationPolicy(map[types.AuthorizationPolicyAction]struct{}{
				types.AuthZActionInstantiate: {},
			}),
		},
		"non propagating gov policy - rejected in sub-msg": {
			policy: newGovAuthorizationPolicy(nil),
			expErr: sdkerrors.ErrUnauthorized,
		},
		"propagating gov policy with diff action - rejected": {
			policy: newGovAuthorizationPolicy(map[types.AuthorizationPolicyAction]struct{}{
				types.AuthZActionMigrateContract: {},
			}),
			expErr: sdkerrors.ErrUnauthorized,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			tCtx, _ := ctx.CacheContext()
			instanceLevel = 0

			_, _, gotErr := k.instantiate(tCtx, example1.CodeID, example1.CreatorAddr, nil, []byte(`{"first":{}}`), "from ext msg", nil, k.ClassicAddressGenerator(), spec.policy)
			if spec.expErr != nil {
				assert.ErrorIs(t, gotErr, spec.expErr)
				return
			}
			require.NoError(t, gotErr)
			var instanceCount int
			k.IterateContractsByCode(tCtx, example1.CodeID, func(address sdk.AccAddress) bool {
				instanceCount++
				return false
			})
			assert.Equal(t, 3, instanceCount)
			assert.Equal(t, 2, instanceLevel)
		})
	}
}

func TestMigrateGovSubMsgAuthzPropagated(t *testing.T) {
	mockWasmVM := &wasmtesting.MockWasmEngine{}
	wasmtesting.MakeInstantiable(mockWasmVM)
	ctx, keepers := CreateTestInput(t, false, AvailableCapabilities, WithWasmEngine(mockWasmVM))
	k := keepers.WasmKeeper

	example1 := InstantiateHackatomExampleContract(t, ctx, keepers)
	example2 := InstantiateIBCReflectContract(t, ctx, keepers)

	var instanceLevel int
	// mock wasvm to return new migrate msgs with the response
	mockWasmVM.MigrateFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
		if instanceLevel == 1 {
			return &wasmvmtypes.ContractResult{Ok: &wasmvmtypes.Response{}}, 0, nil
		}
		instanceLevel++
		submsgPayload := fmt.Sprintf(`{"sub":%d}`, instanceLevel)
		return &wasmvmtypes.ContractResult{
			Ok: &wasmvmtypes.Response{
				Messages: []wasmvmtypes.SubMsg{
					{
						ReplyOn: wasmvmtypes.ReplyNever,
						Msg: wasmvmtypes.CosmosMsg{
							Wasm: &wasmvmtypes.WasmMsg{Migrate: &wasmvmtypes.MigrateMsg{
								ContractAddr: example1.Contract.String(),
								NewCodeID:    example2.CodeID,
								Msg:          []byte(submsgPayload),
							}},
						},
					},
				},
			},
		}, 0, nil
	}
	mockWasmVM.MigrateWithInfoFn = func(codeID wasmvm.Checksum, env wasmvmtypes.Env, migrateMsg []byte, migrateInfo wasmvmtypes.MigrateInfo, store wasmvm.KVStore, goapi wasmvm.GoAPI, querier wasmvm.Querier, gasMeter wasmvm.GasMeter, gasLimit uint64, deserCost wasmvmtypes.UFraction) (*wasmvmtypes.ContractResult, uint64, error) {
		return mockWasmVM.MigrateFn(codeID, env, migrateMsg, store, goapi, querier, gasMeter, gasLimit, deserCost)
	}

	specs := map[string]struct {
		policy types.AuthorizationPolicy
		expErr *errorsmod.Error
	}{
		"default policy - rejected": {
			policy: DefaultAuthorizationPolicy{},
			expErr: sdkerrors.ErrUnauthorized,
		},
		"propagating gov policy - accepted": {
			policy: newGovAuthorizationPolicy(map[types.AuthorizationPolicyAction]struct{}{
				types.AuthZActionMigrateContract: {},
			}),
		},
		"non propagating gov policy - rejected in sub-msg": {
			policy: newGovAuthorizationPolicy(nil),
			expErr: sdkerrors.ErrUnauthorized,
		},
		"propagating gov policy with diff action - rejected": {
			policy: newGovAuthorizationPolicy(map[types.AuthorizationPolicyAction]struct{}{
				types.AuthZActionInstantiate: {},
			}),
			expErr: sdkerrors.ErrUnauthorized,
		},
	}
	for name, spec := range specs {
		t.Run(name, func(t *testing.T) {
			tCtx, _ := ctx.CacheContext()
			instanceLevel = 0

			_, gotErr := k.migrate(tCtx, example1.Contract, RandomAccountAddress(t), example2.CodeID, []byte(`{}`), spec.policy)
			if spec.expErr != nil {
				assert.ErrorIs(t, gotErr, spec.expErr)
				return
			}
			require.NoError(t, gotErr)
			assert.Equal(t, 1, instanceLevel)
		})
	}
}
