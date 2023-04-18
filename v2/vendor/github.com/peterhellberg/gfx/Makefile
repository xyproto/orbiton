all: test

test:
	go test -v ./...

README.md:
	GO111MODULE=off go get github.com/campoy/embedmd
	embedmd -w README.md

.PHONY:all test README.md
