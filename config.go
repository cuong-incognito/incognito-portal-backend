package main

import (
	"flag"
	"io/ioutil"
	"log"

	"github.com/btcsuite/btcd/chaincfg"
)

var ENABLE_PROFILER bool
var serviceCfg Config
var BTCChainCfg *chaincfg.Params
var BTCTokenID string

type BTCFullnodeConfig struct {
	Address  string `json:"address"`
	User     string `json:"user"`
	Password string `json:"pass"`
	Https    bool   `json:"https"`
}

type Config struct {
	APIPort           int               `json:"apiport"`
	MongoAddress      string            `json:"mongo"`
	MongoDB           string            `json:"mongodb"`
	BTCFullnode       BTCFullnodeConfig `json:"btcfullnode"`
	BlockchainFeeHost string            `json:"blockchainfee"`
	Net               string            `json:"net"`
}

func readConfigAndArg() {
	data, err := ioutil.ReadFile("./cfg.json")
	if err != nil {
		log.Println(err)
		// return
	}
	var tempCfg Config
	if data != nil {
		err = json.Unmarshal(data, &tempCfg)
		if err != nil {
			panic(err)
		}
	}

	argProfiler := flag.Bool("profiler", false, "set profiler")
	flag.Parse()
	if tempCfg.APIPort == 0 {
		tempCfg.APIPort = DefaultAPIPort
	}
	if tempCfg.MongoAddress == "" {
		tempCfg.MongoAddress = DefaultMongoAddress
	}
	if tempCfg.MongoDB == "" {
		tempCfg.MongoDB = DefaultMongoDB
	}
	if tempCfg.Net == "test" {
		BTCChainCfg = &chaincfg.TestNet3Params
		BTCTokenID = TESTNET_BTC_ID
	} else if tempCfg.Net == "main" {
		BTCChainCfg = &chaincfg.MainNetParams
		BTCTokenID = MAINNET_BTC_ID
	} else {
		panic("Invalid config network Bitcoin")
	}
	ENABLE_PROFILER = *argProfiler
	serviceCfg = tempCfg
}
