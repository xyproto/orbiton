# fullname

This is both a Go package and a utility for trying to find the full name of the current user, on Linux.

## Installing the utility

Requires Go 1.16 or later:

```bash
go install github.com/xyproto/fullname/cmd/fullname@latest
```

## Example use

```go
package main

import (
    "fmt"

    "github.com/xyproto/fullname"
)

func main() {
    fmt.Println(fullname.Get())
}
```

## General info

* Version: 1.1.0
* License: MIT
