#!/bin/sh
cd o
go build -mod=vendor
./o --version
