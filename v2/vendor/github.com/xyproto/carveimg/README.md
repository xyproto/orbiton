# Carve & Img

Two image viewing utilities for the terminal. Both of them can display images in only 16 colors.

* `carve` - uses content-aware image resizing before displaying the image
* `img` - uses regular image resizing before displaying the image

## Screenshots

| Original PNG image                    | In a VT100 compatible terminal emulator, using seam carving for content-aware image resizing |
|---------------------------------------|----------------------------------------------------------------------------------------------|
| <img src=img/grumpycat.png width=512> |                                                <img src=img/grumpycat16colors.png width=512> |

| Original PNG image                           | In `carve` (wonky, but higher information density)       | In `img` (may look better, but retains less information |
|----------------------------------------------|----------------------------------------------------------|---------------------------------------------------------|
| <img src=img/goals_objectives.png width=512> |<img src=img/goals_objectives_carve.png width=512>        | <img src=img/goals_objectives_img.png width=512>        |

## Installation

With Go 1.17 or later:

    go install github.com/xyproto/carveimg/cmd/img@latest
    go install github.com/xyproto/carveimg/cmd/carve@latest

## The `carve` utility

* The image resizing is done with [`github.com/esimov/caire`](https://github.com/esimov/caire).
* The palette reduction is done with [`github.com/xyproto/palgen`](https://github.com/xyproto/palgen).
* The image reszing may be very slow for larger images.

## The `img` utilitiy

* The image resizing is done with [`golang.org/x/image/draw`](https://golang.org/x/image/draw) and the [`CatmullRom`](https://pkg.go.dev/golang.org/x/image@v0.3.0/draw#pkg-variables) kernel.
* The palette reduction is done with [`github.com/xyproto/palgen`](https://github.com/xyproto/palgen).

## General info

* Version: 1.4.9
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
