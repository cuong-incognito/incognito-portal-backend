#!/bin/bash

echo "building portal_backend..."
go build -tags=jsoniter -v -o portal_backend