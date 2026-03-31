package imagepreview

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	_ "github.com/dkua/go-ico"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
	_ "github.com/xfmoulet/qoi"
)

// LoadAndEncode decodes an image file, re-encodes it as PNG, and base64-encodes
// the result. The returned PreviewResult is ready to be passed to FlushImage.
//
// PNG files are forwarded verbatim when they are large enough to not require
// upscaling. All other formats (JPEG, GIF, etc.) are decoded and re-encoded as
// PNG because the Kitty graphics protocol only supports raw pixel data and PNG
// (f=100). Small images are pre-scaled with nearest-neighbor so Kitty renders
// sharp pixels instead of a blurry bilinear upscale.
//
// SVG files are rendered via native Go packages; JPEG XL files are converted via
// ImageMagick (magick).
func LoadAndEncode(ctx context.Context, path string, panePixW, panePixH uint) (PreviewResult, error) {
	ext := strings.ToLower(filepath.Ext(path))

	var encoded string
	var imgW, imgH uint

	if ext == ".svg" {
		// SVG: render via native Go packages at the pane's pixel dimensions.
		w := panePixW
		if w == 0 {
			w = 800
		}

		f, err := os.Open(path)
		if err != nil {
			return PreviewResult{}, err
		}
		defer f.Close()

		icon, err := oksvg.ReadIconStream(f)
		if err != nil {
			return PreviewResult{}, err
		}

		var h uint
		if icon.ViewBox.W != 0 {
			h = uint(float64(w) * icon.ViewBox.H / icon.ViewBox.W)
		}
		if h == 0 {
			h = w
		}
		icon.SetTarget(0, 0, float64(w), float64(h))

		img := image.NewRGBA(image.Rect(0, 0, int(w), int(h)))
		scanner := rasterx.NewScannerGV(int(w), int(h), img, img.Bounds())
		dasher := rasterx.NewDasher(int(w), int(h), scanner)

		icon.Draw(dasher, 1.0)

		imgW, imgH = w, h
		var buf bytes.Buffer
		if err := png.Encode(&buf, img); err != nil {
			return PreviewResult{}, err
		}
		if ctx.Err() != nil {
			return PreviewResult{}, ctx.Err()
		}
		encoded = base64.StdEncoding.EncodeToString(buf.Bytes())
	} else if ext == ".jxl" {
		// JPEG XL: convert via ImageMagick.
		enc, iw, ih, err := ConvertToPNG(ctx, "magick", path, "png:-")
		if err != nil {
			return PreviewResult{}, err
		}
		if ctx.Err() != nil {
			return PreviewResult{}, ctx.Err()
		}
		encoded, imgW, imgH = enc, iw, ih
	} else {
		// Standard bitmap formats via Go's image package.
		f, err := os.Open(path)
		if err != nil {
			return PreviewResult{}, err
		}
		defer f.Close()

		// Use DecodeConfig to read dimensions from the header cheaply.
		config, format, err := image.DecodeConfig(f)
		if err != nil {
			return PreviewResult{}, err
		}
		if ctx.Err() != nil {
			return PreviewResult{}, ctx.Err()
		}
		imgW = uint(config.Width)
		imgH = uint(config.Height)

		// needsUpscale is true when the image is smaller than the pane in both
		// dimensions and would be stretched by Kitty's bilinear filter.
		needsUpscale := imgW < panePixW && imgH < panePixH

		if format == "png" && ext == ".png" && !needsUpscale {
			// PNG can be forwarded verbatim -- Kitty accepts it as f=100.
			data, err := os.ReadFile(path)
			if err != nil {
				return PreviewResult{}, err
			}
			if ctx.Err() != nil {
				return PreviewResult{}, ctx.Err()
			}
			encoded = base64.StdEncoding.EncodeToString(data)
		} else {
			// JPEG, GIF, or small PNG: decode and re-encode as PNG.
			if _, err := f.Seek(0, 0); err != nil {
				return PreviewResult{}, err
			}
			img, _, err := image.Decode(f)
			if err != nil {
				return PreviewResult{}, err
			}
			if ctx.Err() != nil {
				return PreviewResult{}, ctx.Err()
			}
			if needsUpscale {
				// Scale up with nearest-neighbor so Kitty renders sharp pixels
				// rather than a blurry bilinear upscale.
				var targetW, targetH uint
				if imgW*panePixH > imgH*panePixW {
					targetW = panePixW
					targetH = panePixW * imgH / imgW
				} else {
					targetH = panePixH
					targetW = panePixH * imgW / imgH
				}
				img = ScaleNearestNeighbor(img, int(targetW), int(targetH))
				imgW, imgH = targetW, targetH
			}
			var buf bytes.Buffer
			if err := png.Encode(&buf, img); err != nil {
				return PreviewResult{}, err
			}
			if ctx.Err() != nil {
				return PreviewResult{}, ctx.Err()
			}
			encoded = base64.StdEncoding.EncodeToString(buf.Bytes())
		}
	}

	if ctx.Err() != nil {
		return PreviewResult{}, ctx.Err()
	}
	return PreviewResult{Path: path, Encoded: encoded, ImgW: imgW, ImgH: imgH}, nil
}

// ConvertToPNG runs an external command that writes PNG data to stdout,
// base64-encodes the result, and returns the encoded string with pixel
// dimensions.
func ConvertToPNG(ctx context.Context, args ...string) (encoded string, w, h uint, err error) {
	cmd := exec.CommandContext(ctx, args[0], args[1:]...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	if err = cmd.Run(); err != nil {
		return
	}
	cfg, _, cfgErr := image.DecodeConfig(bytes.NewReader(buf.Bytes()))
	if cfgErr != nil {
		err = cfgErr
		return
	}
	encoded = base64.StdEncoding.EncodeToString(buf.Bytes())
	w, h = uint(cfg.Width), uint(cfg.Height)
	return
}
