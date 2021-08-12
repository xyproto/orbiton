#!/bin/sh
go test -bench=. | tee bench.out; sort -t' ' -nk3 bench.out
