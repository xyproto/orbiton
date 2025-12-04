.PHONY: clean install

PROJECT ?= megacli

GOFLAGS ?= -mod=vendor -trimpath -v -ldflags "-s -w" -buildvcs=false

GOBUILD := go build

GOEXPERIMENT := greenteagc

SRCFILES := $(wildcard go.* *.go)

# macOS and FreeBSD detection
UNAME_S := $(shell uname -s)
ifeq ($(UNAME_S),Darwin)
  PREFIX ?= /usr/local
  MAKE ?= make
else ifeq ($(UNAME_S),FreeBSD)
  PREFIX ?= /usr/local
  MAKE ?= gmake
else
  PREFIX ?= /usr
  MAKE ?= make
endif

MANDIR ?= $(PREFIX)/share/man/man1

UNAME_R ?= $(shell uname -r)
ifneq (,$(findstring arch,$(UNAME_R)))
# Arch Linux
LDFLAGS ?= -Wl,-O2,--as-needed,-z,relro,-z,now
GOFLAGS += -buildmode=pie
BUILDFLAGS ?= -ldflags "-s -w -linkmode=external -extldflags $(LDFLAGS)"
endif

megacli: $(SRCFILES)
	cd cmd/megacli && $(GOBUILD) $(GOFLAGS) $(BUILDFLAGS) -o ../../megacli

install: megacli
	mkdir -p "$(DESTDIR)$(PREFIX)/bin"
	install -m755 megacli "$(DESTDIR)$(PREFIX)/bin/megacli"
	mkdir -p "$(DESTDIR)$(MANDIR)"
	install -m644 megacli.1.gz "$(DESTDIR)$(MANDIR)/megacli.1.gz"

license:
	mkdir -p "$(DESTDIR)$(PREFIX)/share/licenses/$(PROJECT)"
	install -m644 LICENSE "$(DESTDIR)$(PREFIX)/share/licenses/$(PROJECT)/LICENSE"

clean:
	-rm -f megacli
