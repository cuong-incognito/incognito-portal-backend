package main

import (
	"context"
	"fmt"
	"github.com/btcsuite/btcd/chaincfg/chainhash"
	"github.com/btcsuite/btcutil"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/kamva/mgm/v3"
	stats "github.com/semihalev/gin-stats"
)

func startGinService() {
	log.Println("initiating api-service...")

	r := gin.Default()
	r.Use(gzip.Gzip(gzip.DefaultCompression))
	r.Use(stats.RequestStats())

	r.GET("/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, stats.Report())
	})
	r.GET("/health", API_HealthCheck)
	r.GET("/checkportalshieldingaddressexisted", API_CheckPortalShieldingAddressExisted)
	r.POST("/addportalshieldingaddress", API_AddPortalShieldingAddress)
	r.GET("/getlistportalshieldingaddress", API_GetListPortalShieldingAddress)
	r.GET("/getestimatedunshieldingfee", API_GetEstimatedUnshieldingFee)
	r.GET("/getshieldhistory", API_GetShieldHistory)
	r.GET("/getshieldhistorybyexternaltxid", API_GetShieldHistoryByExternalTxID)
	err := r.Run("0.0.0.0:" + strconv.Itoa(serviceCfg.APIPort))
	if err != nil {
		panic(err)
	}
}

func API_CheckPortalShieldingAddressExisted(c *gin.Context) {
	incAddress := c.Query("incaddress")
	btcAddress := c.Query("btcaddress")

	// check unique
	isExisted, err := DBCheckPortalAddressExisted(incAddress, btcAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}

	c.JSON(http.StatusOK, API_respond{
		Result: isExisted,
		Error:  nil,
	})
}

func API_AddPortalShieldingAddress(c *gin.Context) {
	var req API_add_portal_shielding_request
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
		return
	}

	// check unique
	isExisted, err := DBCheckPortalAddressExisted(req.IncAddress, req.BTCAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}
	if isExisted {
		msg := "Record has already been inserted"
		c.JSON(http.StatusOK, API_respond{
			Result: nil,
			Error:  &msg,
		})
		return
	}

	err = importBTCAddressToFullNode(req.BTCAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}

	item := NewPortalAddressData(req.IncAddress, req.BTCAddress)
	err = DBSavePortalAddress(*item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}

	c.JSON(http.StatusOK, API_respond{
		Result: true,
		Error:  nil,
	})
}

func API_GetListPortalShieldingAddress(c *gin.Context) {
	fromTimeStamp, err1 := strconv.ParseInt(c.Query("from"), 10, 64)
	toTimeStamp, err2 := strconv.ParseInt(c.Query("to"), 10, 64)
	if err1 != nil || err2 != nil {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(fmt.Errorf("Invalid parameters")))
		return
	}

	list, err := DBGetPortalAddressesByTimestamp(fromTimeStamp, toTimeStamp)
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(err))
		return
	}

	c.JSON(http.StatusOK, API_respond{
		Result: list,
		Error:  nil,
	})
}

func API_GetEstimatedUnshieldingFee(c *gin.Context) {
	vBytePerInput := 192.25
	vBytePerOutput := 43.0
	vByteOverhead := 10.75

	feeRWLock.RLock()
	defer feeRWLock.RUnlock()
	if feePerVByte < 0 {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(fmt.Errorf("Could not get fee from external API")))
		return
	}
	estimatedFee := feePerVByte * (2.0*vBytePerInput + 2.0*vBytePerOutput + vByteOverhead)
	estimatedFee *= 1.15 // overpay

	c.JSON(http.StatusOK, API_respond{
		Result: estimatedFee,
		Error:  nil,
	})
}

func API_GetShieldHistory(c *gin.Context) {
	incAddress := c.Query("incaddress")
	tokenID := c.Query("tokenid")
	if tokenID != BTCTokenID {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(fmt.Errorf(
			"TokenID is not a portal token %v", tokenID)))
		return
	}

	btcAddressStr, err := DBGetBTCAddressByIncAddress(incAddress)
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(fmt.Errorf(
			"Could not get btc address by inc address %v from DB", incAddress)))
		return
	}

	btcAddress, err := btcutil.DecodeAddress(btcAddressStr, BTCChainCfg)
	if err != nil {
		log.Printf(fmt.Sprintf("Could not decode address %v - with err: %v", btcAddressStr, err))
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(
			fmt.Errorf("Could not decode address %v - with err: %v", btcAddressStr, err)))
		return
	}

	utxos, err := btcClient.ListUnspentMinMaxAddresses(BTCMinConf, BTCMaxConf, []btcutil.Address{btcAddress})
	if err != nil {
		log.Printf(fmt.Sprintf("Could not get utxos of address %v - with err: %v", btcAddressStr, err))
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(
			fmt.Errorf("Could not get utxos of address %v - with err: %v", btcAddressStr, err)))
		return
	}

	histories, err := ParseUTXOsToPortalShieldHistory(utxos, incAddress)
	if err != nil {
		log.Printf(fmt.Sprintf("Could not get histories from utxos of address %v - with err: %v", btcAddressStr, err))
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(
			fmt.Errorf("Could not get histories from utxos of address  %v - with err: %v", btcAddressStr, err)))
		return
	}

	c.JSON(http.StatusOK, API_respond{
		Result: histories,
		Error:  nil,
	})
}

func API_GetShieldHistoryByExternalTxID(c *gin.Context) {
	externalTxID := c.Query("externaltxid")
	tokenID := c.Query("tokenid")
	if tokenID != BTCTokenID {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(fmt.Errorf(
			"TokenID is not a portal token %v", tokenID)))
		return
	}

	txIDHash, err := chainhash.NewHashFromStr(externalTxID)
	if err != nil {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(
			fmt.Errorf("Invalid external txID %v - with err: %v", externalTxID, err)))
		return
	}

	res, err := btcClient.GetTransaction(txIDHash)
	if err != nil {
		c.JSON(http.StatusInternalServerError, buildGinErrorRespond(
			fmt.Errorf("Could not get external txID %v - with err: %v", externalTxID, err)))
		return
	}

	status, statusStr, statusDetail := getStatusFromConfirmation(int(res.Confirmations))
	history := PortalShieldHistory{
		ExternalTxID:     externalTxID,
		Status:           status,
		StatusStr:        statusStr,
		StatusDetail:     statusDetail,
	}
	c.JSON(http.StatusOK, API_respond{
		Result: history,
		Error:  nil,
	})

}

func API_HealthCheck(c *gin.Context) {
	//ping pong vs mongo
	status := "healthy"
	mongoStatus := "connected"
	btcNodeStatus := "connected"
	_, cd, _, _ := mgm.DefaultConfigs()
	err := cd.Ping(context.Background(), nil)
	if err != nil {
		status = "unhealthy"
		mongoStatus = "disconnected"
	}
	err = btcClient.Ping()
	if err != nil {
		status = "unhealthy"
		btcNodeStatus = "disconnected"
	}
	c.JSON(http.StatusOK, gin.H{
		"status":      status,
		"mongo":       mongoStatus,
		"btcfullnode": btcNodeStatus,
	})
}

func buildGinErrorRespond(err error) *API_respond {
	errStr := err.Error()
	respond := API_respond{
		Result: nil,
		Error:  &errStr,
	}
	return &respond
}
