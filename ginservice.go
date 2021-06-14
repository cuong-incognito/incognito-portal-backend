package main

import (
	"context"
	"fmt"
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
	r.GET("/getlistportalshieldingaddress", API_GetListPortalShieldingAddress)
	r.POST("/addportalshieldingaddress", API_AddPortalShieldingAddress)
	r.Run("0.0.0.0:" + strconv.Itoa(serviceCfg.APIPort))
}

func API_AddPortalShieldingAddress(c *gin.Context) {
	var req API_add_portal_shielding_request
	err := c.ShouldBindJSON(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
	}

	// check unique
	isExisted, err := DBCheckPortalAddressExisted(req.IncAddress, req.BTCAddress)
	if err != nil {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
	}
	if isExisted {
		msg := "Record has already been inserted"
		c.JSON(http.StatusOK, API_respond{
			Result: false,
			Error:  &msg,
		})
	}

	item := NewPortalAddressData(req.IncAddress, req.BTCAddress)
	err = DBSavePortalAddress(*item)
	if err != nil {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
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
	}

	list, err := DBGetPortalAddressesByTimestamp(fromTimeStamp, toTimeStamp)
	if err != nil {
		c.JSON(http.StatusBadRequest, buildGinErrorRespond(err))
	}

	c.JSON(http.StatusOK, API_respond{
		Result: list,
		Error:  nil,
	})
}

func API_HealthCheck(c *gin.Context) {
	//ping pong vs mongo
	status := "healthy"
	mongoStatus := "connected"
	_, cd, _, _ := mgm.DefaultConfigs()
	err := cd.Ping(context.Background(), nil)
	if err != nil {
		status = "unhealthy"
		mongoStatus = "disconnected"
	}
	c.JSON(http.StatusOK, gin.H{
		"status": status,
		"mongo":  mongoStatus,
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
