#!/bin/sh
#
# Create release tarballs/zip-files
#

platforms="
  linux,amd64,,linux_x86_64_static,tar.xz
  linux,arm64,,linux_aarch64_static,tar.xz
  linux,arm,6,linux_armv6_static,tar.xz
  linux,arm,7,linux_armv7_static,tar.xz
  linux,riscv64,,linux_riscv64_static,tar.xz
  darwin,amd64,,macos_x86_64_static,tar.gz
  darwin,arm64,,macos_aarch64_static,tar.gz
  freebsd,amd64,,freebsd_x86_64_static,tar.gz
  freebsd,arm64,,freebsd_aarch64_static,tar.gz
  freebsd,arm,6,freebsd_armv6_static,tar.gz
  freebsd,arm,7,freebsd_armv7_static,tar.gz
  freebsd,386,,freebsd_i386_static,tar.gz
  freebsd,riscv64,,freebsd_riscv64_static,tar.gz
  netbsd,amd64,,netbsd_x86_64_static,tar.gz
  netbsd,arm64,,netbsd_aarch64_static,tar.gz
  netbsd,arm,6,netbsd_armv6_static,tar.gz
  netbsd,arm,7,netbsd_armv7_static,tar.gz
  netbsd,386,,netbsd_i386_static,tar.gz
  openbsd,amd64,,openbsd_x86_64_static,tar.gz
  openbsd,arm64,,openbsd_aarch64_static,tar.gz
  openbsd,arm,6,openbsd_armv6_static,tar.gz
  openbsd,arm,7,openbsd_armv7_static,tar.gz
  openbsd,386,,openbsd_i386_static,tar.gz
  openbsd,riscv64,,openbsd_riscv64_static,tar.gz
  windows,amd64,,windows_x86_64_static,zip
  windows,arm64,,windows_aarch64_static,zip
"

cd v2
name=orbiton
version=$(grep -i version main.go | head -1 | cut -d' ' -f4 | cut -d'"' -f1)
echo "Version $version"

export CGO_ENABLED=0
export GOEXPERIMENT=greenteagc
export GOFLAGS='-mod=vendor -trimpath -buildvcs=false'

compile_and_compress() {
  goos="$1"
  goarch="$2"
  goarm="$3"
  platform="$4"
  compression="$5"
  exe_ext=""
  [ "$goos" = "windows" ] && exe_ext=".exe"

  echo "Compiling $name.$platform..."

  [ -n "$goarm" ] && GOARM="$goarm" || unset GOARM
  GOOS="$goos" GOARCH="$goarch" go build -mod=vendor -trimpath -ldflags="-s -w" -a -o "$name.$platform$exe_ext" || {
    echo "Error: failed to compile for $platform"
    echo "Target: GOOS=$goos GOARCH=$goarch GOARM=$goarm"
    echo "Environment variables: GOOS=$goos GOARCH=$goarch GOARM=$goarm"
    exit 1
  }

  echo "Compressing $name-$version.$platform.$compression"
  mkdir "$name-$version-$platform"
  cp ../o.1 "$name-$version-$platform/"
  gzip "$name-$version-$platform/o.1"
  cp "$name.$platform$exe_ext" "$name-$version-$platform/o$exe_ext"
  cp ../LICENSE "$name-$version-$platform/"

  case "$compression" in
    tar.xz)
      tar Jcf "$name-$version-$platform.$compression" "$name-$version-$platform"
      ;;
    tar.gz)
      tar zcf "$name-$version-$platform.$compression" "$name-$version-$platform"
      ;;
    zip)
      zip -r "$name-$version-$platform.$compression" "$name-$version-$platform"
      ;;
  esac

  rm -r "$name-$version-$platform"
  rm "$name.$platform$exe_ext"
}

echo 'Compiling...'
pids=""
while read -r p; do
  [ -z "$p" ] && continue
  IFS=',' read -r goos goarch goarm platform compression <<EOF
$p
EOF
  compile_and_compress "$goos" "$goarch" "$goarm" "$platform" "$compression" &
  pids="$pids $!"
done <<EOF
$platforms
EOF

build_failed=0
for pid in $pids; do
  if ! wait "$pid"; then
    build_failed=1
  fi
done

if [ "$build_failed" -ne 0 ]; then
  echo "Error: one or more platform builds failed"
  exit 1
fi

cd ..
mkdir -p release
mv -v v2/$name-$version* release
