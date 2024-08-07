package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/suite"

	"github.com/CosmWasm/wasmd/x/evmutil/testutil"
	"github.com/CosmWasm/wasmd/x/evmutil/types"
)

type keeperTestSuite struct {
	testutil.Suite
}

func (suite *keeperTestSuite) SetupTest() {
	suite.Suite.SetupTest()
}

func (suite *keeperTestSuite) TestGetAllAccounts() {
	tests := []struct {
		name        string
		expAccounts []types.Account
	}{
		{
			"no accounts",
			[]types.Account{},
		},
		{
			"with accounts",
			[]types.Account{
				{Address: suite.Addrs[0], Balance: sdkmath.NewInt(100)},
				{Address: suite.Addrs[1], Balance: sdkmath.NewInt(200)},
			},
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			for _, account := range tt.expAccounts {
				suite.Keeper.SetBalance(suite.Ctx, account.Address, account.Balance)
			}

			accounts := suite.Suite.Keeper.GetAllAccounts(suite.Ctx)
			if len(tt.expAccounts) == 0 {
				suite.Require().Len(tt.expAccounts, 0)
			} else {
				suite.Require().Equal(tt.expAccounts, accounts)
			}
		})
	}
}

func (suite *keeperTestSuite) TestSetAccount_ZeroBalance() {
	existingAccount := types.Account{
		Address: suite.Addrs[0],
		Balance: sdkmath.NewInt(100),
	}
	err := suite.Keeper.SetAccount(suite.Ctx, existingAccount)
	suite.Require().NoError(err)
	err = suite.Keeper.SetAccount(suite.Ctx, types.Account{
		Address: suite.Addrs[0],
		Balance: sdkmath.ZeroInt(),
	})
	suite.Require().NoError(err)
	bal := suite.Keeper.GetBalance(suite.Ctx, suite.Addrs[0])
	suite.Require().Equal(sdkmath.ZeroInt(), bal)
	expAcct := suite.Keeper.GetAccount(suite.Ctx, suite.Addrs[0])
	suite.Require().Nil(expAcct)
}

func (suite *keeperTestSuite) TestSetAccount() {
	existingAccount := types.Account{
		Address: suite.Addrs[0],
		Balance: sdkmath.NewInt(100),
	}
	tests := []struct {
		name    string
		account types.Account
		success bool
	}{
		{
			"invalid address",
			types.Account{Address: nil, Balance: sdkmath.NewInt(100)},
			false,
		},
		{
			"invalid balance",
			types.Account{Address: suite.Addrs[0], Balance: sdkmath.NewInt(-100)},
			false,
		},
		{
			"empty account",
			types.Account{},
			false,
		},
		{
			"valid account",
			types.Account{Address: suite.Addrs[1], Balance: sdkmath.NewInt(100)},
			true,
		},
		{
			"replaces account",
			types.Account{Address: suite.Addrs[0], Balance: sdkmath.NewInt(50)},
			true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			err := suite.Keeper.SetAccount(suite.Ctx, existingAccount)
			suite.Require().NoError(err)
			err = suite.Keeper.SetAccount(suite.Ctx, tt.account)
			if tt.success {
				suite.Require().NoError(err)
				expAcct := suite.Keeper.GetAccount(suite.Ctx, tt.account.Address)
				suite.Require().Equal(tt.account, *expAcct)
			} else {
				suite.Require().Error(err)
				suite.Require().Nil(suite.Keeper.GetAccount(suite.Ctx, suite.Addrs[1]))
			}
		})
	}
}

func (suite *keeperTestSuite) TestSendBalance() {
	startingSenderBal := sdkmath.NewInt(100)
	startingRecipientBal := sdkmath.NewInt(50)
	tests := []struct {
		name            string
		amt             sdkmath.Int
		expSenderBal    sdkmath.Int
		expRecipientBal sdkmath.Int
		success         bool
	}{
		{
			"fails when sending negative amount",
			sdkmath.NewInt(-5),
			sdkmath.ZeroInt(),
			sdkmath.ZeroInt(),
			false,
		},
		{
			"send zero amount",
			sdkmath.ZeroInt(),
			startingSenderBal,
			startingRecipientBal,
			true,
		},
		{
			"fails when sender does not have enough balance",
			sdkmath.NewInt(101),
			sdkmath.ZeroInt(),
			sdkmath.ZeroInt(),
			false,
		},
		{
			"send valid amount",
			sdkmath.NewInt(80),
			sdkmath.NewInt(20),
			sdkmath.NewInt(130),
			true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			err := suite.Keeper.SetBalance(suite.Ctx, suite.Addrs[0], startingSenderBal)
			suite.Require().NoError(err)
			err = suite.Keeper.SetBalance(suite.Ctx, suite.Addrs[1], startingRecipientBal)
			suite.Require().NoError(err)

			err = suite.Keeper.SendBalance(suite.Ctx, suite.Addrs[0], suite.Addrs[1], tt.amt)
			if tt.success {
				suite.Require().NoError(err)
				suite.Require().Equal(tt.expSenderBal, suite.Keeper.GetBalance(suite.Ctx, suite.Addrs[0]))
				suite.Require().Equal(tt.expRecipientBal, suite.Keeper.GetBalance(suite.Ctx, suite.Addrs[1]))
			} else {
				suite.Require().Error(err)
			}
		})
	}
}

func (suite *keeperTestSuite) TestSetBalance() {
	existingAccount := types.Account{
		Address: suite.Addrs[0],
		Balance: sdkmath.NewInt(100),
	}
	tests := []struct {
		name    string
		address sdk.AccAddress
		balance sdkmath.Int
		success bool
	}{
		{
			"invalid balance",
			suite.Addrs[0],
			sdkmath.NewInt(-100),
			false,
		},
		{
			"set new account balance",
			suite.Addrs[1],
			sdkmath.NewInt(100),
			true,
		},
		{
			"replace account balance",
			suite.Addrs[0],
			sdkmath.NewInt(50),
			true,
		},
		{
			"invalid address",
			nil,
			sdkmath.NewInt(100),
			false,
		},
		{
			"zero balance",
			suite.Addrs[0],
			sdkmath.ZeroInt(),
			true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			err := suite.Keeper.SetAccount(suite.Ctx, existingAccount)
			suite.Require().NoError(err)
			err = suite.Keeper.SetBalance(suite.Ctx, tt.address, tt.balance)
			if tt.success {
				suite.Require().NoError(err)
				expBal := suite.Keeper.GetBalance(suite.Ctx, tt.address)
				suite.Require().Equal(expBal, tt.balance)

				if tt.balance.IsZero() {
					account := suite.Keeper.GetAccount(suite.Ctx, tt.address)
					suite.Require().Nil(account)
				}
			} else {
				suite.Require().Error(err)
				expBal := suite.Keeper.GetBalance(suite.Ctx, existingAccount.Address)
				suite.Require().Equal(expBal, existingAccount.Balance)
			}
		})
	}
}

func (suite *keeperTestSuite) TestRemoveBalance() {
	existingAccount := types.Account{
		Address: suite.Addrs[0],
		Balance: sdkmath.NewInt(100),
	}
	tests := []struct {
		name    string
		amt     sdkmath.Int
		expBal  sdkmath.Int
		success bool
	}{
		{
			"fails if amount is negative",
			sdkmath.NewInt(-10),
			sdkmath.ZeroInt(),
			false,
		},
		{
			"remove zero amount",
			sdkmath.ZeroInt(),
			existingAccount.Balance,
			true,
		},
		{
			"not enough balance",
			sdkmath.NewInt(101),
			sdkmath.ZeroInt(),
			false,
		},
		{
			"remove full balance",
			sdkmath.NewInt(100),
			sdkmath.ZeroInt(),
			true,
		},
		{
			"remove some balance",
			sdkmath.NewInt(10),
			sdkmath.NewInt(90),
			true,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			err := suite.Keeper.SetAccount(suite.Ctx, existingAccount)
			suite.Require().NoError(err)
			err = suite.Keeper.RemoveBalance(suite.Ctx, existingAccount.Address, tt.amt)
			if tt.success {
				suite.Require().NoError(err)
				expBal := suite.Keeper.GetBalance(suite.Ctx, existingAccount.Address)
				suite.Require().Equal(expBal, tt.expBal)
			} else {
				suite.Require().Error(err)
				expBal := suite.Keeper.GetBalance(suite.Ctx, existingAccount.Address)
				suite.Require().Equal(expBal, existingAccount.Balance)
			}
		})
	}
}

func (suite *keeperTestSuite) TestGetBalance() {
	existingAccount := types.Account{
		Address: suite.Addrs[0],
		Balance: sdkmath.NewInt(100),
	}
	tests := []struct {
		name   string
		addr   sdk.AccAddress
		expBal sdkmath.Int
	}{
		{
			"returns 0 balance if account does not exist",
			suite.Addrs[1],
			sdkmath.ZeroInt(),
		},
		{
			"returns account balance",
			suite.Addrs[0],
			sdkmath.NewInt(100),
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			suite.SetupTest()

			err := suite.Keeper.SetAccount(suite.Ctx, existingAccount)
			suite.Require().NoError(err)
			balance := suite.Keeper.GetBalance(suite.Ctx, tt.addr)
			suite.Require().Equal(tt.expBal, balance)
		})
	}
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(keeperTestSuite))
}
