package interest

import (
	"time"

	"github.com/xar-network/xar-network/x/interest/internal/types"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

const (
	// Do not apply inflation if less than this period has elapsed since last accrual
	minimumMintingPeriod = 10 * time.Second
)

// BeginBlocker mints new tokens for the previous block.
func BeginBlocker(ctx sdk.Context, k Keeper) {
	state := k.GetState(ctx)
	blockTime := ctx.BlockTime()

	// Gate-keep this functionality based on time since last block to prevent a cascade of blocks
	if blockTime.Sub(state.LastAppliedTime) < minimumMintingPeriod {
		return
	}

	if ctx.BlockHeight() == state.LastAppliedHeight.Int64()+1 {
		return
	}

	totalTokenSupply := k.TotalTokenSupply(ctx)

	mintedCoins := applyInterest(&state, totalTokenSupply, blockTime)
	state.LastAppliedHeight = sdk.NewInt(ctx.BlockHeight())

	if mintedCoins.IsZero() {
		return
	}

	k.Logger(ctx).Info("Interest minted coins", toKeyValuePairs(mintedCoins)...)

	k.SetState(ctx, state)
	err := k.MintCoins(ctx, mintedCoins)
	if err != nil {
		panic(err)
	}

	err = k.AddMintedCoins(ctx, mintedCoins)
	if err != nil {
		panic(err)
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeMint,
			sdk.NewAttribute(types.AttributeKeyAmount, mintedCoins.String()),
			sdk.NewAttribute(types.AttributeKeyEvent, "true"),
		),
	)
}

func applyInterest(state *InterestState, totalTokenSupply sdk.Coins, currentTime time.Time) sdk.Coins {
	lastAccrual := state.LastAppliedTime
	mintedCoins := sdk.Coins{}

	// Interest may be set to start in the future. Do nothing in that case.
	if currentTime.Before(lastAccrual) {
		return mintedCoins
	}

	state.LastAppliedTime = currentTime

	for i, asset := range state.InterestAssets {
		supply := totalTokenSupply.AmountOf(asset.Denom)

		accum, minted := calculateInterest(asset.Accum, supply, asset.Interest, lastAccrual, currentTime)

		if minted.IsPositive() { // Coins.IsValid() considers any coin of amount 0 to be invalid, so filter 0 coins.
			mintedCoins = append(mintedCoins, sdk.NewCoin(asset.Denom, minted))
		}

		asset.Accum = accum
		state.InterestAssets[i] = asset
	}

	return mintedCoins.Sort()
}

func calculateInterest(prevAccum sdk.Dec, supply sdk.Int, annualInterest sdk.Dec, lastAccrual, currentTime time.Time) (accum sdk.Dec, minted sdk.Int) {
	annualNS := 365 * 24 * time.Hour.Nanoseconds()

	periodNS := sdk.NewDec(currentTime.Sub(lastAccrual).Nanoseconds())
	accum = annualInterest.MulInt(supply).Mul(periodNS).Add(prevAccum)

	minted = accum.Quo(sdk.NewDec(annualNS)).TruncateInt()
	accum = accum.Sub(minted.MulRaw(annualNS).ToDec())

	return
}

// For use in logging
func toKeyValuePairs(coins sdk.Coins) (res []interface{}) {
	for _, coin := range coins {
		res = append(res, coin.Denom, coin.Amount.String())
	}
	return
}