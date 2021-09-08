package main

import (
	"context"
	"log"
	"time"

	"github.com/kamva/mgm/v3"
	"github.com/kamva/mgm/v3/operator"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/x/bsonx"
)

func connectDB() error {
	err := mgm.SetDefaultConfig(nil, serviceCfg.MongoDB, options.Client().ApplyURI(serviceCfg.MongoAddress))
	if err != nil {
		return err
	}
	_, cd, _, _ := mgm.DefaultConfigs()
	err = cd.Ping(context.Background(), nil)
	if err != nil {
		return err
	}
	log.Println("Database Connected!")
	return nil
}

func DBCreatePortalAddressIndex() error {
	startTime := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(5)*DB_OPERATION_TIMEOUT)

	coinMdl := []mongo.IndexModel{
		{
			Keys:    bsonx.Doc{{Key: "incaddress", Value: bsonx.Int32(1)}, {Key: "btcaddress", Value: bsonx.Int32(1)}},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: bsonx.Doc{{Key: "timestamp", Value: bsonx.Int32(1)}},
		},
	}
	_, err := mgm.Coll(&PortalAddressData{}).Indexes().CreateMany(ctx, coinMdl)
	if err != nil {
		log.Printf("failed to index portal addresses in %v", time.Since(startTime))
		return err
	}

	log.Printf("success index portal addresses in %v", time.Since(startTime))
	return nil
}

func DBCheckPortalAddressExisted(incAddress, btcAddress string) (bool, error) {
	startTime := time.Now()

	filter := bson.M{"incaddress": bson.M{operator.Eq: incAddress}, "btcaddress": bson.M{operator.Eq: btcAddress}}
	var result PortalAddressData
	err := mgm.Coll(&PortalAddressData{}).First(filter, &result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			log.Printf("check portal address not existed in %v", time.Since(startTime))
			return false, nil
		}
		return false, err
	}
	log.Printf("check portal address existed in %v", time.Since(startTime))
	return true, nil
}

func DBSavePortalAddress(item PortalAddressData) error {
	startTime := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), time.Duration(5)*DB_OPERATION_TIMEOUT)

	_, err := mgm.Coll(&PortalAddressData{}).InsertOne(ctx, item)
	if err != nil {
		log.Printf("failed to insert portal address %v in %v", item, time.Since(startTime))
		return err
	}

	log.Printf("inserted portal address %v in %v", item, time.Since(startTime))
	return nil
}

func DBGetPortalAddressesByTimestamp(fromTimeStamp int64, toTimeStamp int64) ([]PortalAddressData, error) {
	startTime := time.Now()
	list := []PortalAddressData{}
	filter := bson.M{"timestamp": bson.M{operator.Gte: fromTimeStamp, operator.Lt: toTimeStamp}}

	err := mgm.Coll(&PortalAddressData{}).SimpleFind(&list, filter)
	if err != nil {
		return nil, err
	}
	log.Printf("found %v addresses in %v", len(list), time.Since(startTime))

	return list, nil
}

func DBGetBTCAddressByIncAddress(incAddress string) (string, error) {
	startTime := time.Now()

	filter := bson.M{"incaddress": bson.M{operator.Eq: incAddress}}
	var result PortalAddressData
	err := mgm.Coll(&PortalAddressData{}).First(filter, &result)
	if err != nil {
		return "", err
	}
	log.Printf("get btc address by inc address in %v", time.Since(startTime))
	return result.BTCAddress, nil
}
