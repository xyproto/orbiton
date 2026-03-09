#!/bin/bash
set -e

cd "$(dirname "$0")"

export VHS_NO_SANDBOX=1

rm -rf ./tmp
mkdir -p tmp
for lang in c go; do
  # github.com/xyproto/vhs has --keypress-overlay support
  $HOME/clones/vhs/build/vhs debug_$lang.tape --keypress-overlay || vhs debug_$lang.tape
  mv -v debug_$lang.gif gif/
  rm -rf ./tmp
  mkdir -p tmp
done

echo 'All debug recordings done.'
