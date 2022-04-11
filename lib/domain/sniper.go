package domain

import "math/big"

type (
	Sniper struct {
		// Address of smart contract
		AddressTrigger string
		// Address of base/paired token. WETH by default.
		AddressTargetPaired string
		// Address of target token.
		AddressTargetToken string
		// Min liquidity required to snipe target token.
		MinimumLiquidity *big.Int
		// ID of the network. i.e. 1 for mainnet.
		ChainID *big.Int
	}
)

func NewSniper(
	at, atp, att string,
	ml, ci *big.Int,
) Sniper {
	return Sniper{
		AddressTrigger:      at,
		AddressTargetPaired: atp,
		AddressTargetToken:  att,
		MinimumLiquidity:    ml,
		ChainID:             ci,
	}
}
