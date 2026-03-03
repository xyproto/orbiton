#!/bin/bash
set -e

cd "$(dirname "$0")"

export VHS_NO_SANDBOX=1

rm -rf ./tmp
mkdir -p tmp
for lang in c cpp rust zig python bash odin go haskell; do
  # github.com/xyproto/vhs has --keypress-overlay support
  $HOME/clones/vhs/build/vhs simple_$lang.tape --keypress-overlay || vhs simple_$lang.tape
  mv -v simple_$lang.gif gif/
  rm -rf ./tmp
  mkdir -p tmp
done

echo 'All recordings done.'
