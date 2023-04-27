.PHONY: clean install gui gui-install install-gui og og-install ko ko-install

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

CXX ?= g++
CXXFLAGS ?= -O2 -pipe -fPIC -fno-plt -fstack-protector-strong -Wall -Wshadow -Wpedantic -Wno-parentheses -Wfatal-errors -Wvla -Wignored-qualifiers -pthread
CXXFLAGS += $(shell pkg-config --cflags --libs vte-2.91)

UNAME := $(shell uname)

ifeq ($(UNAME), Darwin)
  CXXFLAGS += -std=c++20
else
  CXXFLAGS += -Wl,--as-needed
endif

o: $(SRCFILES)
	cd v2 && $(GOBUILD) -o ../o

trace: clean $(SRCFILES)
	cd v2 && $(GOBUILD) -tags trace -o ../o

bench:
	cd v2 && go test -bench=. -benchmem

gui: og
ko: og
og: og/og

og/og: og/main.cpp
	$(CXX) "$<" -o "$@" $(CXXFLAGS)

o.1.gz: o.1
	gzip -f -k o.1

install: o o.1.gz
	mkdir -p "$(DESTDIR)$(PREFIX)/bin"
	install -m755 o "$(DESTDIR)$(PREFIX)/bin/o"
	mkdir -p "$(DESTDIR)$(MANDIR)/bin"
	install -m644 o.1.gz "$(DESTDIR)$(MANDIR)/o.1.gz"

gui-install: install-og
install-gui: install-og
ko-install: install-og
install-ko: install-og
og-install: install-og

# using mkdir -p instead of install -D, to make it macOS friendly
install-og: og/og
	mkdir -p "$(DESTDIR)$(PREFIX)/bin"
	install -m755 og/og "$(DESTDIR)$(PREFIX)/bin/og"
	mkdir -p "$(DESTDIR)$(PREFIX)/share/applications"
	install -m644 og/og.desktop "$(DESTDIR)$(PREFIX)/share/applications/og.desktop"
	mkdir -p "$(DESTDIR)$(PREFIX)/share/pixmaps"
	install -m644 img/icon_48x48.png "$(DESTDIR)$(PREFIX)/share/pixmaps/og.png"

install-symlinks: symlinks
symlinks-install: symlinks

symlinks:
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/o" "$(DESTDIR)$(PREFIX)/bin/li"
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/o" "$(DESTDIR)$(PREFIX)/bin/redblack"
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/o" "$(DESTDIR)$(PREFIX)/bin/sw"
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/o" "$(DESTDIR)$(PREFIX)/bin/edi"
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/o" "$(DESTDIR)$(PREFIX)/bin/vs"

symlinks-gui: gui-symlinks
symlinks-gui-install: gui-symlinks
install-symlinks-gui: gui-symlinks

gui-symlinks:
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/lig"
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/redblackg"
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/swg"
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/edig"
	ln -s -f "$(DESTDIR)$(PREFIX)/bin/og" "$(DESTDIR)$(PREFIX)/bin/vsg"

clean:
	-rm -f o v2/o o.1.gz og/og v2/orbiton
