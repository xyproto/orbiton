#!/bin/bash
set -e

cd "$(dirname "$0")"

export VHS_NO_SANDBOX=1

vhs simple_c.tape
rm /tmp/tmp /tmp/main.*

vhs simple_cpp.tape
rm /tmp/tmp /tmp/main.*

vhs simple_rust.tape
rm /tmp/tmp /tmp/main.*

vhs simple_zig.tape
rm /tmp/tmp /tmp/main.*

echo 'All recordings done.'
