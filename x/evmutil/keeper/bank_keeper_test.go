package keeper_test

import (
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	sdkmath "cosmossdk.io/math"
	"github.com/CosmWasm/wasmd/x/evmutil/keeper"
	"github.com/CosmWasm/wasmd/x/evmutil/testutil"
	"github.com/CosmWasm/wasmd/x/evmutil/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	vesting "github.com/cosmos/cosmos-sdk/x/auth/vesting/types"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
)

type evmBankKeeperTestSuite struct {
	testutil.Suite
}

func (suite *evmBankKeeperTestSuite) SetupTest() {
	suite.Suite.SetupTest()
}

func (suite *evmBankKeeperTestSuite) TestGetBalance_ReturnsSpendable() {
	startingCoins := sdk.NewCoins(sdk.NewInt64Coin("orai", 10))
	startingAkava := sdkmath.NewInt(100)

	now := time.Now()
	endTime := now.Add(24 * time.Hour)
	bacc := authtypes.NewBaseAccountWithAddress(suite.Addrs[0])
	vacc, err := vesting.NewContinuousVestingAccount(bacc, startingCoins, now.Unix(), endTime.Unix())
	suite.Require().NoError(err)
	suite.AccountKeeper.SetAccount(suite.Ctx, vacc)

	err = suite.App.FundAccount(suite.Ctx, suite.Addrs[0], startingCoins)
	suite.Require().NoError(err)
	err = suite.Keeper.SetBalance(suite.Ctx, suite.Addrs[0], startingAkava)
	suite.Require().NoError(err)

	coin := suite.EvmBankKeeper.GetBalance(suite.Ctx, suite.Addrs[0], "aorai")
	suite.Require().Equal(startingAkava, coin.Amount)

	ctx := suite.Ctx.WithBlockTime(now.Add(12 * time.Hour))
	coin = suite.EvmBankKeeper.GetBalance(ctx, suite.Addrs[0], "aorai")
	suite.Require().Equal(sdkmath.NewIntFromUint64(5_000_000_000_100), coin.Amount)
}

func (suite *evmBankKeeperTestSuite) TestGetBalance_NotEvmDenom() {
	suite.Require().Panics(func() {
		suite.EvmBankKeeper.GetBalance(suite.Ctx, suite.Addrs[0], "orai")
	})
	suite.Require().Panics(func() {
		suite.EvmBankKeeper.GetBalance(suite.Ctx, suite.Addrs[0], "busd")
	})
}

func (suite *evmBankKeeperTestSuite) TestGetBalance() {
	tests := []struct {
		name           string
		startingAmount sdk.Coins
		expAmount      sdkmath.Int
	}{
		{
			"ukava with akava",
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 100),
				sdk.NewInt64Coin("orai", 10),
			),
			sdkmath.NewInt(10_000_000_000_100),
		},
		{
			"just akava",
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 100),
				sdk.NewInt64Coin("busd", 100),
			),
			sdkmath.NewInt(100),
		},
		{
			"just ukava",
			sdk.NewCoins(
				sdk.NewInt64Coin("orai", 10),
				sdk.NewInt64Coin("busd", 100),
			),
			sdkmath.NewInt(10_000_000_000_000),
		},
		{
			"no ukava or akava",
			sdk.NewCoins(),
			sdkmath.ZeroInt(),
		},
		{
			"with avaka that is more than 1 ukava",
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 20_000_000_000_220),
				sdk.NewInt64Coin("orai", 11),
			),
			sdkmath.NewInt(31_000_000_000_220),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			suite.FundAccountWithKava(suite.Addrs[0], tt.startingAmount)
			coin := suite.EvmBankKeeper.GetBalance(suite.Ctx, suite.Addrs[0], "aorai")
			suite.Require().Equal(tt.expAmount, coin.Amount)
		})
	}
}

func (suite *evmBankKeeperTestSuite) TestSendCoinsFromModuleToAccount() {
	startingModuleCoins := sdk.NewCoins(
		sdk.NewInt64Coin("aorai", 200),
		sdk.NewInt64Coin("orai", 100),
	)
	tests := []struct {
		name           string
		sendCoins      sdk.Coins
		startingAccBal sdk.Coins
		expAccBal      sdk.Coins
		hasErr         bool
	}{
		{
			"send more than 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 12_000_000_000_010)),
			sdk.Coins{},
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 10),
				sdk.NewInt64Coin("orai", 12),
			),
			false,
		},
		{
			"send less than 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 122)),
			sdk.Coins{},
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 122),
				sdk.NewInt64Coin("orai", 0),
			),
			false,
		},
		{
			"send an exact amount of ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 98_000_000_000_000)),
			sdk.Coins{},
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 0o0),
				sdk.NewInt64Coin("orai", 98),
			),
			false,
		},
		{
			"send no akava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 0)),
			sdk.Coins{},
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 0),
				sdk.NewInt64Coin("orai", 0),
			),
			false,
		},
		{
			"errors if sending other coins",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 500), sdk.NewInt64Coin("busd", 1000)),
			sdk.Coins{},
			sdk.Coins{},
			true,
		},
		{
			"errors if not enough total akava to cover",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100_000_000_001_000)),
			sdk.Coins{},
			sdk.Coins{},
			true,
		},
		{
			"errors if not enough ukava to cover",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 200_000_000_000_000)),
			sdk.Coins{},
			sdk.Coins{},
			true,
		},
		{
			"converts receiver's akava to ukava if there's enough akava after the transfer",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 99_000_000_000_200)),
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 999_999_999_900),
				sdk.NewInt64Coin("orai", 1),
			),
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 100),
				sdk.NewInt64Coin("orai", 101),
			),
			false,
		},
		{
			"converts all of receiver's akava to ukava even if somehow receiver has more than 1ukava of akava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 12_000_000_000_100)),
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 5_999_999_999_990),
				sdk.NewInt64Coin("orai", 1),
			),
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 90),
				sdk.NewInt64Coin("orai", 19),
			),
			false,
		},
		{
			"swap 1 ukava for akava if module account doesn't have enough akava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 99_000_000_001_000)),
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 200),
				sdk.NewInt64Coin("orai", 1),
			),
			sdk.NewCoins(
				sdk.NewInt64Coin("aorai", 1200),
				sdk.NewInt64Coin("orai", 100),
			),
			false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			suite.FundAccountWithKava(suite.Addrs[0], tt.startingAccBal)
			suite.FundModuleAccountWithKava(evmtypes.ModuleName, startingModuleCoins)

			// fund our module with some ukava to account for converting extra akava back to ukava
			suite.FundModuleAccountWithKava(types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("orai", 10)))

			err := suite.EvmBankKeeper.SendCoinsFromModuleToAccount(suite.Ctx, evmtypes.ModuleName, suite.Addrs[0], tt.sendCoins)
			if tt.hasErr {
				suite.Require().Error(err)
				return
			} else {
				suite.Require().NoError(err)
			}

			// check ukava
			ukavaSender := suite.BankKeeper.GetBalance(suite.Ctx, suite.Addrs[0], "orai")
			suite.Require().Equal(tt.expAccBal.AmountOf("orai").Int64(), ukavaSender.Amount.Int64())

			// check akava
			actualAkava := suite.Keeper.GetBalance(suite.Ctx, suite.Addrs[0])
			suite.Require().Equal(tt.expAccBal.AmountOf("aorai").Int64(), actualAkava.Int64())
		})
	}
}

func (suite *evmBankKeeperTestSuite) TestSendCoinsFromAccountToModule() {
	startingAccCoins := sdk.NewCoins(
		sdk.NewInt64Coin("aorai", 200),
		sdk.NewInt64Coin("orai", 100),
	)
	startingModuleCoins := sdk.NewCoins(
		sdk.NewInt64Coin("aorai", 100_000_000_000),
	)
	tests := []struct {
		name           string
		sendCoins      sdk.Coins
		expSenderCoins sdk.Coins
		expModuleCoins sdk.Coins
		hasErr         bool
	}{
		{
			"send more than 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 12_000_000_000_010)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 190), sdk.NewInt64Coin("orai", 88)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100_000_000_010), sdk.NewInt64Coin("orai", 12)),
			false,
		},
		{
			"send less than 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 122)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 78), sdk.NewInt64Coin("orai", 100)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100_000_000_122), sdk.NewInt64Coin("orai", 0)),
			false,
		},
		{
			"send an exact amount of ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 98_000_000_000_000)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 200), sdk.NewInt64Coin("orai", 2)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100_000_000_000), sdk.NewInt64Coin("orai", 98)),
			false,
		},
		{
			"send no akava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 0)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 200), sdk.NewInt64Coin("orai", 100)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100_000_000_000), sdk.NewInt64Coin("orai", 0)),
			false,
		},
		{
			"errors if sending other coins",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 500), sdk.NewInt64Coin("busd", 1000)),
			sdk.Coins{},
			sdk.Coins{},
			true,
		},
		{
			"errors if have dup coins",
			sdk.Coins{
				sdk.NewInt64Coin("aorai", 12_000_000_000_000),
				sdk.NewInt64Coin("aorai", 2_000_000_000_000),
			},
			sdk.Coins{},
			sdk.Coins{},
			true,
		},
		{
			"errors if not enough total akava to cover",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100_000_000_001_000)),
			sdk.Coins{},
			sdk.Coins{},
			true,
		},
		{
			"errors if not enough ukava to cover",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 200_000_000_000_000)),
			sdk.Coins{},
			sdk.Coins{},
			true,
		},
		{
			"converts 1 ukava to akava if not enough akava to cover",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 99_001_000_000_000)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 999_000_000_200), sdk.NewInt64Coin("orai", 0)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 101_000_000_000), sdk.NewInt64Coin("orai", 99)),
			false,
		},
		{
			"converts receiver's akava to ukava if there's enough akava after the transfer",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 5_900_000_000_200)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100_000_000_000), sdk.NewInt64Coin("orai", 94)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 200), sdk.NewInt64Coin("orai", 6)),
			false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()
			suite.FundAccountWithKava(suite.Addrs[0], startingAccCoins)
			suite.FundModuleAccountWithKava(evmtypes.ModuleName, startingModuleCoins)

			err := suite.EvmBankKeeper.SendCoinsFromAccountToModule(suite.Ctx, suite.Addrs[0], evmtypes.ModuleName, tt.sendCoins)
			if tt.hasErr {
				suite.Require().Error(err)
				return
			} else {
				suite.Require().NoError(err)
			}

			// check sender balance
			ukavaSender := suite.BankKeeper.GetBalance(suite.Ctx, suite.Addrs[0], "orai")
			suite.Require().Equal(tt.expSenderCoins.AmountOf("orai").Int64(), ukavaSender.Amount.Int64())
			actualAkava := suite.Keeper.GetBalance(suite.Ctx, suite.Addrs[0])
			suite.Require().Equal(tt.expSenderCoins.AmountOf("aorai").Int64(), actualAkava.Int64())

			// check module balance
			moduleAddr := suite.AccountKeeper.GetModuleAddress(evmtypes.ModuleName)
			ukavaSender = suite.BankKeeper.GetBalance(suite.Ctx, moduleAddr, "orai")
			suite.Require().Equal(tt.expModuleCoins.AmountOf("orai").Int64(), ukavaSender.Amount.Int64())
			actualAkava = suite.Keeper.GetBalance(suite.Ctx, moduleAddr)
			suite.Require().Equal(tt.expModuleCoins.AmountOf("aorai").Int64(), actualAkava.Int64())
		})
	}
}

func (suite *evmBankKeeperTestSuite) TestBurnCoins() {
	startingUkava := sdkmath.NewInt(100)
	tests := []struct {
		name       string
		burnCoins  sdk.Coins
		expUkava   sdkmath.Int
		expAkava   sdkmath.Int
		hasErr     bool
		akavaStart sdkmath.Int
	}{
		{
			"burn more than 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 12_021_000_000_002)),
			sdkmath.NewInt(88),
			sdkmath.NewInt(100_000_000_000),
			false,
			sdkmath.NewInt(121_000_000_002),
		},
		{
			"burn less than 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 122)),
			sdkmath.NewInt(100),
			sdkmath.NewInt(878),
			false,
			sdkmath.NewInt(1000),
		},
		{
			"burn an exact amount of ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 98_000_000_000_000)),
			sdkmath.NewInt(2),
			sdkmath.NewInt(10),
			false,
			sdkmath.NewInt(10),
		},
		{
			"burn no akava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 0)),
			startingUkava,
			sdkmath.ZeroInt(),
			false,
			sdkmath.ZeroInt(),
		},
		{
			"errors if burning other coins",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 500), sdk.NewInt64Coin("busd", 1000)),
			startingUkava,
			sdkmath.NewInt(100),
			true,
			sdkmath.NewInt(100),
		},
		{
			"errors if have dup coins",
			sdk.Coins{
				sdk.NewInt64Coin("aorai", 12_000_000_000_000),
				sdk.NewInt64Coin("aorai", 2_000_000_000_000),
			},
			startingUkava,
			sdkmath.ZeroInt(),
			true,
			sdkmath.ZeroInt(),
		},
		{
			"errors if burn amount is negative",
			sdk.Coins{sdk.Coin{Denom: "aorai", Amount: sdkmath.NewInt(-100)}},
			startingUkava,
			sdkmath.NewInt(50),
			true,
			sdkmath.NewInt(50),
		},
		{
			"errors if not enough akava to cover burn",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100_999_000_000_000)),
			sdkmath.NewInt(0),
			sdkmath.NewInt(99_000_000_000),
			true,
			sdkmath.NewInt(99_000_000_000),
		},
		{
			"errors if not enough ukava to cover burn",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 200_000_000_000_000)),
			sdkmath.NewInt(100),
			sdkmath.ZeroInt(),
			true,
			sdkmath.ZeroInt(),
		},
		{
			"converts 1 ukava to akava if not enough akava to cover",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 12_021_000_000_002)),
			sdkmath.NewInt(87),
			sdkmath.NewInt(980_000_000_000),
			false,
			sdkmath.NewInt(1_000_000_002),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()
			startingCoins := sdk.NewCoins(
				sdk.NewCoin("orai", startingUkava),
				sdk.NewCoin("aorai", tt.akavaStart),
			)
			suite.FundModuleAccountWithKava(evmtypes.ModuleName, startingCoins)

			err := suite.EvmBankKeeper.BurnCoins(suite.Ctx, evmtypes.ModuleName, tt.burnCoins)
			if tt.hasErr {
				suite.Require().Error(err)
				return
			} else {
				suite.Require().NoError(err)
			}

			// check ukava
			ukavaActual := suite.BankKeeper.GetBalance(suite.Ctx, suite.EvmModuleAddr, "orai")
			suite.Require().Equal(tt.expUkava, ukavaActual.Amount)

			// check akava
			akavaActual := suite.Keeper.GetBalance(suite.Ctx, suite.EvmModuleAddr)
			suite.Require().Equal(tt.expAkava, akavaActual)
		})
	}
}

func (suite *evmBankKeeperTestSuite) TestMintCoins() {
	tests := []struct {
		name       string
		mintCoins  sdk.Coins
		ukava      sdkmath.Int
		akava      sdkmath.Int
		hasErr     bool
		akavaStart sdkmath.Int
	}{
		{
			"mint more than 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 12_021_000_000_002)),
			sdkmath.NewInt(12),
			sdkmath.NewInt(21_000_000_002),
			false,
			sdkmath.ZeroInt(),
		},
		{
			"mint less than 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 901_000_000_001)),
			sdkmath.ZeroInt(),
			sdkmath.NewInt(901_000_000_001),
			false,
			sdkmath.ZeroInt(),
		},
		{
			"mint an exact amount of ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 123_000_000_000_000_000)),
			sdkmath.NewInt(123_000),
			sdkmath.ZeroInt(),
			false,
			sdkmath.ZeroInt(),
		},
		{
			"mint no akava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 0)),
			sdkmath.ZeroInt(),
			sdkmath.ZeroInt(),
			false,
			sdkmath.ZeroInt(),
		},
		{
			"errors if minting other coins",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 500), sdk.NewInt64Coin("busd", 1000)),
			sdkmath.ZeroInt(),
			sdkmath.NewInt(100),
			true,
			sdkmath.NewInt(100),
		},
		{
			"errors if have dup coins",
			sdk.Coins{
				sdk.NewInt64Coin("aorai", 12_000_000_000_000),
				sdk.NewInt64Coin("aorai", 2_000_000_000_000),
			},
			sdkmath.ZeroInt(),
			sdkmath.ZeroInt(),
			true,
			sdkmath.ZeroInt(),
		},
		{
			"errors if mint amount is negative",
			sdk.Coins{sdk.Coin{Denom: "aorai", Amount: sdkmath.NewInt(-100)}},
			sdkmath.ZeroInt(),
			sdkmath.NewInt(50),
			true,
			sdkmath.NewInt(50),
		},
		{
			"adds to existing akava balance",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 12_021_000_000_002)),
			sdkmath.NewInt(12),
			sdkmath.NewInt(21_000_000_102),
			false,
			sdkmath.NewInt(100),
		},
		{
			"convert akava balance to ukava if it exceeds 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 10_999_000_000_000)),
			sdkmath.NewInt(12),
			sdkmath.NewInt(1_200_000_001),
			false,
			sdkmath.NewInt(1_002_200_000_001),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()
			suite.FundModuleAccountWithKava(types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("orai", 10)))
			suite.FundModuleAccountWithKava(evmtypes.ModuleName, sdk.NewCoins(sdk.NewCoin("aorai", tt.akavaStart)))

			err := suite.EvmBankKeeper.MintCoins(suite.Ctx, evmtypes.ModuleName, tt.mintCoins)
			if tt.hasErr {
				suite.Require().Error(err)
				return
			} else {
				suite.Require().NoError(err)
			}

			// check ukava
			ukavaActual := suite.BankKeeper.GetBalance(suite.Ctx, suite.EvmModuleAddr, "orai")
			suite.Require().Equal(tt.ukava, ukavaActual.Amount)

			// check akava
			akavaActual := suite.Keeper.GetBalance(suite.Ctx, suite.EvmModuleAddr)
			suite.Require().Equal(tt.akava, akavaActual)
		})
	}
}

func (suite *evmBankKeeperTestSuite) TestValidateEvmCoins() {
	tests := []struct {
		name      string
		coins     sdk.Coins
		shouldErr bool
	}{
		{
			"valid coins",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 500)),
			false,
		},
		{
			"dup coins",
			sdk.Coins{sdk.NewInt64Coin("aorai", 500), sdk.NewInt64Coin("aorai", 500)},
			true,
		},
		{
			"not evm coins",
			sdk.NewCoins(sdk.NewInt64Coin("orai", 500)),
			true,
		},
		{
			"negative coins",
			sdk.Coins{sdk.Coin{Denom: "aorai", Amount: sdkmath.NewInt(-500)}},
			true,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			err := keeper.ValidateEvmCoins(tt.coins, keeper.DefaultEvmDenom)
			if tt.shouldErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
			}
		})
	}
}

func (suite *evmBankKeeperTestSuite) TestConvertOneUkavaToAkavaIfNeeded() {
	akavaNeeded := sdkmath.NewInt(200)
	tests := []struct {
		name          string
		startingCoins sdk.Coins
		expectedCoins sdk.Coins
		success       bool
	}{
		{
			"not enough ukava for conversion",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100)),
			false,
		},
		{
			"converts 1 ukava to akava",
			sdk.NewCoins(sdk.NewInt64Coin("orai", 10), sdk.NewInt64Coin("aorai", 100)),
			sdk.NewCoins(sdk.NewInt64Coin("orai", 9), sdk.NewInt64Coin("aorai", 1_000_000_000_100)),
			true,
		},
		{
			"conversion not needed",
			sdk.NewCoins(sdk.NewInt64Coin("orai", 10), sdk.NewInt64Coin("aorai", 200)),
			sdk.NewCoins(sdk.NewInt64Coin("orai", 10), sdk.NewInt64Coin("aorai", 200)),
			true,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			suite.FundAccountWithKava(suite.Addrs[0], tt.startingCoins)
			err := suite.EvmBankKeeper.ConvertOneUkavaToAkavaIfNeeded(suite.Ctx, suite.Addrs[0], akavaNeeded)
			moduleKava := suite.BankKeeper.GetBalance(suite.Ctx, suite.AccountKeeper.GetModuleAddress(types.ModuleName), "orai")
			if tt.success {
				suite.Require().NoError(err)
				if tt.startingCoins.AmountOf("aorai").LT(akavaNeeded) {
					suite.Require().Equal(sdkmath.OneInt(), moduleKava.Amount)
				}
			} else {
				suite.Require().Error(err)
				suite.Require().Equal(sdkmath.ZeroInt(), moduleKava.Amount)
			}

			akava := suite.Keeper.GetBalance(suite.Ctx, suite.Addrs[0])
			suite.Require().Equal(tt.expectedCoins.AmountOf("aorai"), akava)
			ukava := suite.BankKeeper.GetBalance(suite.Ctx, suite.Addrs[0], "orai")
			suite.Require().Equal(tt.expectedCoins.AmountOf("orai"), ukava.Amount)
		})
	}
}

func (suite *evmBankKeeperTestSuite) TestConvertAkavaToUkava() {
	baseFee := int64(suite.App.FeeMarketKeeper.GetBaseFee(suite.Ctx).Uint64())
	tests := []struct {
		name          string
		startingCoins sdk.Coins
		expectedCoins sdk.Coins
	}{
		{
			"not enough ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 100+baseFee), sdk.NewInt64Coin("orai", 0+baseFee)),
		},
		{
			"converts akava for 1 ukava",
			sdk.NewCoins(sdk.NewInt64Coin("orai", 10), sdk.NewInt64Coin("aorai", 1_000_000_000_003)),
			sdk.NewCoins(sdk.NewInt64Coin("orai", 11+baseFee), sdk.NewInt64Coin("aorai", 3)),
		},
		{
			"converts more than 1 ukava of akava",
			sdk.NewCoins(sdk.NewInt64Coin("orai", 10), sdk.NewInt64Coin("aorai", 8_000_000_000_123)),
			sdk.NewCoins(sdk.NewInt64Coin("orai", 18+baseFee), sdk.NewInt64Coin("aorai", 123)),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			err := suite.App.FundModuleAccount(suite.Ctx, types.ModuleName, sdk.NewCoins(sdk.NewInt64Coin("orai", 10)))
			suite.Require().NoError(err)
			suite.FundAccountWithKava(suite.Addrs[0], tt.startingCoins)
			err = suite.EvmBankKeeper.ConvertAkavaToUkava(suite.Ctx, suite.Addrs[0])
			suite.Require().NoError(err)
			akava := suite.Keeper.GetBalance(suite.Ctx, suite.Addrs[0])
			suite.Require().Equal(tt.expectedCoins.AmountOf("aorai"), akava)
			ukava := suite.BankKeeper.GetBalance(suite.Ctx, suite.Addrs[0], "orai")
			suite.Require().Equal(tt.expectedCoins.AmountOf("orai"), ukava.Amount)

		})
	}
}

func (suite *evmBankKeeperTestSuite) TestSplitAkavaCoins() {
	tests := []struct {
		name          string
		coins         sdk.Coins
		expectedCoins sdk.Coins
		shouldErr     bool
	}{
		{
			"invalid coins",
			sdk.NewCoins(sdk.NewInt64Coin("orai", 500)),
			nil,
			true,
		},
		{
			"empty coins",
			sdk.NewCoins(),
			sdk.NewCoins(),
			false,
		},
		{
			"ukava & akava coins",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 8_000_000_000_123)),
			sdk.NewCoins(sdk.NewInt64Coin("orai", 8), sdk.NewInt64Coin("aorai", 123)),
			false,
		},
		{
			"only akava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 10_123)),
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 10_123)),
			false,
		},
		{
			"only ukava",
			sdk.NewCoins(sdk.NewInt64Coin("aorai", 5_000_000_000_000)),
			sdk.NewCoins(sdk.NewInt64Coin("orai", 5)),
			false,
		},
	}
	for _, tt := range tests {
		suite.Run(tt.name, func() {
			ukava, akava, err := keeper.SplitAkavaCoins(tt.coins, keeper.DefaultEvmDenom, keeper.DefaultCosmosDenom)
			if tt.shouldErr {
				suite.Require().Error(err)
			} else {
				suite.Require().NoError(err)
				suite.Require().Equal(tt.expectedCoins.AmountOf("orai"), ukava.Amount)
				suite.Require().Equal(tt.expectedCoins.AmountOf("aorai"), akava)
			}
		})
	}
}

func TestEvmBankKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(evmBankKeeperTestSuite))
}
