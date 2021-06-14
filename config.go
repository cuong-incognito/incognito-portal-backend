package main

import (
	"flag"
	"io/ioutil"
	"log"
)

var ENABLE_PROFILER bool
var serviceCfg Config

type Config struct {
	APIPort      int    `json:"apiport"`
	MongoAddress string `json:"mongo"`
	MongoDB      string `json:"mongodb"`
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
