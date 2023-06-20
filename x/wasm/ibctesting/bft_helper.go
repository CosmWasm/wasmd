package ibctesting

import (
	"fmt"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	cmttypes "github.com/cometbft/cometbft/types"
	"time"
)

// MakeExtCommit
// Deprecated: should be cmttypes.MakeExtCommit instead. See https://github.com/cometbft/cometbft/issues/1001
func MakeExtCommit(
	blockID cmttypes.BlockID,
	height int64,
	round int32,
	voteSet *cmttypes.VoteSet,
	validators []cmttypes.PrivValidator,
	now time.Time,
	extEnabled bool,
) (*cmttypes.ExtendedCommit, error) {

	// all sign
	for i := 0; i < len(validators); i++ {
		pubKey, err := validators[i].GetPubKey()
		if err != nil {
			return nil, fmt.Errorf("can't get pubkey: %w", err)
		}
		vote := &cmttypes.Vote{
			ValidatorAddress: pubKey.Address(),
			ValidatorIndex:   int32(i),
			Height:           height,
			Round:            round,
			Type:             cmtproto.PrecommitType,
			BlockID:          blockID,
			Timestamp:        now,
		}
		if extEnabled {
			vote.Extension = []byte(`my-vote-extension`)
		}
		_, err = signAddVote(validators[i], vote, voteSet, extEnabled)
		if err != nil {
			return nil, err
		}
	}

	var enableHeight int64
	if extEnabled {
		enableHeight = height
	}

	return voteSet.MakeExtendedCommit(cmttypes.ABCIParams{VoteExtensionsEnableHeight: enableHeight}), nil
}

func signAddVote(privVal cmttypes.PrivValidator, vote *cmttypes.Vote, voteSet *cmttypes.VoteSet, extEnabled bool) (bool, error) {
	//if vote.Type != voteSet.signedMsgType {
	//	return false, fmt.Errorf("vote and voteset are of different types; %d != %d", vote.Type, voteSet.signedMsgType)
	//}
	if _, err := signAndCheckVote(vote, privVal, voteSet.ChainID(), extEnabled && (vote.Type == cmtproto.PrecommitType)); err != nil {
		return false, err
	}
	return voteSet.AddVote(vote)
}

func signAndCheckVote(
	vote *cmttypes.Vote,
	privVal cmttypes.PrivValidator,
	chainID string,
	extensionsEnabled bool,
) (bool, error) {
	v := vote.ToProto()
	if err := privVal.SignVote(chainID, v); err != nil {
		// Failing to sign a vote has always been a recoverable error, this function keeps it that way
		return true, err // true = recoverable
	}
	vote.Signature = v.Signature

	isPrecommit := vote.Type == cmtproto.PrecommitType
	if !isPrecommit && extensionsEnabled {
		// Non-recoverable because the caller passed parameters that don't make sense
		return false, fmt.Errorf("only Precommit votes may have extensions enabled; vote type: %d", vote.Type)
	}

	isNil := vote.BlockID.IsZero()
	extData := len(v.Extension) > 0

	if !extensionsEnabled && extData ||
		extensionsEnabled && !extData {
		return false, fmt.Errorf(
			"extensions must be present IFF vote is a non-nil Precommit; present %t, vote type %d, is nil %t",
			extData,
			vote.Type,
			isNil,
		)
	}

	vote.ExtensionSignature = nil
	if extensionsEnabled {
		vote.ExtensionSignature = v.ExtensionSignature
	}
	vote.Timestamp = v.Timestamp

	return true, nil
}
