package services

import (
        "context"
        "fmt"

        "github.com/ethereum/go-ethereum/common"
        "github.com/ethereum/go-ethereum/core/types"
        // "github.com/ethereum/go-ethereum/ethclient"
        "github.com/ethereum/go-ethereum/rpc"

)

func StreamTx(rpcClient *rpc.Client) {

  TxChannel := make(chan common.Hash)

  rpcClient.EthSubscribe(
    context.Background(), TxChannel, "newPendingTransactions",
  )

  client := DialClient()
  fmt.Println("Subscribed to mempool")

  chainID, _ := client.NetworkID(context.Background())

  signer := types.NewEIP155Signer(chainID)

  for {
    select{

    case transactionHash := <-TxChannel:
      fmt.Println("Tx Detected")
      tx, pending, _ := client.TransactionByHash(context.Background(), transactionHash)

      if pending {
        _,_ = signer.Sender(tx)
        handleTX(tx)
      }
    }
  }
}

func handleTX(tx *types.Transaction) {
    fmt.Println("New TX detected. Hash: ", tx.Hash().Hex())
}
