package common

import (
	"context"

	sdk "github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	"github.com/ethereum/go-ethereum/common"
)

type EVMKeeper interface {
	GetCosmosAddressMapping(ctx sdk.Context, evmAddress common.Address) sdk.AccAddress
	GetEvmAddressMapping(ctx sdk.Context, addr sdk.AccAddress) (*common.Address, error)
	SetMappingEvmAddressInner(
		ctx sdk.Context,
		msgSigner string,
		msgPubKey string,
	) error
}

type WasmdKeeper interface {
	Instantiate(ctx sdk.Context, codeID uint64, creator, admin sdk.AccAddress, initMsg []byte, label string, deposit sdk.Coins) (sdk.AccAddress, []byte, error)
	Execute(ctx sdk.Context, contractAddress sdk.AccAddress, caller sdk.AccAddress, msg []byte, coins sdk.Coins) ([]byte, error)
}

type WasmdViewKeeper interface {
	QuerySmart(ctx context.Context, contractAddr sdk.AccAddress, req []byte) ([]byte, error)
}

type BankKeeper interface {
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	GetAllBalances(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	GetDenomMetaData(ctx context.Context, denom string) (banktypes.Metadata, bool)
	GetSupply(ctx context.Context, denom string) sdk.Coin
	LockedCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
}

type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
	HasAccount(ctx context.Context, addr sdk.AccAddress) bool
	SetAccount(ctx context.Context, acc sdk.AccountI)
	RemoveAccount(ctx context.Context, acc sdk.AccountI)
	NewAccountWithAddress(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}
