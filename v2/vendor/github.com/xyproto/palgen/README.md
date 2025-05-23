# palgen ![Build](https://github.com/xyproto/palgen/workflows/Build/badge.svg) [![GoDoc](https://godoc.org/github.com/xyproto/palgen?status.svg)](http://godoc.org/github.com/xyproto/palgen) [![Go Report Card](https://goreportcard.com/badge/github.com/xyproto/palgen)](https://goreportcard.com/report/github.com/xyproto/palgen)

## Render, extract and save palettes

Given an image, create a palette of N colors, or convert True Color images to indexed ones.

Palettes can also be rendered as images.

### Included utilities

* `png256`, for converting a True Color PNG image to an indexed PNG image, with a custom palette of 256 colors.
* `png2png`, for extracting a palette from a True Color PNG image and write the palette as an indexed 256 color PNG image.
* `png2gpl`, for extracting a palette from a True Color PNG image and write the palette as a GIMP palette file (`.gpl`).
* `png2act`, for extracting a palette from a True Color PNG image and write the palette as a Photoshop palette file (`.act`).

### Palette extraction

| source image | extracted 256 color palette |
| :---:    | :---:                       |
| ![png](testdata/splash.png) | ![png](testdata/splash_pal.png) |
| ![png](testdata/tm_small.png) | ![png](testdata/tm_small_pal.png) |

The palette can be extracted and saved as a PNG image, using `png2png`, or as a GIMP palette, using `png2gpl`.

Palettes can be sorted by hue, luminance and chroma, using the HCL colorspace and the [go-colorful](https://github.com/lucasb-eyer/go-colorful) package, with the included `palgen.Sort` function. The above palettes are sorted with this method.

### Features and limitations

* Can generate palettes of N colors relatively quickly.
* The palette is generated by first grouping colors into N intensity levels and then use the median color of each group.
* The generated palette is not 100% optimal (based on how the human eye is more sensitive for green etc), but it's usable.
* Can export any given `color.Palette` to a GIMP `.gpl` palette file.
* Can export any given `color.Palette` to a Photoshop `.act` palette file.
* Can convert True Color `image.Image` images to indexed `image.Paletted` images.

### Example use

```go
// Read a PNG file
imageData, err := os.Open("input.png")
if err != nil {
    return err
}

// Decode the PNG image
img, err := png.Decode(imageData)
if err != nil {
    return err
}

// Generate a palette with 256 colors
pal, err := palgen.Generate(img, 256)
if err != nil {
    return err
}

// Output a .gpl palette file with the name "Untitled"
err = palgen.Save(pal, "output.gpl", "Untitled")
if err != nil {
    return err
}
```

### Image comparison

The palettes are generated by palgen

| 8 color palette | 16 color palette | 32 color palette | 64 color palette | 128 color palette | 256 color palette | original |
| :---: | :---: | :---: | :---: | :---: | :---: | :---: |
| ![png](testdata/splash8.png)   | ![png](testdata/splash16.png)   | ![png](testdata/splash32.png)   | ![png](testdata/splash64.png)   | ![png](testdata/splash128.png)   | ![png](testdata/splash256.png)   | ![png](testdata/splash.png) |
| ![png](testdata/tm_small8.png)   | ![png](testdata/tm_small16.png)   | ![png](testdata/tm_small32.png)   | ![png](testdata/tm_small64.png)   | ![png](testdata/tm_small128.png)   | ![png](testdata/tm_small256.png)   | ![png](testdata/tm_small.png) |

### Algorithm

A palette is generated by first dividing the colors in the image into N groups, sorted by intensity. For each group, the median value is used. When the number of colors in a group is even, the average of the two center colors are used as the median, but the two center colors are saved for later and added to the generated palette if there are duplicate colors, adding the most used colors first. The generated palette may be shorter than N if there are not enough colors in the given image. As far as I know, no other software uses this algorithm. It works fine, and is relatively fast, but there even better algorithms out there if you are looking for the optimal palette and want to adjust for which colors the human eye are most sensitive for.

As far as I am aware, this is a unique algorithm that has not been thought of or implemented before (create an issue if not), so I'll hereby name it "Rodseth's Median Intensity Algorithm" or RMIA for short.

### Render existing palettes to images

`palgen` can also be used for rendering palettes to images. Here are the two built-in palettes in the Go standard library, with and without the colors sorted:

#### [Plan9](https://golang.org/pkg/image/color/palette/#Plan9)

| Sorted | Original |
| :---: | :---: |
| ![png](img/plan9.png) | ![png](img/plan9_unsorted.png) |

#### [WebSafe](https://golang.org/pkg/image/color/palette/#WebSafe)

| Sorted | Original |
| :---: | :---: |
| ![png](img/websafe.png) | ![png](img/websafe_unsorted.png) |

And one extra:

#### [Burn](https://github.com/xyproto/burnpal)

| Sorted | Original |
| :---: | :---: |
| ![png](img/burn.png) | ![png](img/burn_unsorted.png) |


### General info

* Version: 1.6.1
* License: BSD-3
* Author: Alexander F. Rødseth &lt;xyproto@archlinux.org&gt;
