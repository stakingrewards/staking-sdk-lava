package keeper

import (
	"context"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/utils"
	"github.com/lavanet/lava/x/conflict/types"
)

func (k msgServer) ConflictVoteCommit(goCtx context.Context, msg *types.MsgConflictVoteCommit) (*types.MsgConflictVoteCommitResponse, error) {
	ctx := sdk.UnwrapSDKContext(goCtx)
	logger := k.Keeper.Logger(ctx)

	conflictVote, found := k.GetConflictVote(ctx, strconv.FormatUint(msg.VoteID, 10))
	if !found {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_commit", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "invalid vote id")
	}
	if conflictVote.VoteState != types.StateCommit {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_commit", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "vote is not in commit state")
	}
	if _, ok := conflictVote.VotersHash[msg.Creator]; !ok {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_commit", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider is not in the voters list")
	}
	if conflictVote.VotersHash[msg.Creator].Result != types.NoVote {
		return nil, utils.LavaError(ctx, logger, "response_conflict_detection_commit", map[string]string{"provider": msg.Creator, "voteID": strconv.FormatUint(msg.VoteID, 10)}, "provider already commited")
	}

	conflictVote.VotersHash[msg.Creator] = types.Vote{Hash: msg.Hash, Result: types.Commit}
	k.SetConflictVote(ctx, conflictVote)

	return &types.MsgConflictVoteCommitResponse{}, nil
}
