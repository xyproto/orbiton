#!/bin/bash
filename=${1:-main.go}
export GORACE="halt_on_error=1 log_path=$PWD/o.log"
go build -race && ./o "$filename"
clear
reset
ls -al o.log*
