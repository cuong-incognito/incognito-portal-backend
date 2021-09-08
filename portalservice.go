package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"net/http"
	"sync"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	"github.com/incognitochain/go-incognito-sdk-v2/wallet"
)

var btcClient *rpcclient.Client
var feeRWLock sync.RWMutex
var feePerVByte float64 // satoshi / byte

type BlockCypherFeeResponse struct {
	HighFee   uint `json:"high_fee_per_kb"`
	MediumFee uint `json:"medium_fee_per_kb"`
	LowFee    uint `json:"low_fee_per_kb"`
}

var masterPubKeys = [][]byte{
	{3, 178, 211, 22, 125, 148, 156, 37, 3, 230, 156, 159, 41, 120, 125, 156, 8, 141, 57, 23, 141, 180, 117, 64, 53, 245, 174, 106, 240, 23, 18, 17, 0},
	{3, 152, 122, 135, 209, 153, 19, 189, 227, 239, 240, 85, 121, 2, 180, 144, 87, 237, 28, 156, 139, 50, 249, 2, 187, 187, 133, 113, 58, 153, 31, 220, 65},
	{3, 115, 35, 94, 177, 200, 241, 132, 231, 89, 23, 108, 227, 135, 55, 183, 145, 25, 71, 27, 186, 99, 86, 188, 171, 141, 204, 20, 75, 66, 153, 134, 1},
	{3, 41, 231, 89, 49, 137, 202, 122, 246, 1, 182, 53, 103, 61, 177, 83, 212, 25, 215, 6, 25, 3, 42, 50, 148, 87, 118, 178, 179, 128, 101, 225, 93},
}
var numSigsRequired = 3
var chainCfg = &chaincfg.TestNet3Params

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
				response, err := http.Get("https://api.blockcypher.com/v1/btc/main")
				feeRWLock.Lock()
				defer func() {
					feeRWLock.Unlock()
					time.Sleep(3 * time.Minute)
				}()
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

func generateOTMultisigAddress(masterPubKeys [][]byte, numSigsRequired int, chainCodeSeed string, chainParam *chaincfg.Params) ([]byte, string, error) {
	if len(masterPubKeys) < numSigsRequired || numSigsRequired < 0 {
		return []byte{}, "", fmt.Errorf("Invalid signature requirement")
	}

	pubKeys := [][]byte{}
	// this Incognito address is marked for the address that received change UTXOs
	if chainCodeSeed == "" {
		pubKeys = masterPubKeys[:]
	} else {
		chainCode := chainhash.HashB([]byte(chainCodeSeed))
		for idx, masterPubKey := range masterPubKeys {
			// generate BTC child public key for this Incognito address
			extendedBTCPublicKey := hdkeychain.NewExtendedKey(chainParam.HDPublicKeyID[:], masterPubKey, chainCode, []byte{}, 0, 0, false)
			extendedBTCChildPubKey, _ := extendedBTCPublicKey.Child(0)
			childPubKey, err := extendedBTCChildPubKey.ECPubKey()
			if err != nil {
				return []byte{}, "", fmt.Errorf("Master BTC Public Key (#%v) %v is invalid - Error %v", idx, masterPubKey, err)
			}
			pubKeys = append(pubKeys, childPubKey.SerializeCompressed())
		}
	}

	// create redeem script for m of n multi-sig
	builder := txscript.NewScriptBuilder()
	// add the minimum number of needed signatures
	builder.AddOp(byte(txscript.OP_1 - 1 + numSigsRequired))
	// add the public key to redeem script
	for _, pubKey := range pubKeys {
		builder.AddData(pubKey)
	}
	// add the total number of public keys in the multi-sig script
	builder.AddOp(byte(txscript.OP_1 - 1 + len(pubKeys)))
	// add the check-multi-sig op-code
	builder.AddOp(txscript.OP_CHECKMULTISIG)

	redeemScript, err := builder.Script()
	if err != nil {
		return []byte{}, "", fmt.Errorf("Could not build script - Error %v", err)
	}

	// generate P2WSH address
	scriptHash := sha256.Sum256(redeemScript)
	addr, err := btcutil.NewAddressWitnessScriptHash(scriptHash[:], chainParam)
	if err != nil {
		return []byte{}, "", fmt.Errorf("Could not generate address from script - Error %v", err)
	}
	addrStr := addr.EncodeAddress()

	return redeemScript, addrStr, nil
}

func generateBTCAddress(incAddress string) (string, error) {
	_, address, err := generateOTMultisigAddress(masterPubKeys, numSigsRequired, incAddress, chainCfg)
	if err != nil {
		return "", err
	}
	return address, nil
}

func isValidPortalAddressPair(incAddress string, btcAddress string) error {
	_, err := wallet.Base58CheckDeserialize(incAddress)
	if err != nil {
		return err
	}

	generatedBTCAddress, err := generateBTCAddress(incAddress)
	if err != nil {
		return err
	}
	if generatedBTCAddress != btcAddress {
		return fmt.Errorf("Invalid BTC address")
	}

	return nil
}
