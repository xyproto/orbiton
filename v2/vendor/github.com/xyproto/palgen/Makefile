.PHONY: all clean install

DESTDIR ?=
PREFIX ?= /usr
UNAME_R ?= $(shell uname -r)

ifneq (,$(findstring arch,$(UNAME_R)))
# Arch Linux
LDFLAGS ?= -Wl,-O2,--sort-common,--as-needed,-z,relro,-z,now
BUILDFLAGS ?= -mod=vendor -buildmode=pie -trimpath -ldflags "-s -w -extldflags $(LDFLAGS)"
else
# Default settings
BUILDFLAGS ?= -mod=vendor -trimpath
endif

all:
	go build ${BUILDFLAGS}
	(cd cmd/png2act; go build ${BUILDFLAGS})
	(cd cmd/png2gpl; go build ${BUILDFLAGS})
	(cd cmd/png2png; go build ${BUILDFLAGS})
	(cd cmd/png256; go build ${BUILDFLAGS})

fmt:
	go fmt
	(cd cmd/png2act; go fmt)
	(cd cmd/png2gpl; go fmt)
	(cd cmd/png2png; go fmt)
	(cd cmd/png256; go fmt)

install:
	install -Dm755 -t "$(DESTDIR)$(PREFIX)/bin" cmd/png2act/png2act
	install -Dm755 -t "$(DESTDIR)$(PREFIX)/bin" cmd/png2gpl/png2gpl
	install -Dm755 -t "$(DESTDIR)$(PREFIX)/bin" cmd/png2png/png2png
	install -Dm755 -t "$(DESTDIR)$(PREFIX)/bin" cmd/png256/png256

clean:
	(cd cmd/png2act; go clean)
	(cd cmd/png2gpl; go clean)
	(cd cmd/png2png; go clean)
	(cd cmd/png256; go clean)
	go clean
