#!/bin/sh
if [$USER == ""]
then 
JSON='{
    "apiport":'"$PORT"',
    "mongo":"mongodb://'"$MONGOHOST"':27017",
    "mongodb":"portal",
    "btcfullnode": {
		"address": "'$BTCADDRESS'",
		"user": "'$BTCUSER'",
		"pass": "'$BTCPASS'",
		"https": '$BTCHTTPS'
    }
}'
else
JSON='{
    "apiport":'"$PORT"',
    "mongo":"mongodb://'"$USER"'@'"$MONGOHOST"':27017",
    "mongodb":"portal",
    "btcfullnode": {
		"address": "'$BTCADDRESS'",
		"user": "'$BTCUSER'",
		"pass": "'$BTCPASS'",
		"https": '$BTCHTTPS'
    }
}'
fi
echo $JSON 
echo $JSON > cfg.json
./portal_backend