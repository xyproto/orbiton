# distrodetector

[![GoDoc](https://godoc.org/github.com/xyproto/distrodetector?status.svg)](http://godoc.org/github.com/xyproto/distrodetector) [![License](http://img.shields.io/badge/license-BSD-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/distrodetector/master/LICENSE) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/distrodetector)](https://goreportcard.com/report/github.com/xyproto/distrodetector)

Detects which Linux distro or BSD a system is running.

Aims to detect:

* The 100 most popular Linux distros and BSDs, according to distrowatch
* macOS

The `distro` utility and the `distrodetector` package has no external dependencies.

Pull requests for additional systems are welcome!

## Installation of the distro utility

The `distro` utility can be used as a drop-in replacement for the `distro` command that comes with `python-distro`.

Installation of the development version of the `distro` utility:

    go get -u github.com/xyproto/distrodetector/cmd/distro

Example use:

    distro

## Use of the Go package

```go
package main

import (
    "fmt"
    "github.com/xyproto/distrodetector"
)

func main() {
    distro := distrodetector.New()
    fmt.Println(distro.Name())
}
```
## Example output

The parts can be retrieved separately with `.Platform()`, `.Name()`, `.Codename()` and `.Version()`. A combined string can be returned with the `.String()` function:

    Linux (Arch Linux)
    Linux (Ubuntu Bionic 18.04)
    macOS (High Sierra 10.13.3)
    Linux (Void Linux)

## Testing

* More testing is always needed when detecting Linux distros and BSDs.
* Please test the distro detection on your distro/BSD and submit an issue or pull request if it should fail.

## General Info

* License: BSD-3
* Version: 1.3.1
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
