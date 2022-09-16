.PHONY: clean install gui gui-install install-gui ko ko-install o

PREFIX ?= /usr
MANDIR ?= "$(PREFIX)/share/man/man1"
GOBUILD := $(shell test $$(go version | tr ' ' '\n' | head -3 | tail -1 | tr '.' '\n' | head -2 | tail -1) -le 12 2>/dev/null && echo GO111MODULES=on go build -v || echo go build -mod=vendor -v)

SRCFILES := $(wildcard go.* v2/*.go)

CXX ?= g++
CXXFLAGS ?= -O2 -pipe -fPIC -fno-plt -fstack-protector-strong -Wall -Wshadow -Wpedantic -Wno-parentheses -Wfatal-errors -Wvla -Wignored-qualifiers -pthread -Wl,--as-needed
CXXFLAGS += $(shell pkg-config --cflags --libs vte-2.91)

o: $(SRCFILES)
	cd v2 && $(GOBUILD) -o ../o

gui: ko
ko: ko/ko

ko/ko: ko/main.cpp
	$(CXX) "$<" -o "$@" $(CXXFLAGS)

o.1.gz: o.1
	gzip -f -k o.1

install: o o.1.gz
	install -Dm755 o "$(DESTDIR)$(PREFIX)/bin/o"
	install -Dm644 o.1.gz "$(DESTDIR)$(MANDIR)/o.1.gz"

gui-install: install-ko
install-gui: install-ko
ko-install: install-ko

install-ko: ko/ko
	install -Dm755 ko/ko "$(DESTDIR)$(PREFIX)/bin/ko"
	install -Dm644 ko/ko.desktop "$(DESTDIR)$(PREFIX)/share/applications/ko.desktop"
	install -Dm644 img/icon_48x48.png "$(DESTDIR)$(PREFIX)/share/pixmaps/ko.png"

clean:
	-rm -f o v2/o o.1.gz ko/ko
