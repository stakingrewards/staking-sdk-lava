// Code generated by protoc-gen-gogo. DO NOT EDIT.
// source: lavanet/lava/epochstorage/stake_entry.proto

package types

import (
	"cosmossdk.io/math"
)

func (se StakeEntry) EffectiveStake() math.Int {
	effective := se.Stake.Amount
	if se.DelegateLimit.Amount.LT(se.DelegateTotal.Amount) {
		effective.Add(se.DelegateLimit.Amount)
	} else {
		effective.Add(se.DelegateTotal.Amount)
	}
	return effective
}