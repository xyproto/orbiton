#!/bin/bash
set -e

cd "$(dirname "$0")"

export VHS_NO_SANDBOX=1

for lang in c cpp rust zig python bash; do
  rm -f /tmp/main.* /tmp/tmp
  vhs simple_$lang.tape
done

# Final cleanup
rm -f /tmp/main.* /tmp/tmp

# Move all gifs to the gif directory
mv -f -v *.gif gif/

echo 'All recordings done.'
