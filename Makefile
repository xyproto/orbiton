.PHONY: clean trace bench install install-gui symlinks vg-symlink gui-symlinks feedgame-symlink feedgame-gui-symlink license

PROJECT ?= orbiton
PROGRAM ?= o

MANDIR ?= "$(PREFIX)/share/man/man1"

GOBUILD := $(shell test $$(go version | tr ' ' '\n' | head -3 | tail -1 | tr '.' '\n' | head -2 | tail -1) -le 12 2>/dev/null && echo GO111MODULES=on go build -v || echo go build -mod=vendor -v)
SRCFILES := $(wildcard go.* v2/*.go v2/go.*)

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

UNAME_R ?= $(shell uname -r)
ifneq (,$(findstring arch,$(UNAME_R)))
# Arch Linux
LDFLAGS ?= -Wl,-O2,--sort-common,--as-needed,-z,relro,-z,now
BUILDFLAGS ?= -mod=vendor -buildmode=pie -trimpath -ldflags "-s -w -linkmode=external -extldflags $(LDFLAGS)"
else
# Default settings
BUILDFLAGS ?= -mod=vendor -trimpath
endif

CXX ?= g++
CXXFLAGS ?= -O2 -pipe -fPIC -fno-plt -fstack-protector-strong -Wall -Wshadow -Wpedantic -Wno-parentheses -Wfatal-errors -Wvla -Wignored-qualifiers -pthread
CXXFLAGS += $(shell pkg-config --cflags --libs vte-2.91)

ifeq ($(UNAME_S),Darwin)
  CXXFLAGS += -std=c++20
else
  CXXFLAGS += -Wl,--as-needed
endif

$(PROGRAM): $(SRCFILES)
	cd v2 && $(GOBUILD) $(BUILDFLAGS) -o ../$(PROGRAM)

trace: clean $(SRCFILES)
	cd v2 && $(GOBUILD) $(BUILDFLAGS) -tags trace -o ../$(PROGRAM)

bench:
	cd v2 && go test -bench=. -benchmem

og/og: og/main.cpp
	$(CXX) "$<" -o "$@" $(CXXFLAGS) $(LDFLAGS)

o.1.gz: o.1
	gzip -f -k o.1


install: $(PROGRAM) o.1.gz
	mkdir -p "$(DESTDIR)$(PREFIX)/bin"
	install -m755 $(PROGRAM) "$(DESTDIR)$(PREFIX)/bin/$(PROGRAM)"
	mkdir -p "$(DESTDIR)$(MANDIR)/bin"
	install -m644 o.1.gz "$(DESTDIR)$(MANDIR)/o.1.gz"

install-gui: og/og vg-symlink
	mkdir -p "$(DESTDIR)$(PREFIX)/bin"
	install -m755 og/og "$(DESTDIR)$(PREFIX)/bin/og"
	mkdir -p "$(DESTDIR)$(PREFIX)/share/pixmaps"
	install -m644 img/og.png "$(DESTDIR)$(PREFIX)/share/pixmaps/og.png"
	install -m644 img/vg.png "$(DESTDIR)$(PREFIX)/share/pixmaps/vg.png"
	mkdir -p "$(DESTDIR)$(PREFIX)/share/applications"
	install -m644 og/og.desktop "$(DESTDIR)$(PREFIX)/share/applications/og.desktop"
	install -m644 og/vg.desktop "$(DESTDIR)$(PREFIX)/share/applications/vg.desktop"

symlinks:
	ln -s -f "$(PREFIX)/bin/$(PROGRAM)" "$(DESTDIR)$(PREFIX)/bin/li"
	ln -s -f "$(PREFIX)/bin/$(PROGRAM)" "$(DESTDIR)$(PREFIX)/bin/redblack"
	ln -s -f "$(PREFIX)/bin/$(PROGRAM)" "$(DESTDIR)$(PREFIX)/bin/sw"
	ln -s -f "$(PREFIX)/bin/$(PROGRAM)" "$(DESTDIR)$(PREFIX)/bin/edi"
	ln -s -f "$(PREFIX)/bin/$(PROGRAM)" "$(DESTDIR)$(PREFIX)/bin/vs"

vg-symlink:
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/vg"

gui-symlinks: vg-symlink
	ln -s -f "$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/lig"
	ln -s -f "$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/redblackg"
	ln -s -f "$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/swg"
	ln -s -f "$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/edg"

feedgame-symlink:
	ln -s -f "$(PREFIX)/bin/$(PROGRAM)" "$(DESTDIR)$(PREFIX)/bin/feedgame"

feedgame-gui-symlink:
	ln -s -f "$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/feedgameg"

license:
	install -d "$(DESTDIR)$(PREFIX)/share/licenses/$(PROJECT)"
	install -m644 LICENSE "$(DESTDIR)$(PREFIX)/share/licenses/$(PROJECT)/LICENSE"

clean:
	-rm -f o v2/o o.1.gz og/og v2/orbiton
