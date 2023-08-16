go-ico
======

A library for parsing and working with `.ico` image files. Compatible with Go’s standard `image` library.

*NOTE: This library is not being maintained and might not fully work anymore.*

## Installation

```
go get github.com/dkua/go-ico
```

## Dependencies

There is a single dependency on [github.com/jsummers/gobmp](http://github.com/jsummers/gobmp), a library for working with `.bmp` files in Go.
There is no builtin support for `.bmp` in the `image` package, there is an experimental library in `image/x/bmp` but it is not very good.

## Usage

```
reader, err := os.Open("example.ico")
if err != nil {
        log.Fatal(err)
}
defer reader.Close()

// To decode and return the first (and usually largest) image of an .ico image
image, err := Decode(r)  // image is of image.Image type
if err != nil {
        log.Fatal(err)
}

// To decode and return all the images of an .ico image
ic, err := DecodeAll(r)  // ic is a custom ico.ICO containing an array of image.Image
if err != nil {
        log.Fatal(err)
}
```
