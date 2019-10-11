#!/bin/sh
#
# Remember to search for packages case insensitively, with "sudo pkg search -i asdf"
#

# Update package repo
pkg-static update -f

# Upgrade packages
pkg-static upgrade -y

# Install basic packages for Linux-like development
pkg install -y bash git go

rm -rf o || true
git clone https://github.com/xyproto/o
