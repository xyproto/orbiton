.PHONY: clean install gui gui-install install-gui og og-install ko ko-install

PREFIX ?= /usr
MANDIR ?= "$(PREFIX)/share/man/man1"
GOBUILD := $(shell test $$(go version | tr ' ' '\n' | head -3 | tail -1 | tr '.' '\n' | head -2 | tail -1) -le 12 2>/dev/null && echo GO111MODULES=on go build -v || echo go build -mod=vendor -v)

SRCFILES := $(wildcard go.* v2/*.go v2/go.*)

CXX ?= g++
CXXFLAGS ?= -O2 -pipe -fPIC -fno-plt -fstack-protector-strong -Wall -Wshadow -Wpedantic -Wno-parentheses -Wfatal-errors -Wvla -Wignored-qualifiers -pthread -Wl,--as-needed
CXXFLAGS += $(shell pkg-config --cflags --libs vte-2.91)

o: $(SRCFILES)
	cd v2 && $(GOBUILD) -o ../o

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
	install -Dm644 img/icon_48x48.png "$(DESTDIR)$(PREFIX)/share/pixmaps/og.png"

clean:
	-rm -f o v2/o o.1.gz og/og
