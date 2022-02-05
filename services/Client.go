package services

import (

        "flag"
        "fmt"
        "log"
        "os"
        "reflect"
        "unsafe"
        "github.com/ethereum/go-ethereum/ethclient"
        "github.com/ethereum/go-ethereum/rpc"
        "github.com/joho/godotenv"

)

func getEnvVariable(key string) string {
  err:= godotenv.Load(".env")
  if err != nil {
    log.Fatalf("Error loading .env file")
  }
  return os.Getenv(key)
}

// Init client to handle communication with GETH protocol

var endpointURL = getEnvVariable("RPC_ENDPOINT")

func DialClient() *ethclient.Client {
  client, err := ethclient.Dial(endpointURL)
  if err != nil {
    log.Fatalln(err)
  }
  return client
}

func initRPCClient() *rpc.Client {
  var clientVal = reflect.Value
  clientVal = reflect.ValueOf(DialClient()).Elem()
  fieldStruct := clientValue.FieldByName("c")
  clientPointer := reflect.NewAt(fieldStruct.Type(), unsafe.Pointer(fieldStruct.UnsafeAddr())).Elem()
  finalClient, _ := clientPointer.Interface().(*rpc.Client)
  return finalClient
}
