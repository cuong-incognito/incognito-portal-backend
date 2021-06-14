package main

import (
	"time"

	"github.com/kamva/mgm/v3"
)

type PortalAddressData struct {
	mgm.DefaultModel `bson:",inline"`
	IncAddress       string `json:"incaddress" bson:"incaddress"`
	BTCAddress       string `json:"btcaddress" bson:"btcaddress"`
	TimeStamp        int64  `json:"timestamp" bson:"timestamp"`
}

func NewPortalAddressData(incAddress, btcAddress string) *PortalAddressData {
	timestamp := time.Now().UnixNano()
	return &PortalAddressData{
		IncAddress: incAddress, BTCAddress: btcAddress, TimeStamp: timestamp,
	}
}

func (model *PortalAddressData) Creating() error {
	// Call the DefaultModel Creating hook
	if err := model.DefaultModel.Creating(); err != nil {
		return err
	}

	return nil
}

func (model *PortalAddressData) Saving() error {
	// Call the DefaultModel Creating hook
	if err := model.DefaultModel.Saving(); err != nil {
		return err
	}

	return nil
}
