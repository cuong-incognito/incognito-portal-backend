package main

import (
	"flag"
	"io/ioutil"
	"log"
)

var ENABLE_PROFILER bool
var serviceCfg Config

type BTCFullnodeConfig struct {
	Address  string `json:"address"`
	User     string `json:"user"`
	Password string `json:"pass"`
	Https    bool   `json:"https"`
}

type Config struct {
	APIPort      int               `json:"apiport"`
	MongoAddress string            `json:"mongo"`
	MongoDB      string            `json:"mongodb"`
	BTCFullnode  BTCFullnodeConfig `json:"btcfullnode"`
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
	ENABLE_PROFILER = *argProfiler
	serviceCfg = tempCfg
}
