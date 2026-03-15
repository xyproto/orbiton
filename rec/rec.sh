#!/bin/bash
set -e

cd "$(dirname "$0")"

export VHS_NO_SANDBOX=1

VHS=$(which vhs)
if [ -x "$HOME/clones/vhs/build/vhs" ]; then
  VHS="$HOME/clones/vhs/build/vhs"
elif [ -x "$HOME/clones/vhs/vhs" ]; then
  VHS="$HOME/clones/vhs/vhs"
fi
if [ ! -x $VHS ]; then
  echo 'Could not find vhs'
  exit 1
fi

rm -rf ./tmp
mkdir -p tmp
for lang in c cpp rust zig python bash odin go haskell; do
  # github.com/xyproto/vhs has --keypress-overlay support
  "$VHS" debug_$lang.tape --keypress-overlay || "$VHS" debug_$lang.tape
  mv -v simple_$lang.gif gif/
  rm -rf ./tmp
  mkdir -p tmp
done

echo 'All recordings done.'
