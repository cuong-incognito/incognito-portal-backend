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
	[]byte{0x2, 0x30, 0x34, 0xcb, 0x1a, 0x50, 0xf6, 0x7f, 0x5e, 0xb2, 0x53, 0x9e, 0x68, 0x3b, 0xd4,
		0x80, 0x73, 0x71, 0x2a, 0xdf, 0xf3, 0x25, 0x94, 0x34, 0x72, 0x6d, 0x62, 0x80, 0x83, 0xd2, 0x6f, 0x4c, 0xdd},
	[]byte{0x2, 0x74, 0x61, 0x32, 0x93, 0xe7, 0x93, 0x85, 0x94, 0xd2, 0x58, 0xfb, 0xcf, 0xc5, 0x33,
		0x78, 0xdc, 0x82, 0xcd, 0x64, 0xd1, 0xc0, 0x33, 0x1, 0x71, 0x2f, 0x90, 0x85, 0x72, 0xb9, 0x17, 0xab, 0xc7},
	[]byte{0x3, 0x67, 0x7a, 0x81, 0xfc, 0x9c, 0x4c, 0x9c, 0x6, 0x28, 0xd2, 0xf6, 0xd0, 0x1e, 0x27,
		0x15, 0xbb, 0x54, 0x11, 0x75, 0xe9, 0x62, 0xae, 0x78, 0x8f, 0xff, 0x26, 0x75, 0x1e, 0xb5, 0x24, 0xe0, 0xeb},
	[]byte{0x3, 0x2, 0xdb, 0xd4, 0xd4, 0x6b, 0x4e, 0xef, 0xe9, 0xa6, 0xe8, 0x64, 0xce, 0xeb, 0xb5,
		0x11, 0x25, 0x71, 0x28, 0x8a, 0xc4, 0xce, 0xca, 0xf4, 0x10, 0xd4, 0x16, 0x5f, 0x4c, 0x4c, 0xeb, 0x27, 0xe3},
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
