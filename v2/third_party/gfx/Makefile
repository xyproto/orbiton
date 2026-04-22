all: test

test:
	go test -v ./...

README.md:
	go install github.com/campoy/embedmd@latest
	embedmd -w README.md

.PHONY:all test README.md
