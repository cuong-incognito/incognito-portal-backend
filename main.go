package main

import (
	"net/http"
	_ "net/http/pprof"

	jsoniter "github.com/json-iterator/go"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

func main() {
	readConfigAndArg()
	err := connectDB()
	if err != nil {
		panic(err)
	}
	initPortalService()
	go startGinService()
	if ENABLE_PROFILER {
		http.ListenAndServe("localhost:8091", nil)
	}
	select {}
}
