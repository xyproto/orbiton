#!/bin/bash
set -e

cd "$(dirname "$0")"

export VHS_NO_SANDBOX=1

# Record simple C
vhs simple_c.tape
if [ -f /tmp/main.c ]; then
    echo "FAIL: /tmp/main.c was not deleted by the file browser"
    exit 1
fi

# Record simple C++
vhs simple_cpp.tape
if [ -f /tmp/main.cpp ]; then
    echo "FAIL: /tmp/main.cpp was not deleted by the file browser"
    exit 1
fi

# Record simple Rust
vhs simple_rust.tape
if [ -f /tmp/main.rs ]; then
    echo "FAIL: /tmp/main.rs was not deleted by the file browser"
    exit 1
fi

# Record simple Zig
vhs simple_zig.tape
if [ -f /tmp/main.zig ]; then
    echo "FAIL: /tmp/main.zig was not deleted by the file browser"
    exit 1
fi

echo "All recordings done."
