# Guessica

![Build](https://github.com/xyproto/guessica/workflows/Build/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/guessica)](https://goreportcard.com/report/github.com/xyproto/guessica) [![License](https://img.shields.io/badge/license-MIT-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/guessica/master/LICENSE)

Update a `PKGBUILD` file by guessing the latest version number and finding the latest git tag and hash online.

![logo](img/guessica.svg)

## Installation (development version)

    go get -u github.com/xyproto/guessica/cmd/guessica

## Usage

### Detect the latest version

	guessica PKGBUILD

### Detect the latest version and write the changes back to the PKGBUILD

    guessica -i PKGBUILD

## Note

The `pkgver` and `source` arrays will be guessed by searching the project webpage as defined by the `url`. For for projects on GitHub, `github.com` may also be visited.

## General info

* Version: 1.1.0
* License: MIT
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
