package services

import (
	"fmt"
	"os"
	"reflect"
	"unsafe"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/joho/godotenv"
)

type (
	bee struct {
		Address string `json:"addr"`
		PK      string `json:"pk"`
	}

	logHandler struct {
		format log.Format
	}
)

func (l *logHandler) Log(r *log.Record) error {
	fmt.Print(string(l.format.Format(r)))
	return nil
}

func configureLog(level log.Lvl) {
	glog := log.NewGlogHandler(&logHandler{
		format: log.TerminalFormat(true),
	})

	glog.Verbosity(level)
	log.Root().SetHandler(glog)
}

func getEnvVariable(key string) string {
	err := godotenv.Load("../.env")
	if err != nil {
		panic("Error loading .env file")
	}
	return os.Getenv(key)
}

// Init client to handle communication with GETH protocol

var endpointURL = getEnvVariable("RPC_ENDPOINT")

func newRPCClient() *ethclient.Client {
	client, err := ethclient.Dial(endpointURL)
	if err != nil {
		panic(err)
	}
	return client
}

func InitRPCClient() *rpc.Client {
	var clientVal reflect.Value
	clientVal = reflect.ValueOf(newRPCClient()).Elem()
	fieldStruct := clientVal.FieldByName("c")
	clientPointer := reflect.NewAt(fieldStruct.Type(), unsafe.Pointer(fieldStruct.UnsafeAddr())).Elem()
	finalClient, _ := clientPointer.Interface().(*rpc.Client)
	return finalClient
}
