package main

import (
	"github.com/btcsuite/btcd/rpcclient"
)

var btcClient *rpcclient.Client

func initPortalService() {
	err := DBCreatePortalAddressIndex()
	if err != nil {
		panic(err)
	}

	connCfg := &rpcclient.ConnConfig{
		Host:         serviceCfg.BTCFullnode.Address,
		User:         serviceCfg.BTCFullnode.User,
		Pass:         serviceCfg.BTCFullnode.Password,
		HTTPPostMode: true,                          // Bitcoin core only supports HTTP POST mode
		DisableTLS:   !serviceCfg.BTCFullnode.Https, // Bitcoin core does not provide TLS by default
	}
	btcClient, err = rpcclient.New(connCfg, nil)
	if err != nil {
		panic(err)
	}
}

func importBTCAddressToFullNode(btcAddress string) error {
	err := btcClient.ImportAddressRescan(btcAddress, "", false)
	return err
}
