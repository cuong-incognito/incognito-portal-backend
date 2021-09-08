package main

import (
	"github.com/btcsuite/btcd/btcjson"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"log"
	"strconv"
)

type PortalShieldHistory struct {
	Amount           uint64 `json:"amount,omitempty"`
	ExternalTxID     string `json:"externalTxID"`
	IncognitoAddress string `json:"incognitoAddress,omitempty"`
	Status           int    `json:"status"`
	StatusStr        string `json:"statusStr"`
	StatusDetail     string `json:"statusDetail"`
	Time             int64 `json:"time,omitempty"`
	TxType           int    `json:"txType,omitempty"`
	TxTypeStr        string `json:"txTypeStr,omitempty"`
}

const ShieldTxType = 101
const ShieldTxTypeStr = "Shield"

const ShieldStatusFailed = 0
const ShieldStatusSuccess = 1
const ShieldStatusPending = 2
const ShieldStatusProcessing = 3

var ShieldStatusStr = map[int]string{
	ShieldStatusFailed:     "Failed",
	ShieldStatusSuccess:    "Complete",
	ShieldStatusPending:    "Pending",
	ShieldStatusProcessing: "Processing",
}

func convertBTCAmtToPBTCAmt(btcAmt float64) uint64 {
	return uint64(btcAmt*1e8+0.5) * 10
}

func getStatusFromConfirmation(confirmationBlks int) (status int, statusStr, statusDetail string) {
	if confirmationBlks > 0 {
		status = ShieldStatusProcessing
		statusDetail = "The shielding transaction is confirmed with " +
			strconv.Itoa(confirmationBlks) + " blocks."
	} else {
		status = ShieldStatusPending
		statusDetail = "The shielding transaction is waiting to confirm."
	}
	statusStr = ShieldStatusStr[status]
	return
}

func ParseUTXOsToPortalShieldHistory(
	utxos []btcjson.ListUnspentResult, incAddress string,
) ([]PortalShieldHistory, error) {
	histories := []PortalShieldHistory{}
	status := 0
	statusStr := ""
	statusDetail := ""
	for _, u := range utxos {
		status, statusStr, statusDetail = getStatusFromConfirmation(int(u.Confirmations))
		txIDHash, err := chainhash.NewHashFromStr(u.TxID)
		if err != nil {
			log.Printf("Could not new hash from external tx id %v - Error %v\n", u.TxID, err)
			continue
		}
		tx, err := btcClient.GetTransaction(txIDHash)
		if err != nil {
			log.Printf("Could not get external tx id %v - Error %v\n", u.TxID, err)
			continue
		}
		h := PortalShieldHistory{
			Amount:           convertBTCAmtToPBTCAmt(u.Amount),
			ExternalTxID:     u.TxID,
			IncognitoAddress: incAddress,
			Status:           status,
			StatusStr:        statusStr,
			StatusDetail:     statusDetail,
			Time:             tx.Time,
			TxType:           ShieldTxType,
			TxTypeStr:        ShieldTxTypeStr,
		}
		histories = append(histories, h)
	}

	return histories, nil
}
