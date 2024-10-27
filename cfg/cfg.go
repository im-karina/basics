package cfg

import (
	"errors"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var Environment string
var IsProd bool
var IsStg bool
var IsDev bool
var ErrCannotRunInProd = errors.New("cannot run in prod")

var DbConnectionString string

var ListenAddr string

var HttpsCertPath string
var HttpsKeyPath string

func Load() {
	err := godotenv.Load("./data/.env")
	if err != nil {
		panic(err)
	}
	environment := os.Getenv("ENVIRONMENT")
	switch environment {
	case "", "DEV":
		Environment = "DEV"
		IsDev = true
	case "STG":
		Environment = "STG"
		IsStg = true
	case "PROD":
		Environment = "PROD"
		IsProd = true
	default:
		log.Fatalf("unknown environment: '%v'\nexpected one of: 'DEV' (default), 'STG', 'PROD'\n", environment)
	}

	var ok bool
	DbConnectionString, ok = os.LookupEnv("DATABASE_URL")
	if !ok {
		DbConnectionString = fmt.Sprintf("file:data/%v.sqlite3", Environment)
	}

	ListenAddr = ":3000"
}
