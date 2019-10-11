#!/bin/sh
rm -rf o || true
git clone https://github.com/xyproto/o
cd o
go build -mod=vendor
./o --version
