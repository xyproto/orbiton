#!/bin/sh -e
#
# Self-modifying script that updates the version numbers
#

# The current version goes here, as the default value
VERSION=${1:-'0.70.0'}

if [ -z "$1" ]; then
  echo "The current version is $VERSION, pass the new version as the first argument if you wish to change it"
  exit 0
fi

echo "Setting the version to $VERSION"

# Update the date and version in the man page, README.md file and also this script
d=$(LC_ALL=C date +'%d %b %Y')

# macOS
sed -E -i '' "s/\"[0-9]* [A-Z][a-z]* [0-9]*\"/\"$d\"/g" o.1 2> /dev/null || true
sed -E -i '' "s/2\.[[:digit:]]+\.[[:digit:]]+/$VERSION/g" o.1 README.md "$0" v2/main.go web/docker.sh 2> /dev/null || true

# Linux
sed -r -i "s/\"[0-9]* [A-Z][a-z]* [0-9]*\"/\"$d\"/g" o.1 2> /dev/null || true
sed -r -i "s/2\.[[:digit:]]+\.[[:digit:]]+/$VERSION/g" o.1 README.md "$0" v2/main.go web/docker.sh 2> /dev/null || true
