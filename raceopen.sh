#!/bin/bash
filename=${1:-main.go}
rm -f o.log*
export GORACE="halt_on_error=1 log_path=$PWD/o.log"
go build -race && ./o "$filename"
reset
bat o.log*
