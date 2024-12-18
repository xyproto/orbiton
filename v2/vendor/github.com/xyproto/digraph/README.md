# Digraph

![Build](https://github.com/xyproto/digraph/workflows/Build/badge.svg) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/digraph)](https://goreportcard.com/report/github.com/xyproto/digraph) [![License](https://img.shields.io/badge/license-BSD-green.svg?style=flat)](https://raw.githubusercontent.com/xyproto/digraph/main/LICENSE)

Lookup ViM-style digraphs.

### Example use

```go
package main

import (
    "fmt"

    "github.com/xyproto/digraph"
)

func main() {
    fmt.Printf("The symbol for My is %c\n", digraph.MustLookup("My"))
}
```

This outputs:

    The symbol for My is µ

[Full list of digraphs](https://github.com/xyproto/digraph/blob/ac809dda476022952cdfc5b12249e8e5fe9f1547/digraphs.txt#L83)

### General info

* License: BSD-3
* Version: 1.3.0
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
