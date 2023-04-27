package xadr8

import (
	"encoding/json"
	"errors"

	errorsmod "cosmossdk.io/errors"

	storetypes "github.com/cosmos/cosmos-sdk/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	channeltypes "github.com/cosmos/ibc-go/v7/modules/core/04-channel/types"
)

type PacketId = channeltypes.PacketId

type Keeper struct {
	storeKey      storetypes.StoreKey
	channelKeeper ChannelKeeper
	auth          Authorization
}
type Authorization interface {
	// IsAuthorized returns if actor authorized to register a callback for a package
	// returning an error aborts the operations
	IsAuthorized(ctx sdk.Context, actor sdk.AccAddress) (bool, error)
}

var _ Authorization = AuthorizationFn(nil)

type AuthorizationFn func(ctx sdk.Context, actor sdk.AccAddress) (bool, error)

func (a AuthorizationFn) IsAuthorized(ctx sdk.Context, actor sdk.AccAddress) (bool, error) {
	return a(ctx, actor)
}

func NewKeeper(storeKey storetypes.StoreKey, channelKeeper ChannelKeeper, auth Authorization) *Keeper {
	// todoL ensure non nil values
	return &Keeper{
		storeKey:      storeKey,
		channelKeeper: channelKeeper,
		auth:          auth,
	}
}

func (k Keeper) RegisterPacketProcessedCallback(
	ctx sdk.Context,
	sender sdk.AccAddress,
	packetID PacketId,
	limit uint64,
	meta CustomDataI,
) error {
	ok, err := k.auth.IsAuthorized(ctx, sender)
	if err != nil {
		return errorsmod.Wrap(err, "authorization")
	}
	if !ok {
		return errors.New("unauthorized") // todo: register proper error
	}
	// do some sanity checks. copied from ics29 async fees
	//
	nextSeqSend, found := k.channelKeeper.GetNextSequenceSend(ctx, packetID.PortId, packetID.ChannelId)
	if !found {
		return channeltypes.ErrSequenceSendNotFound.Wrapf("channel does not exist, portID: %s, channelID: %s", packetID.PortId, packetID.ChannelId)
	}

	// only allow packets which have been sent
	if packetID.Sequence >= nextSeqSend {
		return channeltypes.ErrPacketNotSent
	}

	// only allow packets which have not completed the packet life cycle
	// todo: better use channelKeeper.HasPacketCommitment ???

	if bz := k.channelKeeper.GetPacketCommitment(ctx, packetID.PortId, packetID.ChannelId, packetID.Sequence); len(bz) == 0 {
		return channeltypes.ErrPacketCommitmentNotFound.Wrap("packet has already been acknowledged or timed out")
	}
	// as there is no information about the package stored on chain, we can not do more to ensure the
	// actor is authorized.

	// burn the gas to charge the caller
	ctx.GasMeter().ConsumeGas(limit, "register ibc callback hook")

	store := ctx.KVStore(k.storeKey)
	bz, err := json.Marshal(&CallbackData{
		ContractAddr: sender,
		Limit:        limit,
		MetaData:     meta,
	})
	if err != nil {
		return err
	}
	// register with address as we can not check for the correct sender here
	// just allow multiple.They will be cleaned up on packet ack/timeout
	store.Set(BuildCallbackKey(packetID, sender.String()), bz)
	return nil
}

type Executor interface {
	// Do calls the contract to execute the hook. It is executed with the
	// gas limit that was registered with the hook. The operation runs within
	// a cached store so that an error rolls back this state but does not affect
	// the store otherwise.
	Do(ctx sdk.Context, contractAddr sdk.AccAddress, meta CustomDataI) error
}

var _ Executor = ExecutorFn(nil)

type ExecutorFn func(ctx sdk.Context, contractAddr sdk.AccAddress, meta CustomDataI) error

func (e ExecutorFn) Do(ctx sdk.Context, contractAddr sdk.AccAddress, meta CustomDataI) error {
	return e(ctx, contractAddr, meta)
}

func (k Keeper) ExecutePacketCallback(
	ctx sdk.Context,
	packetID PacketId,
	sender string,
	exec Executor,
) error {
	store := ctx.KVStore(k.storeKey)
	key := BuildCallbackKey(packetID, sender)
	if !store.Has(key) { // this cheaper on gas and would handle the majority of packets
		return nil
	}

	// always do housekeeping
	defer func() {
		iter := store.Iterator(PrefixRange(BuildCallbackKeyPrefix(packetID)))
		defer iter.Close()
		for ; iter.Valid(); iter.Next() {
			store.Delete(iter.Key())
		}
	}()

	var o CallbackData
	// deserialize. Should be from proto instead
	if err := json.Unmarshal(store.Get(key), &o); err != nil {
		return err
	}
	if o.ContractAddr.String() != sender {
		// somebody wasted money for a callback they are not allowed to receive.
		// too bad we could not check on register but here we do nothing
		return nil
	}

	// execute callback in a cached store environment so that we can revert
	// on any failure
	cachedCtx, commit := ctx.CacheContext()

	// setup a new gas meter with the defined limit.
	// the gas was paid before by the contract. no need to charge twice
	gm := sdk.NewGasMeter(o.Limit)
	cachedCtx = cachedCtx.WithGasMeter(gm)

	// todo: handle panics for the callback call
	if err := exec.Do(cachedCtx, o.ContractAddr, o.MetaData); err != nil {
		// todo: emit event with error
	} else {
		commit()
	}
	return nil
}

// CallbackData contains the persisted data to execute a callback
// Todo: define in protobuf. Using json de-/serialization only for rapid prototyping
type CallbackData struct {
	Limit        sdk.Gas
	ContractAddr sdk.AccAddress
	MetaData     CustomDataI // todo: should be Any type in proto
}

// CustomDataI is an extension that is persisted as a proto any type and can be used by to add additional custom data
type CustomDataI interface{}

type ChannelKeeper interface {
	GetChannel(ctx sdk.Context, srcPort, srcChan string) (channel channeltypes.Channel, found bool)
	GetPacketCommitment(ctx sdk.Context, portID, channelID string, sequence uint64) []byte
	GetNextSequenceSend(ctx sdk.Context, portID, channelID string) (uint64, bool)
}
