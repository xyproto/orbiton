#!/bin/sh
#
# Create release tarballs/zip for 64-bit linux, BSD and Plan9 + 64-bit ARM + raspberry pi 2/3
#
cd v2
name=o
version=$(grep -i version main.go | head -1 | cut -d' ' -f4 | cut -d'"' -f1)
echo "Version $version"
echo 'Compiling...'
export GOARCH=amd64
echo '* Linux'
GOOS=linux go build -mod=vendor -o $name.linux_amd64
#echo '* Plan9'
#GOOS=plan9 go build -mod=vendor -o $name.plan9
echo '* macOS'
GOOS=darwin go build -mod=vendor -o $name.macos_amd64
echo '* FreeBSD'
GOOS=freebsd go build -mod=vendor -o $name.freebsd_amd64
echo '* NetBSD'
GOOS=netbsd go build -mod=vendor -o $name.netbsd_amd64
# OpenBSD support: https://github.com/pkg/term/issues/27
#echo '* OpenBSD'
#GOOS=openbsd go build -mod=vendor -o $name.openbsd
echo '* Linux armv7 (RPI 2/3/4)'
GOOS=linux GOARCH=arm GOARM=7 go build -mod=vendor -o $name.linux_rpi234
echo '* Linux amd64 static w/ upx'
GOOS=linux CGO_ENABLED=0 GOARCH=amd64 go build -mod=vendor -trimpath -ldflags "-s" -a -o $name.linux_amd64_static && upx $name.linux_amd64_static
echo '* Linux arm64 static'
GOOS=linux CGO_ENABLED=0 GOARCH=arm64 go build -mod=vendor -trimpath -ldflags "-s" -a -o $name.linux_arm64_static
echo '* Linux armv6 static'
GOOS=linux CGO_ENABLED=0 GOARCH=arm GOARM=6 go build -mod=vendor -trimpath -ldflags "-s" -a -o $name.linux_armv6_static
echo '* Linux armv7 static'
GOOS=linux CGO_ENABLED=0 GOARCH=arm GOARM=7 go build -mod=vendor -trimpath -ldflags "-s" -a -o $name.linux_armv7_static
echo '* No OS RISC-V static'
GOOS=noos CGO_ENABLED=0 GOARCH=riscv go build -mod=vendor -trimpath -ldflags "-s" -a -o $name.noos_riscv_static

# Compress the Linux releases with xz
for p in linux_amd64 linux_rpi234 linux_amd64_static linux_arm64_static linux_armv6_static linux_armv7_static noos_riscv_static; do
  echo "Compressing $name-$version.$p.tar.xz"
  mkdir "$name-$version-$p"
  cp ../$name.1 "$name-$version-$p/"
  gzip "$name-$version-$p/$name.1"
  cp $name.$p "$name-$version-$p/$name"
  cp ../LICENSE "$name-$version-$p/"
  tar Jcf "$name-$version-$p.tar.xz" "$name-$version-$p/"
  rm -r "$name-$version-$p"
  rm $name.$p
done

# Compress the other tarballs with gz
for p in macos_amd64 freebsd_amd64 netbsd_amd64; do
  echo "Compressing $name-$version.$p.tar.gz"
  mkdir "$name-$version-$p"
  cp ../$name.1 "$name-$version-$p/"
  gzip "$name-$version-$p/$name.1"
  cp $name.$p "$name-$version-$p/$name"
  cp ../LICENSE "$name-$version-$p/"
  tar zcf "$name-$version-$p.tar.gz" "$name-$version-$p/"
  rm -r "$name-$version-$p"
  rm $name.$p
done
cd ..

mkdir -p release
mv -v v2/$name-$version* release
