# guessica

Update a PKGBUILD file by guessing the latest version number and finding the latest git tag and hash online.

## Installation (development version)

    go get -u github.com/xyproto/guessica/cmd/guessica

## Usage

	guessica PKGBUILD

## Notes

The `pkgver` and `source` arrays will be guessed by searching the project webpage as defined by the `url`. For for projects on GitHub, `github.com` may also be visited.

Updating a `PKGBUILD` may or may not work. `guessica` is doing its best, by guessing. Take a backup of your `PKGBUILD` first, if you need to.

## General info

* Version: 0.0.1
* License: MIT
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
