.PHONY: clean install

PROJECT ?= megafile

GOFLAGS ?= -mod=vendor -trimpath -v -ldflags "-s -w" -buildvcs=false

GOBUILD := go build

GOEXPERIMENT := greenteagc

SRCFILES := $(wildcard go.* *.go cmd/megafile/*.go)

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

megafile: $(SRCFILES)
	cd cmd/megafile && $(GOBUILD) $(GOFLAGS) $(BUILDFLAGS) -o ../../megafile

install: megafile
	mkdir -p "$(DESTDIR)$(PREFIX)/bin"
	install -m755 megafile "$(DESTDIR)$(PREFIX)/bin/megafile"
	mkdir -p "$(DESTDIR)$(MANDIR)"
	install -m644 megafile.1.gz "$(DESTDIR)$(MANDIR)/megafile.1.gz"

license:
	mkdir -p "$(DESTDIR)$(PREFIX)/share/licenses/$(PROJECT)"
	install -m644 LICENSE "$(DESTDIR)$(PREFIX)/share/licenses/$(PROJECT)/LICENSE"

clean:
	-rm -f megafile
