# lookslikegoasm

This is a package that tries to determine if the given Assembly source code looks like Go/Plan9 style Assembly or Intel/AT&T style Assembly.

The `Consider` function returns `true` if it looks like Go/Plan9 style assembly.

The utility in `cmd` can be installed with:

    go install github.com/xyproto/lookslikegoasm/cmd/lookslikegoasm@latest

Example use:

```go
package main

import (
    "fmt"

    "github.com/xyproto/lookslikegoasm"
)

func main() {
    asmSource := `
    TEXT hello(SB), $0-0
    MOVQ AX, BX
    ADDQ $1, AX
    CALL somefunction
    `

    if lookslikegoasm.Consider(asmSource) {
        fmt.Println("Looks like Go/Plan9 style Assembly")
    } else {
        fmt.Println("Does not look like Go/Plan9 style Assembly")
    }
}
```

General info:

* Version: 1.0.2
* License: BSD-3
