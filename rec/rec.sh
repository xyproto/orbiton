#!/usr/bin/env bash

cd "$(dirname "$0")"

export VHS_NO_SANDBOX=1

VHS=$(which vhs 2>/dev/null || true)
if [ -x "$HOME/clones/vhs/build/vhs" ]; then
  VHS="$HOME/clones/vhs/build/vhs"
elif [ -x "$HOME/clones/vhs/vhs" ]; then
  VHS="$HOME/clones/vhs/vhs"
fi
if [ ! -x "$VHS" ]; then
  echo 'Could not find vhs'
  exit 1
fi

rm -rf ./tmp
mkdir -p tmp
for lang in gleam c cpp rust zig python bash odin go haskell; do
  # github.com/xyproto/vhs has --keypress-overlay support
  "$VHS" simple_$lang.tape --keypress-overlay || "$VHS" simple_$lang.tape
  mv -v simple_$lang.gif gif/
  rm -rf ./tmp
  mkdir -p tmp
done

echo 'All recordings done.'
