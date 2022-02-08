package services

import (

	     // "fmt"
       // "log"
       // "context"
       "math/big"

       // "github.com/ethereum/go-ethereum/core/types"

)


func formatEthWeiToEther(etherAmount *big.Int) float64{
  var base, exponent = big.NewInt(10), big.NewInt(18)

  denominator := base.Exp(base, exponent, nil)

  tokenAmountFloat := new(big.Float).SetInt(etherAmount)
  denominatorFloat := new(big.Float).SetInt(denominator)

  out,_ := new(big.Float).Quo(tokenAmountFloat, denominatorFloat).Float64()
  return out
}
