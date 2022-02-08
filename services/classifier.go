package services

import (

      "fmt"
      "github.com/ethereum/go-ethereum/core/types"
	    // "github.com/ethereum/go-ethereum/ethclient"
      ."github.com/logrusorgru/aurora"
)

func classifyTx(tx *types.Transaction) {

    if len(tx.Data()) == 0 {
      fmt.Println()
      fmt.Println(Yellow("New Direct ETH Transfer detected"))
      fmt.Println("Hash: ", tx.Hash().Hex(), " Value: ", formatEthWeiToEther(tx.Value()))
    }else{
      fmt.Println("TX Detected: ", tx.Hash().Hex())
    }
}
