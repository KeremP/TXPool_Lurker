package main

import (

"context"
"fmt"
"log"
"os"
"github.com/joho/godotenv"

"github.com/ethereum/go-ethereum/ethclient"

)


func getEnvVariable(key string) string {
  err:= godotenv.Load("../.env")
  if err != nil {
    log.Fatalf("Error loading .env file")
  }
  return os.Getenv(key)
}


func main(){
	var inufra_url = getEnvVariable("RPC_ENDPOINT")
	client,err := ethclient.Dial(inufra_url)

	if err != nil {
		log.Fatalf("Error:",err)
	}else{
		fmt.Println("Connection successful")
	}

	blockNumber,err := client.BlockNumber(context.Background())
	if err != nil {
		log.Fatalf("Error:",err)
	}else{
		fmt.Println(blockNumber)
	}

}
