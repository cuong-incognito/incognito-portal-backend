#!/bin/sh
if [$USER == ""]
then 
    JSON='{"apiport":'"$PORT"',"mongo":"mongodb://'"$MONGOHOST"':27017","mongodb":"portal"}'
else
    JSON='{"apiport":'"$PORT"',"mongo":"mongodb://'"$USER"'@'"$MONGOHOST"':27017","mongodb":"portal"}'
fi
echo $JSON 
echo $JSON > cfg.json
./portal_backend