package main

import (
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/btcsuite/btcd/rpcclient"
)

var btcClient *rpcclient.Client
var feeRWLock sync.RWMutex
var feePerVByte float64 // satoshi / byte

type BlockCypherFeeResponse struct {
	HighFee   uint `json:"high_fee_per_kb"`
	MediumFee uint `json:"medium_fee_per_kb"`
	LowFee    uint `json:"low_fee_per_kb"`
}

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

	go func() {
		for {
			func() {
				feeRWLock.Lock()
				defer func() {
					feeRWLock.Unlock()
					time.Sleep(1 * time.Minute)
				}()
				response, err := http.Get("https://api.blockcypher.com/v1/btc/main")
				if err != nil {
					feePerVByte = -1
					return
				}
				responseData, err := ioutil.ReadAll(response.Body)
				if err != nil {
					feePerVByte = -1
					return
				}
				var responseBody BlockCypherFeeResponse
				err = json.Unmarshal(responseData, &responseBody)
				if err != nil {
					feePerVByte = -1
					return
				}
				feePerVByte = float64(responseBody.MediumFee) / 1024
			}()
		}
	}()
}

func importBTCAddressToFullNode(btcAddress string) error {
	err := btcClient.ImportAddressRescan(btcAddress, "", false)
	return err
}
