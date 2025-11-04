## jpegxl
[![Status](https://github.com/gen2brain/jpegxl/actions/workflows/test.yml/badge.svg)](https://github.com/gen2brain/jpegxl/actions)
[![Go Reference](https://pkg.go.dev/badge/github.com/gen2brain/jpegxl.svg)](https://pkg.go.dev/github.com/gen2brain/jpegxl)

Go encoder/decoder for [JPEG XL Image File Format](https://en.wikipedia.org/wiki/JPEG_XL) with support for animated JXL images (decode only).

Based on [libjxl](https://github.com/libjxl/libjxl) compiled to [WASM](https://en.wikipedia.org/wiki/WebAssembly) and used with [wazero](https://wazero.io/) runtime (CGo-free).

The library will first try to use a dynamic/shared library (if installed) via [purego](https://github.com/ebitengine/purego) and will fall back to WASM.

### Build tags

* `nodynamic` - do not use dynamic/shared library (use only WASM)