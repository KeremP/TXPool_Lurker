package main

import (

"context"
"fmt"
"log"

"github.com/ethereum/go-ethereum/ethclient"

)

func main(){
	client,err := ethclient.Dial("")

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
