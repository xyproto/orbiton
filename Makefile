.PHONY: clean install

PREFIX ?= /usr
MANDIR ?= $(PREFIX)/share/man/man1

SRCFILES := $(wildcard *.go)

# Use this command for Go 1.12 and earlier:
#     GO111MODULES=on go build -v
#
# And this command for later versions of go:
#     go build -mod=vendor -v
#
GOBUILD := $(shell test $$(go version | tr ' ' '\n' | head -3 | tail -1 | tr '.' '\n' | tail -1) -le 12 && echo GO111MODULES=on go build -v || echo go build -mod=vendor -v)

o: $(SRCFILES)
	$(GOBUILD)

o.1.gz: o.1
	gzip -f -k o.1

install: o o.1.gz
	install -Dm755 o "$(DESTDIR)$(PREFIX)/bin/o"
	install -Dm644 o.1.gz "$(DESTDIR)$(MANDIR)/o.1.gz"

clean:
	rm -f o o.1.gz
