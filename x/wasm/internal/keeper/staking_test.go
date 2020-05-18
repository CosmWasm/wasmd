package keeper

import (
	"encoding/json"
	"fmt"
	"github.com/cosmos/cosmos-sdk/x/auth"
	"github.com/cosmos/cosmos-sdk/x/staking"
	"github.com/cosmos/cosmos-sdk/x/staking/types"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

type StakingInitMsg struct {
	Name      string         `json:"name"`
	Symbol    string         `json:"symbol"`
	Decimals  uint8          `json:"decimals"`
	Validator sdk.ValAddress `json:"validator"`
	ExitTax   sdk.Dec        `json:"exit_tax"`
	// MinWithdrawl is uint128 encoded as a string (use sdk.Int?)
	MinWithdrawl string `json:"min_withdrawl"`
}

// StakingHandleMsg is used to encode handle messages
type StakingHandleMsg struct {
	Transfer *transferPayload `json:"transfer,omitempty"`
	Bond     *struct{}        `json:"bond,omitempty"`
	Unbond   *unbondPayload   `json:"unbond,omitempty"`
	Claim    *struct{}        `json:"claim,omitempty"`
	Reinvest *struct{}        `json:"reinvest,omitempty"`
	Change   *ownerPayload    `json:"change_owner,omitempty"`
}

type transferPayload struct {
	Recipient sdk.Address `json:"recipient"`
	// uint128 encoded as string
	Amount string `json:"amount"`
}

type unbondPayload struct {
	// uint128 encoded as string
	Amount string `json:"amount"`
}

// StakingQueryMsg is used to encode query messages
type StakingQueryMsg struct {
	Balance    *addressQuery `json:"balance,omitempty"`
	Claims     *addressQuery `json:"claims,omitempty"`
	TokenInfo  *struct{}     `json:"token_info,omitempty"`
	Investment *struct{}     `json:"investment,omitempty"`
}

type addressQuery struct {
	Address sdk.AccAddress `json:"address"`
}

type BalanceResponse struct {
	Balance string `json:"balance,omitempty"`
}

type ClaimsResponse struct {
	Claims string `json:"claims,omitempty"`
}

type TokenInfoResponse struct {
	Name     string `json:"name"`
	Symbol   string `json:"symbol"`
	Decimals uint8  `json:"decimals"`
}

type InvestmentResponse struct {
	TokenSupply  string         `json:"token_supply"`
	StakedTokens sdk.Coin       `json:"staked_tokens"`
	NominalValue sdk.Dec        `json:"nominal_value"`
	Owner        sdk.AccAddress `json:"owner"`
	Validator    sdk.ValAddress `json:"validator"`
	ExitTax      sdk.Dec        `json:"exit_tax"`
	// MinWithdrawl is uint128 encoded as a string (use sdk.Int?)
	MinWithdrawl string `json:"min_withdrawl"`
}

// adds a few validators and returns a list of validators that are registered
func addValidators(ctx sdk.Context, stakingKeeper staking.Keeper, powers []sdk.Int) []sdk.ValAddress {
	var addrs = make([]sdk.ValAddress, len(powers))
	for i, power := range powers {
		addrs[i] = addValidator(ctx, stakingKeeper, power)
	}
	return addrs
}

// adds a few validators and returns a list of validators that are registered
func addValidator(ctx sdk.Context, stakingKeeper staking.Keeper, power sdk.Int) sdk.ValAddress {
	_, pub, accAddr := keyPubAddr()
	addr := sdk.ValAddress(accAddr)
	// make it a bonded validator with power stake
	val := types.NewValidator(addr, pub, types.Description{Moniker: fmt.Sprintf("Validator power %s", power)})
	val.Status = sdk.Bonded
	val, _ = val.AddTokensFromDel(power)
	// store it
	stakingKeeper.SetValidator(ctx, val)
	stakingKeeper.SetValidatorByPowerIndex(ctx, val)
	return addr
}

func TestInitializeStaking(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)
	ctx, accKeeper, stakingKeeper, keeper := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)

	valAddr := addValidator(ctx, stakingKeeper, sdk.NewInt(1234567))
	v, found := stakingKeeper.GetValidator(ctx, valAddr)
	assert.True(t, found)
	assert.Equal(t, v.GetDelegatorShares(), sdk.NewDec(1234567))

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000), sdk.NewInt64Coin("stake", 500000))
	creator := createFakeFundedAccount(ctx, accKeeper, deposit)

	// upload staking derivates code
	stakingCode, err := ioutil.ReadFile("./testdata/staking.wasm")
	require.NoError(t, err)
	stakingID, err := keeper.Create(ctx, creator, stakingCode, "", "")
	require.NoError(t, err)
	require.Equal(t, uint64(1), stakingID)

	// register to a valid address
	initMsg := StakingInitMsg{
		Name:         "Staking Derivatives",
		Symbol:       "DRV",
		Decimals:     0,
		Validator:    valAddr,
		ExitTax:      sdk.MustNewDecFromStr("0.10"),
		MinWithdrawl: "100",
	}
	initBz, err := json.Marshal(&initMsg)
	require.NoError(t, err)

	stakingAddr, err := keeper.Instantiate(ctx, stakingID, creator, initBz, "staking derivates - DRV", nil)
	require.NoError(t, err)
	require.NotEmpty(t, stakingAddr)

	// nothing spent here
	checkAccount(t, ctx, accKeeper, creator, deposit)

	// try to register with a validator not on the list and it fails
	_, _, bob := keyPubAddr()
	badInitMsg := StakingInitMsg{
		Name:         "Missing Validator",
		Symbol:       "MISS",
		Decimals:     0,
		Validator:    sdk.ValAddress(bob),
		ExitTax:      sdk.MustNewDecFromStr("0.10"),
		MinWithdrawl: "100",
	}
	badBz, err := json.Marshal(&badInitMsg)
	require.NoError(t, err)

	_, err = keeper.Instantiate(ctx, stakingID, creator, badBz, "missing validator", nil)
	require.Error(t, err)

	// no changes to bonding shares
	val, _ := stakingKeeper.GetValidator(ctx, valAddr)
	assert.Equal(t, val.GetDelegatorShares(), sdk.NewDec(1234567))
}

type initInfo struct {
	valAddr      sdk.ValAddress
	creator      sdk.AccAddress
	contractAddr sdk.AccAddress

	ctx           sdk.Context
	accKeeper     auth.AccountKeeper
	stakingKeeper staking.Keeper
	wasmKeeper    Keeper

	cleanup func()
}

func initializeStaking(t *testing.T) initInfo {
	tempDir, err := ioutil.TempDir("", "wasm")
	require.NoError(t, err)
	ctx, accKeeper, stakingKeeper, keeper := CreateTestInput(t, false, tempDir, SupportedFeatures, nil, nil)

	valAddr := addValidator(ctx, stakingKeeper, sdk.NewInt(1234567))
	v, found := stakingKeeper.GetValidator(ctx, valAddr)
	assert.True(t, found)
	assert.Equal(t, v.GetDelegatorShares(), sdk.NewDec(1234567))

	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000), sdk.NewInt64Coin("stake", 500000))
	creator := createFakeFundedAccount(ctx, accKeeper, deposit)

	// upload staking derivates code
	stakingCode, err := ioutil.ReadFile("./testdata/staking.wasm")
	require.NoError(t, err)
	stakingID, err := keeper.Create(ctx, creator, stakingCode, "", "")
	require.NoError(t, err)
	require.Equal(t, uint64(1), stakingID)

	// register to a valid address
	initMsg := StakingInitMsg{
		Name:         "Staking Derivatives",
		Symbol:       "DRV",
		Decimals:     0,
		Validator:    valAddr,
		ExitTax:      sdk.MustNewDecFromStr("0.10"),
		MinWithdrawl: "100",
	}
	initBz, err := json.Marshal(&initMsg)
	require.NoError(t, err)

	stakingAddr, err := keeper.Instantiate(ctx, stakingID, creator, initBz, "staking derivates - DRV", nil)
	require.NoError(t, err)
	require.NotEmpty(t, stakingAddr)

	return initInfo{
		valAddr:       valAddr,
		creator:       creator,
		contractAddr:  stakingAddr,
		ctx:           ctx,
		accKeeper:     accKeeper,
		stakingKeeper: stakingKeeper,
		wasmKeeper:    keeper,
		cleanup:       func() { os.RemoveAll(tempDir) },
	}
}

func TestBonding(t *testing.T) {
	initInfo := initializeStaking(t)
	defer initInfo.cleanup()
	ctx, valAddr, contractAddr := initInfo.ctx, initInfo.valAddr, initInfo.contractAddr
	keeper, stakingKeeper, accKeeper := initInfo.wasmKeeper, initInfo.stakingKeeper, initInfo.accKeeper

	// initial checks of bonding state
	val, found := stakingKeeper.GetValidator(ctx, valAddr)
	require.True(t, found)
	initPower := val.GetDelegatorShares()

	// bob has 160k, putting 80k into the contract
	full := sdk.NewCoins(sdk.NewInt64Coin("stake", 160000))
	funds := sdk.NewCoins(sdk.NewInt64Coin("stake", 80000))
	bob := createFakeFundedAccount(ctx, accKeeper, full)

	// check contract state before
	assertBalance(t, ctx, keeper, contractAddr, bob, "0")
	assertClaims(t, ctx, keeper, contractAddr, bob, "0")
	assertSupply(t, ctx, keeper, contractAddr, "0", sdk.NewInt64Coin("stake", 0))

	bond := StakingHandleMsg{
		Bond: &struct{}{},
	}
	bondBz, err := json.Marshal(bond)
	require.NoError(t, err)
	_, err = keeper.Execute(ctx, contractAddr, bob, bondBz, funds)
	require.NoError(t, err)

	// check some account values - the money is on neither account (cuz it is bonded)
	checkAccount(t, ctx, accKeeper, contractAddr, sdk.Coins{})
	checkAccount(t, ctx, accKeeper, bob, funds)

	// make sure the proper number of tokens have been bonded
	val, _ = stakingKeeper.GetValidator(ctx, valAddr)
	finalPower := val.GetDelegatorShares()
	assert.Equal(t, sdk.NewInt(80000), finalPower.Sub(initPower).TruncateInt())

	// check we have the desired balance
	assertBalance(t, ctx, keeper, contractAddr, bob, "80000")
	assertClaims(t, ctx, keeper, contractAddr, bob, "0")
	assertSupply(t, ctx, keeper, contractAddr, "80000", sdk.NewInt64Coin("stake", 80000))
}

func assertBalance(t *testing.T, ctx sdk.Context, keeper Keeper, contract sdk.AccAddress, addr sdk.AccAddress, expected string) {
	query := StakingQueryMsg{
		Balance: &addressQuery{
			Address: addr,
		},
	}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)
	res, err := keeper.QuerySmart(ctx, contract, queryBz)
	require.NoError(t, err)
	var balance BalanceResponse
	err = json.Unmarshal(res, &balance)
	require.NoError(t, err)
	assert.Equal(t, expected, balance.Balance)
}

func assertClaims(t *testing.T, ctx sdk.Context, keeper Keeper, contract sdk.AccAddress, addr sdk.AccAddress, expected string) {
	query := StakingQueryMsg{
		Claims: &addressQuery{
			Address: addr,
		},
	}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)
	res, err := keeper.QuerySmart(ctx, contract, queryBz)
	require.NoError(t, err)
	var claims ClaimsResponse
	err = json.Unmarshal(res, &claims)
	require.NoError(t, err)
	assert.Equal(t, expected, claims.Claims)
}

func assertSupply(t *testing.T, ctx sdk.Context, keeper Keeper, contract sdk.AccAddress, expectedIssued string, expectedBonded sdk.Coin) {
	query := StakingQueryMsg{Investment: &struct{}{}}
	queryBz, err := json.Marshal(query)
	require.NoError(t, err)
	res, err := keeper.QuerySmart(ctx, contract, queryBz)
	require.NoError(t, err)
	var invest InvestmentResponse
	err = json.Unmarshal(res, &invest)
	require.NoError(t, err)
	assert.Equal(t, expectedIssued, invest.TokenSupply)
	assert.Equal(t, expectedBonded, invest.StakedTokens)
}

//func TestMaskReflectCustomMsg(t *testing.T) {
//	tempDir, err := ioutil.TempDir("", "wasm")
//	require.NoError(t, err)
//	defer os.RemoveAll(tempDir)
//	ctx, accKeeper, keeper := CreateTestInput(t, false, tempDir, MaskFeatures, maskEncoders(MakeTestCodec()), maskPlugins())
//
//	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
//	creator := createFakeFundedAccount(ctx, accKeeper, deposit)
//	bob := createFakeFundedAccount(ctx, accKeeper, deposit)
//	_, _, fred := keyPubAddr()
//
//	// upload code
//	maskCode, err := ioutil.ReadFile("./testdata/reflect.wasm")
//	require.NoError(t, err)
//	codeID, err := keeper.Create(ctx, creator, maskCode, "", "")
//	require.NoError(t, err)
//	require.Equal(t, uint64(1), codeID)
//
//	// creator instantiates a contract and gives it tokens
//	contractStart := sdk.NewCoins(sdk.NewInt64Coin("denom", 40000))
//	contractAddr, err := keeper.Instantiate(ctx, codeID, creator, []byte("{}"), "mask contract 1", contractStart)
//	require.NoError(t, err)
//	require.NotEmpty(t, contractAddr)
//
//	// set owner to bob
//	transfer := MaskHandleMsg{
//		Change: &ownerPayload{
//			Owner: bob,
//		},
//	}
//	transferBz, err := json.Marshal(transfer)
//	require.NoError(t, err)
//	_, err = keeper.Execute(ctx, contractAddr, creator, transferBz, nil)
//	require.NoError(t, err)
//
//	// check some account values
//	checkAccount(t, ctx, accKeeper, contractAddr, contractStart)
//	checkAccount(t, ctx, accKeeper, bob, deposit)
//	checkAccount(t, ctx, accKeeper, fred, nil)
//
//	// bob can send contract's tokens to fred (using SendMsg)
//	msgs := []wasmTypes.CosmosMsg{{
//		Bank: &wasmTypes.BankMsg{
//			Send: &wasmTypes.SendMsg{
//				FromAddress: contractAddr.String(),
//				ToAddress:   fred.String(),
//				Amount: []wasmTypes.Coin{{
//					Denom:  "denom",
//					Amount: "15000",
//				}},
//			},
//		},
//	}}
//	reflectSend := MaskHandleMsg{
//		Reflect: &reflectPayload{
//			Msgs: msgs,
//		},
//	}
//	reflectSendBz, err := json.Marshal(reflectSend)
//	require.NoError(t, err)
//	_, err = keeper.Execute(ctx, contractAddr, bob, reflectSendBz, nil)
//	require.NoError(t, err)
//
//	// fred got coins
//	checkAccount(t, ctx, accKeeper, fred, sdk.NewCoins(sdk.NewInt64Coin("denom", 15000)))
//	// contract lost them
//	checkAccount(t, ctx, accKeeper, contractAddr, sdk.NewCoins(sdk.NewInt64Coin("denom", 25000)))
//	checkAccount(t, ctx, accKeeper, bob, deposit)
//
//	// construct an opaque message
//	var sdkSendMsg sdk.Msg = &bank.MsgSend{
//		FromAddress: contractAddr,
//		ToAddress:   fred,
//		Amount:      sdk.NewCoins(sdk.NewInt64Coin("denom", 23000)),
//	}
//	opaque, err := toMaskRawMsg(keeper.cdc, sdkSendMsg)
//	require.NoError(t, err)
//	reflectOpaque := MaskHandleMsg{
//		Reflect: &reflectPayload{
//			Msgs: []wasmTypes.CosmosMsg{opaque},
//		},
//	}
//	reflectOpaqueBz, err := json.Marshal(reflectOpaque)
//	require.NoError(t, err)
//
//	_, err = keeper.Execute(ctx, contractAddr, bob, reflectOpaqueBz, nil)
//	require.NoError(t, err)
//
//	// fred got more coins
//	checkAccount(t, ctx, accKeeper, fred, sdk.NewCoins(sdk.NewInt64Coin("denom", 38000)))
//	// contract lost them
//	checkAccount(t, ctx, accKeeper, contractAddr, sdk.NewCoins(sdk.NewInt64Coin("denom", 2000)))
//	checkAccount(t, ctx, accKeeper, bob, deposit)
//}
//
//func TestMaskReflectCustomQuery(t *testing.T) {
//	tempDir, err := ioutil.TempDir("", "wasm")
//	require.NoError(t, err)
//	defer os.RemoveAll(tempDir)
//	ctx, accKeeper, keeper := CreateTestInput(t, false, tempDir, MaskFeatures, maskEncoders(MakeTestCodec()), maskPlugins())
//
//	deposit := sdk.NewCoins(sdk.NewInt64Coin("denom", 100000))
//	creator := createFakeFundedAccount(ctx, accKeeper, deposit)
//
//	// upload code
//	maskCode, err := ioutil.ReadFile("./testdata/reflect.wasm")
//	require.NoError(t, err)
//	codeID, err := keeper.Create(ctx, creator, maskCode, "", "")
//	require.NoError(t, err)
//	require.Equal(t, uint64(1), codeID)
//
//	// creator instantiates a contract and gives it tokens
//	contractStart := sdk.NewCoins(sdk.NewInt64Coin("denom", 40000))
//	contractAddr, err := keeper.Instantiate(ctx, codeID, creator, []byte("{}"), "mask contract 1", contractStart)
//	require.NoError(t, err)
//	require.NotEmpty(t, contractAddr)
//
//	// let's perform a normal query of state
//	ownerQuery := MaskQueryMsg{
//		Owner: &struct{}{},
//	}
//	ownerQueryBz, err := json.Marshal(ownerQuery)
//	require.NoError(t, err)
//	ownerRes, err := keeper.QuerySmart(ctx, contractAddr, ownerQueryBz)
//	require.NoError(t, err)
//	var res OwnerResponse
//	err = json.Unmarshal(ownerRes, &res)
//	require.NoError(t, err)
//	assert.Equal(t, res.Owner, creator.String())
//
//	// and now making use of the custom querier callbacks
//	customQuery := MaskQueryMsg{
//		ReflectCustom: &Text{
//			Text: "all Caps noW",
//		},
//	}
//	customQueryBz, err := json.Marshal(customQuery)
//	require.NoError(t, err)
//	custom, err := keeper.QuerySmart(ctx, contractAddr, customQueryBz)
//	require.NoError(t, err)
//	var resp customQueryResponse
//	err = json.Unmarshal(custom, &resp)
//	require.NoError(t, err)
//	assert.Equal(t, resp.Msg, "ALL CAPS NOW")
//}

//func checkAccount(t *testing.T, ctx sdk.Context, accKeeper auth.AccountKeeper, addr sdk.AccAddress, expected sdk.Coins) {
//	acct := accKeeper.GetAccount(ctx, addr)
//	if expected == nil {
//		assert.Nil(t, acct)
//	} else {
//		assert.NotNil(t, acct)
//		if expected.Empty() {
//			// there is confusion between nil and empty slice... let's just treat them the same
//			assert.True(t, acct.GetCoins().Empty())
//		} else {
//			assert.Equal(t, acct.GetCoins(), expected)
//		}
//	}
//}
