#!/bin/bash -e
filename=${1:-v2/main.go}
find -name "o.log*" -exec rm -v -- {} \; 2>/dev/null || rm -f -v o.log*
export GORACE="halt_on_error=1 log_path=$PWD/o.log"
cd v2
go build -v -race -o ../o
cd ..
./o -f -- "$filename"
reset
find -name "o.log*" -exec bat -- {} \; 2>/dev/null || bat o.log*
