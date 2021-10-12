package main

import (
	"crypto/sha256"
	"fmt"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcd/rpcclient"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcutil"
	"github.com/btcsuite/btcutil/hdkeychain"
	resty "github.com/go-resty/resty/v2"
	"github.com/incognitochain/go-incognito-sdk-v2/wallet"
)

var btcClient *rpcclient.Client

type BlockchainFeeResponse struct {
	Result float64
	Error  error
}

var masterPubKeys = [][]byte{
	[]byte{0x2, 0x39, 0x42, 0x3d, 0xad, 0x93, 0x8f, 0xcb, 0xe5, 0xb5, 0xef, 0x7b, 0x7b, 0x9a, 0xf, 0x28,
		0x4, 0x19, 0x53, 0x66, 0x7f, 0xee, 0x72, 0xe4, 0x81, 0xf9, 0xe6, 0xb, 0x81, 0x41, 0xd7, 0x3a, 0x36},
	[]byte{0x2, 0x8d, 0xc, 0xd7, 0x83, 0x9d, 0x5e, 0xc5, 0x7b, 0x77, 0x1a, 0xf1, 0x2, 0xb8, 0x72, 0xd0,
		0x4f, 0x34, 0xb4, 0xeb, 0x17, 0xac, 0xa1, 0x9f, 0xdf, 0xa, 0x64, 0xbf, 0xd, 0x36, 0x76, 0x66, 0x87},
	[]byte{0x3, 0x78, 0x52, 0x33, 0xe3, 0x8, 0x3a, 0xd8, 0x58, 0x77, 0x76, 0x29, 0xa0, 0x17, 0xb6, 0xdd,
		0x16, 0x43, 0x18, 0x8b, 0xb4, 0xa3, 0xaf, 0x45, 0xf0, 0xb5, 0x91, 0x8c, 0x84, 0xf2, 0x73, 0x56, 0x44},
	[]byte{0x3, 0x61, 0x9d, 0xc9, 0xfb, 0x6d, 0x8, 0x2a, 0x5c, 0x98, 0x45, 0xbc, 0xbf, 0x86, 0xfb, 0x47,
		0x4, 0xbe, 0x67, 0x46, 0xa, 0x59, 0xc4, 0xbc, 0x1d, 0xec, 0xc0, 0xe8, 0xe4, 0x3e, 0x1d, 0x6d, 0x0},
	[]byte{0x2, 0xe4, 0x1d, 0x40, 0xe6, 0xf3, 0x80, 0xad, 0x51, 0xca, 0x17, 0x87, 0xfe, 0xc8, 0x23, 0x8d,
		0xa4, 0xc2, 0x88, 0xfc, 0xfb, 0x6f, 0x2b, 0xcc, 0xd9, 0xa6, 0x1c, 0x2, 0xe5, 0x4a, 0x31, 0x34, 0x39},
	[]byte{0x2, 0xf0, 0xc, 0xe3, 0xec, 0x4, 0xdb, 0x75, 0x59, 0x99, 0x70, 0xc6, 0xfd, 0xc5, 0x2, 0x2f,
		0xad, 0x6b, 0x8d, 0x18, 0x86, 0x71, 0x44, 0xcf, 0xe6, 0x93, 0x92, 0xbb, 0xd1, 0x60, 0xc1, 0x1b, 0x5c},
	[]byte{0x2, 0x65, 0x96, 0x49, 0xab, 0xd4, 0xe5, 0x97, 0x7d, 0x5b, 0x67, 0x4c, 0x6d, 0xa1, 0xf, 0x9,
		0x28, 0xa0, 0x8c, 0x67, 0x8d, 0x7f, 0x50, 0xcc, 0x10, 0xf0, 0xfe, 0xe5, 0x68, 0xa8, 0x57, 0x63, 0xd8},
}
var numSigsRequired = 5
var chainCfg = &chaincfg.MainNetParams

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

func getBitcoinFee() (float64, error) {
	client := resty.New()

	response, err := client.R().
		Get(serviceCfg.BlockchainFeeHost)

	if err != nil {
		return 0, err
	}
	if response.StatusCode() != 200 {
		return 0, fmt.Errorf("Response status code: %v", response.StatusCode())
	}
	var responseBody BlockchainFeeResponse
	err = json.Unmarshal(response.Body(), &responseBody)
	if err != nil {
		return 0, fmt.Errorf("Could not parse response: %v", response.Body())
	}
	return responseBody.Result, nil
}
