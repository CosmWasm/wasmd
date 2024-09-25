package app

import (
	"errors"
	"fmt"
	"runtime/debug"

	corestoretypes "cosmossdk.io/core/store"
	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/log"
	circuitante "cosmossdk.io/x/circuit/ante"
	circuitkeeper "cosmossdk.io/x/circuit/keeper"
	globalfeekeeper "github.com/CosmosContracts/juno/v18/x/globalfee/keeper"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
	ibcante "github.com/cosmos/ibc-go/v8/modules/core/ante"
	"github.com/cosmos/ibc-go/v8/modules/core/keeper"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/cosmos/cosmos-sdk/x/auth/ante"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"

	wasmkeeper "github.com/CosmWasm/wasmd/x/wasm/keeper"
	wasmTypes "github.com/CosmWasm/wasmd/x/wasm/types"

	"github.com/cosmos/cosmos-sdk/types/tx/signing"

	storetypes "cosmossdk.io/store/types"
	globalfeeante "github.com/CosmosContracts/juno/v18/x/globalfee/ante"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	evmante "github.com/evmos/ethermint/app/ante"
	"github.com/evmos/ethermint/crypto/ethsecp256k1"
	evmkeeper "github.com/evmos/ethermint/x/evm/keeper"
	evmtypes "github.com/evmos/ethermint/x/evm/types"
	feemarketkeeper "github.com/evmos/ethermint/x/feemarket/keeper"
)

const maxBypassMinFeeMsgGasUsage = 1_000_000

// HandlerOptions extend the SDK's AnteHandler options by requiring the IBC
// channel keeper.
type HandlerOptions struct {
	ante.HandlerOptions
	AccountKeeper         evmtypes.AccountKeeper
	IBCKeeper             *keeper.Keeper
	EvmKeeper             *evmkeeper.Keeper
	GlobalFeeKeeper       globalfeekeeper.Keeper
	StakingKeeper         stakingkeeper.Keeper
	FeeMarketKeeper       feemarketkeeper.Keeper
	WasmConfig            *wasmTypes.WasmConfig
	WasmKeeper            *wasmkeeper.Keeper
	TXCounterStoreService corestoretypes.KVStoreService
	TxCounterStoreKey     storetypes.StoreKey
	MaxTxGasWanted        uint64
	CircuitKeeper         *circuitkeeper.Keeper
	BankKeeper            evmtypes.BankKeeper
	DisabledAuthzMsgs     []string
	BypassMinFeeMsgTypes  []string
}

func (options *HandlerOptions) Validate() error {
	if options.AccountKeeper == nil {
		return errors.New("account keeper is required for ante builder")
	}
	if options.BankKeeper == nil {
		return errors.New("bank keeper is required for ante builder")
	}
	if options.SignModeHandler == nil {
		return errors.New("sign mode handler is required for ante builder")
	}
	if options.WasmConfig == nil {
		return errors.New("wasm config is required for ante builder")
	}
	if options.TXCounterStoreService == nil {
		return errors.New("wasm store service is required for ante builder")
	}
	if options.CircuitKeeper == nil {
		return errors.New("circuit keeper is required for ante builder")
	}
	if options.EvmKeeper == nil {
		return errors.New("evm keeper is required for ante builder")
	}
	return nil
}

// NewAnteHandler constructor
func NewAnteHandler(options HandlerOptions) (sdk.AnteHandler, error) {

	if err := options.Validate(); err != nil {
		return nil, err
	}

	return func(
		ctx sdk.Context, tx sdk.Tx, sim bool,
	) (newCtx sdk.Context, err error) {
		var anteHandler sdk.AnteHandler

		defer Recover(ctx.Logger(), &err)

		txWithExtensions, ok := tx.(ante.HasExtensionOptionsTx)
		if ok {
			opts := txWithExtensions.GetExtensionOptions()
			if len(opts) > 1 {
				return ctx, errorsmod.Wrap(
					sdkerrors.ErrInvalidRequest,
					"rejecting tx with more than 1 extension option",
				)
			}

			if len(opts) == 1 {
				switch typeURL := opts[0].GetTypeUrl(); typeURL {
				case "/ethermint.evm.v1.ExtensionOptionsEthereumTx":
					// handle as *evmtypes.MsgEthereumTx
					anteHandler = newEthAnteHandler(options)
				case "/ethermint.types.v1.ExtensionOptionsWeb3Tx":
					// handle as normal Cosmos SDK tx, except signature is checked for EIP712 representation
					anteHandler = evmante.NewLegacyCosmosAnteHandlerEip712(evmante.HandlerOptions{
						AccountKeeper:          options.AccountKeeper,
						BankKeeper:             options.BankKeeper,
						SignModeHandler:        options.SignModeHandler,
						FeegrantKeeper:         options.FeegrantKeeper,
						SigGasConsumer:         options.SigGasConsumer,
						IBCKeeper:              options.IBCKeeper,
						EvmKeeper:              options.EvmKeeper,
						FeeMarketKeeper:        options.FeeMarketKeeper,
						MaxTxGasWanted:         options.MaxTxGasWanted,
						ExtensionOptionChecker: options.ExtensionOptionChecker,
						TxFeeChecker:           options.TxFeeChecker,
						DisabledAuthzMsgs:      options.DisabledAuthzMsgs,
					})
				default:
					return ctx, errorsmod.Wrapf(
						sdkerrors.ErrUnknownExtensionOptions,
						"rejecting tx with unsupported extension option: %s", typeURL,
					)
				}

				return anteHandler(ctx, tx, sim)
			}
		}

		// handle as totally normal Cosmos SDK tx
		switch tx.(type) {
		case sdk.Tx:
			anteHandler = newCosmosAnteHandler(options)
		default:
			return ctx, errorsmod.Wrapf(sdkerrors.ErrUnknownRequest, "invalid transaction type: %T", tx)
		}

		return anteHandler(ctx, tx, sim)
	}, nil
}

// NewAnteHandler returns an AnteHandler that checks and increments sequence
// numbers, checks signatures & account numbers, and deducts fees from the first
// signer.
func newCosmosAnteHandler(options HandlerOptions) sdk.AnteHandler {

	var sigGasConsumer = options.SigGasConsumer
	if sigGasConsumer == nil {
		sigGasConsumer = ante.DefaultSigVerificationGasConsumer
	}

	decorators := []sdk.AnteDecorator{
		evmante.RejectMessagesDecorator{}, // reject MsgEthereumTxs
		ante.NewSetUpContextDecorator(),   // outermost AnteDecorator. SetUpContext must be called first
		wasmkeeper.NewLimitSimulationGasDecorator(options.WasmConfig.SimulationGasLimit), // after setup context to enforce limits early
		wasmkeeper.NewCountTXDecorator(options.TXCounterStoreService),
		wasmkeeper.NewGasRegisterDecorator(options.WasmKeeper.GetGasRegister()),
		circuitante.NewCircuitBreakerDecorator(options.CircuitKeeper),
		ante.NewExtensionOptionsDecorator(options.ExtensionOptionChecker),
		ante.NewValidateBasicDecorator(),
		ante.NewTxTimeoutHeightDecorator(),
		ante.NewValidateMemoDecorator(options.AccountKeeper),
		ante.NewConsumeGasForTxSizeDecorator(options.AccountKeeper),
		// nil so that it only checks with the min gas price of the chain, not the custom fee checker. For cosmos messages, the default tx fee checker is enough
		globalfeeante.NewFeeDecorator(options.BypassMinFeeMsgTypes, options.GlobalFeeKeeper, options.StakingKeeper, maxBypassMinFeeMsgGasUsage),
		ante.NewDeductFeeDecorator(options.AccountKeeper, options.BankKeeper, options.FeegrantKeeper, nil),
		// we use evmante.NewSetPubKeyDecorator so that for eth_secp256k1 accs, we can validate the signer using the evm-cosmos mapping logic
		evmante.NewSetPubKeyDecorator(options.AccountKeeper, options.EvmKeeper), // SetPubKeyDecorator must be called before all signature verification decorators
		ante.NewValidateSigCountDecorator(options.AccountKeeper),
		ante.NewSigGasConsumeDecorator(options.AccountKeeper, options.SigGasConsumer),
		ante.NewSigVerificationDecorator(options.AccountKeeper, options.SignModeHandler),
		ante.NewIncrementSequenceDecorator(options.AccountKeeper),
		ibcante.NewRedundantRelayDecorator(options.IBCKeeper),
	}

	return sdk.ChainAnteDecorators(decorators...)
}

func newEthAnteHandler(options HandlerOptions) sdk.AnteHandler {
	return sdk.ChainAnteDecorators(
		evmante.NewEthSetUpContextDecorator(options.EvmKeeper), // outermost AnteDecorator. SetUpContext must be called first
		evmante.NewEthMempoolFeeDecorator(options.EvmKeeper),   // Check eth effective gas price against minimal-gas-prices
		evmante.NewEthValidateBasicDecorator(options.EvmKeeper),
		evmante.NewEthSigVerificationDecorator(options.EvmKeeper),
		evmante.NewEthAccountVerificationDecorator(options.AccountKeeper, options.EvmKeeper),
		evmante.NewEthGasConsumeDecorator(options.EvmKeeper, options.MaxTxGasWanted),
		evmante.NewCanTransferDecorator(options.EvmKeeper),
		evmante.NewEthIncrementSenderSequenceDecorator(options.AccountKeeper, options.EvmKeeper), // innermost AnteDecorator.
	)
}

const (
	secp256k1VerifyCost uint64 = 21000
)

func DefaultSigGasConsumer(
	meter storetypes.GasMeter, sig signing.SignatureV2, params authtypes.Params,
) error {
	// support for ethereum ECDSA secp256k1 keys
	_, ok := sig.PubKey.(*ethsecp256k1.PubKey)
	if ok {
		meter.ConsumeGas(secp256k1VerifyCost, "ante verify: eth_secp256k1")
		return nil
	}

	return ante.DefaultSigVerificationGasConsumer(meter, sig, params)
}

func Recover(logger log.Logger, err *error) {
	if r := recover(); r != nil {
		*err = errorsmod.Wrapf(errorsmod.ErrPanic, "%v", r)

		if e, ok := r.(error); ok {
			logger.Error(
				"ante handler panicked",
				"error", e,
				"stack trace", string(debug.Stack()),
			)
		} else {
			logger.Error(
				"ante handler panicked",
				"recover", fmt.Sprintf("%v", r),
			)
		}
	}
}
