gdb
===

Package `gdb` provides a convenient way to interact with the GDB/MI
interface. The methods offered by this module are very low level, the main goals
are:

- avoid the tedious parsing of the MI2 line-based text interface;

- bypass a [known bug][mi2-bug] which prevents to distinguish the target
program's output from MI2 records.

Web interface
-------------

This package comes with an additional [HTTP/WebSocket interface](web/) which
aims to provide a straightforward way to start developing web-based GDB front
ends.

A dummy example can be found in the [example folder](web/example).

Example
-------

```go
package main

import (
	"fmt"
	"github.com/cyrus-and/gdb"
	"io"
	"os"
)

func main() {
	// start a new instance and pipe the target output to stdout
	gdb, _ := gdb.New(nil)
	go io.Copy(os.Stdout, gdb)

	// evaluate an expression
	gdb.Send("var-create", "x", "@", "40 + 2")
	fmt.Println(gdb.Send("var-evaluate-expression", "x"))

	// load and run a program
	gdb.Send("file-exec-file", "wc")
	gdb.Send("exec-arguments", "-w")
	gdb.Write([]byte("This sentence has five words.\n\x04")) // EOT
	gdb.Send("exec-run")

	gdb.Exit()
}
```

Installation
------------

    go get github.com/cyrus-and/gdb

Documentation
-------------

[GoDoc][godoc]

Data representation
-------------------

The objects returned as a result of the commands or as asynchronous
notifications are generic Go maps suitable to be converted to JSON format with
`json.Marshal()`. The fields present in such objects are blindly added
according to the records returned from GDB (see the
[command syntax][mi2-syntax]): tuples are `map[string]interface{}` and lists are
`[]interface{}`.

Yet, some additional fields are added:

- the record class, where present, is represented by the `"class"` field;

- the record type is represented using the `"type"` field as follows:
    - `+`: `"status"`
    - `=`: `"notify"`
    - `~`: `"console"`
    - `@`: `"target"`
    - `&`: `"log"`

- the optional result list is stored into a tuple under the `"payload"` field.

For example, the notification:

    =thread-group-exited,id="i1",exit-code="0"

becomes the Go map:

```go
map[type:notify class:thread-group-exited payload:map[id:i1 exit-code:0]]
```

which can be converted to JSON with `json.Marshal()` obtaining:

```json
{
    "class": "thread-group-exited",
    "payload": {
        "exit-code": "0",
        "id": "i1"
    },
    "type": "notify"
}
```

Mac OS X
--------

### Setting up GDB on Darwin

To use this module is mandatory to have a working version of GDB installed, Mac
OS X users may obtain a copy using [Homebrew][homebrew] for example, then they
may need to give GDB permission to control other processes as described
[here][gdb-on-mac].

### Issues

The Mac OS X support, though, is partial and buggy due to the following issues.

#### Pseudoterminals

I/O operations on the target program happens through a pseudoterminal obtained
using the [pty][pty] package which basically uses the `/dev/ptmx` on *nix
systems to request new terminal instances.

There are some unclear behaviors on Mac OS X. Calling `gdb.Write()` when the
target program is not running is a no-op, on Linux instead writes are somewhat
buffered and delivered later. Likewise, `gdb.Read()` may returns EOF even though
there is actually data to read, a solution may be keep trying.

#### Interrupt

Sending a `SIGINT` signal to GDB has no effect on Mac OS X, on Linux instead
this is equivalent to typing `^C`, so `gdb.Interrupt()` will not work.

Resources
---------

- [The `GDB/MI` Interface][gdb-mi]

[mi2-bug]: https://sourceware.org/bugzilla/show_bug.cgi?id=8759
[mi2-syntax]: https://sourceware.org/gdb/onlinedocs/gdb/GDB_002fMI-Output-Syntax.html
[godoc]: https://godoc.org/github.com/cyrus-and/gdb
[homebrew]: http://brew.sh/
[gdb-on-mac]: http://sourceware.org/gdb/wiki/BuildingOnDarwin
[pty]: https://github.com/kr/pty
[gdb-mi]: https://sourceware.org/gdb/onlinedocs/gdb/GDB_002fMI.html
