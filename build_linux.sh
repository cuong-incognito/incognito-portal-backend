#!/bin/bash

echo "building coin-worker..."
go build -tags=jsoniter -ldflags "-linkmode external -extldflags -static" -o portal_backend