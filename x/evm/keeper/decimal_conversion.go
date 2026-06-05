package keeper

import "math/big"

// EvmDecimalConversionFactor is the factor to convert between Cosmos base denom (8 decimals)
// and EVM wei (18 decimals). 1 umec = 10^10 wei.
var EvmDecimalConversionFactor = new(big.Int).Exp(big.NewInt(10), big.NewInt(10), nil)

// UmecToWei converts a Cosmos bank amount (umec, 8 decimals) to EVM wei (18 decimals).
// wei = umec * 10^10
func UmecToWei(umecAmount *big.Int) *big.Int {
	return new(big.Int).Mul(umecAmount, EvmDecimalConversionFactor)
}

// WeiToUmec converts an EVM wei amount (18 decimals) to Cosmos bank amount (umec, 8 decimals).
// umec = wei / 10^10 (truncated)
func WeiToUmec(weiAmount *big.Int) *big.Int {
	return new(big.Int).Div(weiAmount, EvmDecimalConversionFactor)
}

// WeiToUmecCeil converts an EVM wei amount to umec, rounding up to ensure fees are never underpaid.
// umec = ceil(wei / 10^10)
func WeiToUmecCeil(weiAmount *big.Int) *big.Int {
	result := new(big.Int).Div(weiAmount, EvmDecimalConversionFactor)
	remainder := new(big.Int).Mod(weiAmount, EvmDecimalConversionFactor)
	if remainder.Sign() > 0 {
		result.Add(result, big.NewInt(1))
	}
	return result
}

// IsWeiAmountDivisible checks if a wei amount is exactly divisible by 10^10 (no precision loss).
func IsWeiAmountDivisible(weiAmount *big.Int) bool {
	if weiAmount == nil || weiAmount.Sign() == 0 {
		return true
	}
	remainder := new(big.Int).Mod(weiAmount, EvmDecimalConversionFactor)
	return remainder.Sign() == 0
}
