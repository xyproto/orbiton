.PHONY: clean

PREFIX ?= /usr
MANDIR ?= "$(PREFIX)/share/man/man1"

o:
	GO111MODULES=on go build -mod=vendor -v

o.1.gz: o.1
	gzip -k -v o.1

install: o o.1.gz
	install -Dm755 o "$(DESTDIR)$(PREFIX)/bin/o"
	ln -fs "$(PREFIX)/bin/o" "$(DESTDIR)$(PREFIX)/bin/red"
	install -Dm644 o.1.gz "$(DESTDIR)$(MANDIR)/o.1.gz"

clean:
	rm -f o o.1.gz
