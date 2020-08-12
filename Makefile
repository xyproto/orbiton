.PHONY: clean

PREFIX ?= /usr
MANDIR ?= "$(PREFIX)/share/man/man1"
GOBUILD := $(shell test $$(go version | tr ' ' '\n' | head -3 | tail -1 | tr '.' '\n' | tail -1) -le 12 && echo GO111MODULES=on go build -v || echo go build -mod=vendor -v)

o:
	$(GOBUILD)

o.1.gz: o.1
	gzip -f -k -v o.1

install: o o.1.gz
	install -Dm755 o "$(DESTDIR)$(PREFIX)/bin/o"
	ln -fs "$(PREFIX)/bin/o" "$(DESTDIR)$(PREFIX)/bin/red"
	install -Dm644 o.1.gz "$(DESTDIR)$(MANDIR)/o.1.gz"

clean:
	rm -f o o.1.gz
