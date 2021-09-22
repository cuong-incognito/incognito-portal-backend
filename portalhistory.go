package main

import (
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"log"
	"sync"
)

type PortalShieldHistory struct {
	Amount           uint64 `json:"amount,omitempty"`
	ExternalTxID     string `json:"externalTxID"`
	IncognitoAddress string `json:"incognitoAddress,omitempty"`
	Status           int    `json:"status"`
	Time             int64  `json:"time,omitempty"`
	Confirmations    int64  `json:"confirmations"`
}

const ShieldStatusFailed = 0
const ShieldStatusSuccess = 1
const ShieldStatusPending = 2
const ShieldStatusProcessing = 3

func convertBTCAmtToPBTCAmt(btcAmt float64) uint64 {
	return uint64(btcAmt*1e8+0.5) * 10
}

func getStatusFromConfirmation(confirmationBlks int) (status int) {
	status = ShieldStatusPending
	if confirmationBlks > 0 {
		status = ShieldStatusProcessing
	}
	return
}

func ParseUTXOsToPortalShieldHistory(
	utxos []btcjson.ListUnspentResult, incAddress string,
) ([]PortalShieldHistory, error) {
	histories := []PortalShieldHistory{}

	var wg sync.WaitGroup
	result := make(chan PortalShieldHistory, len(utxos))
	for _, u := range utxos {
		u := u
		wg.Add(1)
		go func() {
			defer wg.Done()
			status := getStatusFromConfirmation(int(u.Confirmations))
			txIDHash, err := chainhash.NewHashFromStr(u.TxID)
			if err != nil {
				log.Printf("Could not new hash from external tx id %v - Error %v\n", u.TxID, err)
				return
			}
			tx, err := btcClient.GetTransaction(txIDHash)
			if err != nil {
				log.Printf("Could not get external tx id %v - Error %v\n", u.TxID, err)
				return
			}
			result <- PortalShieldHistory{
				Amount:           convertBTCAmtToPBTCAmt(u.Amount),
				ExternalTxID:     u.TxID,
				IncognitoAddress: incAddress,
				Status:           status,
				Time:             tx.Time * 1000, // convert to msec
				Confirmations:    u.Confirmations,
			}
		}()
	}
	wg.Wait()
	close(result)

	for h := range result {
		histories = append(histories, h)
	}

	return histories, nil
}
