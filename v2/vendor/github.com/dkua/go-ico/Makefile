.PHONY: all

all:
	@echo "*******************************"
	@echo "** Pressly go.image/ico build tool **"
	@echo "*******************************"
	@echo "make <cmd>"
	@echo ""
	@echo "commands:"
	@echo "  test        - standard go test"
	@echo "  convey      - TDD runner"
	@echo "  build       - build the dist binary"
	@echo "  clean       - clean the dist build"
	@echo ""
	@echo "  tools       - go get's a bunch of tools for dev"
	@echo "  deps        - pull and setup dependencies"
	@echo "  update_deps - update deps lock file"

test:
	@go test -v ./...

convey:
	goconvey

build_pkgs:
	go build ./... 

tools:
	go get github.com/robfig/glock
	go get github.com/pkieltyka/fresh
	go get github.com/smartystreets/goconvey

deps:
	@glock sync -n github.com/pressly/go.image < Glockfile

update_deps:
	@glock save -n github.com/pressly/go.image > Glockfile
