package keeper

import (
	"bytes"
	"fmt"
	"strconv"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/lavanet/lava/relayer/sigs"
	"github.com/lavanet/lava/x/conflict/types"
	pairingtypes "github.com/lavanet/lava/x/pairing/types"
)

func (k Keeper) ValidateFinalizationConflict(ctx sdk.Context, conflictData *types.FinalizationConflict, clientAddr sdk.AccAddress) error {
	return nil
}

func (k Keeper) ValidateResponseConflict(ctx sdk.Context, conflictData *types.ResponseConflict, clientAddr sdk.AccAddress) error {
	//1. validate mismatching data
	chainID := conflictData.ConflictRelayData0.Request.ChainID
	if chainID != conflictData.ConflictRelayData1.Request.ChainID {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", chainID, conflictData.ConflictRelayData1.Request.ChainID)
	}
	block := conflictData.ConflictRelayData0.Request.BlockHeight
	if block != conflictData.ConflictRelayData1.Request.BlockHeight {
		return fmt.Errorf("mismatching request parameters between providers %d, %d", block, conflictData.ConflictRelayData1.Request.BlockHeight)
	}
	if conflictData.ConflictRelayData0.Request.ApiId != conflictData.ConflictRelayData1.Request.ApiId {
		return fmt.Errorf("mismatching request parameters between providers %d, %d", conflictData.ConflictRelayData0.Request.ApiId, conflictData.ConflictRelayData1.Request.ApiId)
	}
	if conflictData.ConflictRelayData0.Request.ApiUrl != conflictData.ConflictRelayData1.Request.ApiUrl {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.ApiUrl, conflictData.ConflictRelayData1.Request.ApiUrl)
	}
	if !bytes.Equal(conflictData.ConflictRelayData0.Request.Data, conflictData.ConflictRelayData1.Request.Data) {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.Data, conflictData.ConflictRelayData1.Request.Data)
	}
	if conflictData.ConflictRelayData0.Request.ApiUrl != conflictData.ConflictRelayData1.Request.ApiUrl {
		return fmt.Errorf("mismatching request parameters between providers %s, %s", conflictData.ConflictRelayData0.Request.ApiUrl, conflictData.ConflictRelayData1.Request.ApiUrl)
	}
	if conflictData.ConflictRelayData0.Request.RequestBlock != conflictData.ConflictRelayData1.Request.RequestBlock {
		return fmt.Errorf("mismatching request parameters between providers %d, %d", conflictData.ConflictRelayData0.Request.RequestBlock, conflictData.ConflictRelayData1.Request.RequestBlock)
	}
	epochStart, _ := k.epochstorageKeeper.GetEpochStartForBlock(ctx, uint64(block))
	k.pairingKeeper.VerifyPairingData(ctx, chainID, clientAddr, epochStart)
	//2. validate signer
	clientEntry, err := k.epochstorageKeeper.GetStakeEntryForClientEpoch(ctx, chainID, clientAddr, epochStart)
	if err != nil || clientEntry == nil {
		return fmt.Errorf("did not find a stake entry for consumer %s on epoch %d, chainID %s error: %s", clientAddr, epochStart, chainID, err.Error())
	}
	verifyClientAddrFromSignatureOnRequest := func(conflictRelayData types.ConflictRelayData) error {
		pubKey, err := sigs.RecoverPubKeyFromRelay(*conflictRelayData.Request)
		if err != nil {
			return fmt.Errorf("invalid consumer signature in relay request %+v , error: %s", conflictRelayData.Request, err.Error())
		}
		derived_clientAddr, err := sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			return fmt.Errorf("invalid consumer address from signature in relay request %+v , error: %s", conflictRelayData.Request, err.Error())
		}
		if !derived_clientAddr.Equals(clientAddr) {
			return fmt.Errorf("mismatching consumer address signature and msg.Creator in relay request %s , %s", derived_clientAddr, clientAddr)
		}
		return nil
	}
	err = verifyClientAddrFromSignatureOnRequest(*conflictData.ConflictRelayData0)
	if err != nil {
		return err
	}
	err = verifyClientAddrFromSignatureOnRequest(*conflictData.ConflictRelayData1)
	if err != nil {
		return err
	}
	//3. validate providers signatures and stakeEntry for that epoch
	providerAddressFromRelayReplyAndVerifyStakeEntry := func(request *pairingtypes.RelayRequest, reply *pairingtypes.RelayReply, first bool) (providerAddress sdk.AccAddress, err error) {
		print_st := "first"
		if !first {
			print_st = "second"
		}
		pubKey, err := sigs.RecoverPubKeyFromRelayReply(reply, request)
		if err != nil {
			return nil, fmt.Errorf("RecoverProviderPubKeyFromQueryAndAllDataHash %s provider: %w", print_st, err)
		}
		providerAddress, err = sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			return nil, fmt.Errorf("AccAddressFromHex %s provider: %w", print_st, err)
		}
		_, err = k.epochstorageKeeper.GetStakeEntryForProviderEpoch(ctx, chainID, providerAddress, epochStart)
		if err != nil {
			return nil, fmt.Errorf("did not find a stake entry for %s provider %s on epoch %d, chainID %s error: %s", print_st, providerAddress, epochStart, chainID, err.Error())
		}
		return providerAddress, nil
	}
	providerAccAddress0, err := providerAddressFromRelayReplyAndVerifyStakeEntry(conflictData.ConflictRelayData0.Request, conflictData.ConflictRelayData0.Reply, true)
	if err != nil {
		return err
	}
	providerAccAddress1, err := providerAddressFromRelayReplyAndVerifyStakeEntry(conflictData.ConflictRelayData1.Request, conflictData.ConflictRelayData1.Reply, false)
	if err != nil {
		return err
	}
	//4. validate finalization
	validateResponseFinalizationData := func(expectedAddress sdk.AccAddress, response *pairingtypes.RelayReply, request *pairingtypes.RelayRequest, first bool) (err error) {
		print_st := "first"
		if !first {
			print_st = "second"
		}

		pubKey, err := sigs.RecoverPubKeyFromResponseFinalizationData(response, request, clientAddr)
		if err != nil {
			return fmt.Errorf("RecoverPubKey %s provider ResponseFinalizationData: %w", print_st, err)
		}
		derived_providerAccAddress, err := sdk.AccAddressFromHex(pubKey.Address().String())
		if err != nil {
			return fmt.Errorf("AccAddressFromHex %s provider ResponseFinalizationData: %w", print_st, err)
		}
		if !derived_providerAccAddress.Equals(expectedAddress) {
			return fmt.Errorf("mismatching %s provider address signature and responseFinazalizationData %s , %s", print_st, derived_providerAccAddress, expectedAddress)
		}
		//validate the responses are finalized
		if !k.specKeeper.IsFinalizedBlock(ctx, chainID, request.RequestBlock, response.LatestBlock) {
			return fmt.Errorf("block isn't finalized on %s provider! %d,%d ", print_st, request.RequestBlock, response.LatestBlock)
		}
		return nil
	}
	err = validateResponseFinalizationData(providerAccAddress0, conflictData.ConflictRelayData0.Reply, conflictData.ConflictRelayData0.Request, true)
	if err != nil {
		return err
	}
	err = validateResponseFinalizationData(providerAccAddress1, conflictData.ConflictRelayData1.Reply, conflictData.ConflictRelayData1.Request, true)
	if err != nil {
		return err
	}
	//5. validate mismatching responses
	if bytes.Equal(conflictData.ConflictRelayData0.Reply.Data, conflictData.ConflictRelayData1.Reply.Data) {
		return fmt.Errorf("no conflict between providers data responses, its the same")
	}
	return nil
}

func (k Keeper) ValidateSameProviderConflict(ctx sdk.Context, conflictData *types.FinalizationConflict, clientAddr sdk.AccAddress) error {
	return nil
}

func (k Keeper) AllocateNewConflictVote(ctx sdk.Context) string {
	found := false
	var index uint64 = 0
	var sIndex string
	for !found {
		index++
		sIndex = strconv.FormatUint(index, 10)
		_, found = k.GetConflictVote(ctx, sIndex)
	}
	return sIndex
}

func (k Keeper) HandleAndCloseVote(ctx sdk.Context, ConflictVote types.ConflictVote) {
	//1) make a list of all voters that didnt vote
	//3) count votes

	//all wrong voters are punished
	//add stake as wieght
	//votecounts is bigint
	//valid only if one of the votes is bigger than 50% from total
	//punish providers that didnt vote - discipline/jail + bail = 20%stake + slash 5%stake
	//(dont add jailed providers to voters)
	//if strong majority punish wrong providers - jail from start of memory to end + slash 100%stake
	//reward pool is the slashed amount from all punished providers
	//reward to stake - client 50%, the original provider 10%, 20% the voters

	var totalVotes int64 = 0
	var firstProviderVotes int64 = 0
	var secondProviderVotes int64 = 0
	var noneProviderVotes int64 = 0
	var providersToPunish []string

	intVal := map[bool]int64{false: 0, true: 1}
	for address, vote := range ConflictVote.VotersHash {
		//switch
		totalVotes++
		firstProviderVotes += intVal[vote.Result == types.Provider0]
		secondProviderVotes += intVal[vote.Result == types.Provider1]
		noneProviderVotes += intVal[vote.Result == types.None]
		if vote.Result == types.NoVote {
			providersToPunish = append(providersToPunish, address)
		}
	}

	//2) check that we have enough votes
	if firstProviderVotes > secondProviderVotes && firstProviderVotes > noneProviderVotes {
		if sdk.NewDecWithPrec(firstProviderVotes, 0).QuoInt64(totalVotes).LT(k.MajorityPercent(ctx)) {
			providersToPunish = []string{}
		}
		providersToPunish = append(providersToPunish, ConflictVote.SecondProvider.Account)
	} else if secondProviderVotes > noneProviderVotes {
		if sdk.NewDecWithPrec(secondProviderVotes, 0).QuoInt64(totalVotes).LT(k.MajorityPercent(ctx)) {
			providersToPunish = []string{}
		}
		providersToPunish = append(providersToPunish, ConflictVote.FirstProvider.Account)
	} else {
		if sdk.NewDecWithPrec(noneProviderVotes, 0).QuoInt64(totalVotes).LT(k.MajorityPercent(ctx)) {
			providersToPunish = []string{}
		}
		providersToPunish = append(providersToPunish, ConflictVote.FirstProvider.Account, ConflictVote.SecondProvider.Account)
	}

	//4) reward voters and providers
	//5) punish fraud providers and voters that didnt vote
	//6) cleanup storage
	k.RemoveConflictVote(ctx, ConflictVote.Index)
	//7) unstake punished providers
	//8) event?
}
