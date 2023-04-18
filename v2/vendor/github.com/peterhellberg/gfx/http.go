//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image"
	"net/http"
	"time"
)

// HTTP is the HTTP client and user agent used by the gfx package.
type HTTP struct {
	*http.Client
	UserAgent string
}

// HTTPClient is the default client used by Get/GetPNG/GetTileset, etc.
var HTTPClient = HTTP{
	Client: &http.Client{
		Timeout: 30 * time.Second,
	},
	UserAgent: "gfx.HTTPClient",
}

// Get performs a HTTP GET request using the DefaultClient.
func Get(rawurl string) (*http.Response, error) {
	req, err := http.NewRequest(http.MethodGet, rawurl, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", HTTPClient.UserAgent)

	return HTTPClient.Do(req)
}

// GetPNG retrieves a remote PNG using DefaultClient
func GetPNG(rawurl string) (image.Image, error) {
	resp, err := Get(rawurl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return DecodePNG(resp.Body)
}

// GetImage retrieves a remote image using DefaultClient
func GetImage(rawurl string) (image.Image, error) {
	resp, err := Get(rawurl)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return DecodeImage(resp.Body)
}

// GetTileset retrieves a remote tileset using GetPNG.
func GetTileset(p Palette, tileSize image.Point, rawurl string) (*Tileset, error) {
	m, err := GetPNG(rawurl)
	if err != nil {
		return nil, err
	}

	return NewTilesetFromImage(p, tileSize, m), nil
}
