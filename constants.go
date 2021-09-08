package main

import "time"

const (
	DB_OPERATION_TIMEOUT time.Duration = 1 * time.Second
)

const (
	version             = "0.9.5"
	DefaultAPIPort      = 9001
	DefaultMongoAddress = ""
	DefaultMongoDB      = "portal"

	BTCMinConf = 0
	BTCMaxConf = 9999999
)

const (
	TESTNET_BTC_ID = "4584d5e9b2fc0337dfb17f4b5bb025e5b82c38cfa4f54e8a3d4fcdd03954ff82"
	MAINNET_BTC_ID = "b832e5d3b1f01a4f0623f7fe91d6673461e1f5d37d91fe78c5c2e6183ff39696"
)
