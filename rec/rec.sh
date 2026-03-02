#!/bin/bash
set -e

cd "$(dirname "$0")"

export VHS_NO_SANDBOX=1

for lang in c cpp rust zig python; do
  rm -f /tmp/main.* /tmp/tmp
  vhs simple_$lang.tape
done

rm -f /tmp/main.* /tmp/tmp

echo 'All recordings done.'
