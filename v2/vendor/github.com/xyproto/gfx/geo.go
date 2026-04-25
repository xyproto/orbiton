//go:build !tinygo
// +build !tinygo

package gfx

import (
	"image"
	"image/draw"
	"math"
)

// GeoPoint represents a geographic point with Lat/Lon.
type GeoPoint struct {
	Lon float64
	Lat float64
}

// GP creates a new GeoPoint
func GP(lat, lon float64) GeoPoint {
	return GeoPoint{Lon: lon, Lat: lat}
}

// Vec returns a vector for the geo point based on the given tileSize and zoom level.
func (gp GeoPoint) Vec(tileSize, zoom int) Vec {
	scale := math.Pow(2, float64(zoom))
	fts := float64(tileSize)

	return V(
		((float64(gp.Lon)+180)/360)*scale*fts,
		(fts/2)-(fts*math.Log(math.Tan((Pi/4)+((float64(gp.Lat)*Pi/180)/2)))/(2*Pi))*scale,
	)
}

// In returns a Vec for the position of the GeoPoint in a GeoTile.
func (gp GeoPoint) In(gt GeoTile, tileSize int) Vec {
	return gt.Vec(gp, tileSize)
}

// GeoTile for the GeoPoint at the given zoom level.
func (gp GeoPoint) GeoTile(zoom int) GeoTile {
	latRad := Degrees(gp.Lat).Radians()
	n := math.Pow(2, float64(zoom))

	return GeoTile{
		Zoom: zoom,
		X:    int(n * (float64(gp.Lon) + 180) / 360),
		Y:    int((1.0 - math.Log(math.Tan(latRad)+(1/math.Cos(latRad)))/Pi) / 2.0 * n),
	}
}

// NewGeoPointFromTileNumbers creates a new GeoPoint based on the given tile numbers.
// https://wiki.openstreetmap.org/wiki/Slippy_map_tilenames#Tile_numbers_to_lon..2Flat.
func NewGeoPointFromTileNumbers(zoom, x, y int) GeoPoint {
	n := math.Pow(2, float64(zoom))
	latRad := math.Atan(math.Sinh(Pi * (1 - (2 * float64(y) / n))))

	return GP(latRad*180/Pi, (float64(x)/n*360)-180)
}

// GeoTiles is a slice of GeoTile.
type GeoTiles []GeoTile

// GeoTile consists of a Zoom level, X and Y values.
type GeoTile struct {
	Zoom int
	X    int
	Y    int
}

// GT creates a new GeoTile.
func GT(zoom, x, y int) GeoTile {
	return GeoTile{Zoom: zoom, X: x, Y: y}
}

// GeoPoint for the GeoTile.
func (gt GeoTile) GeoPoint() GeoPoint {
	n := math.Pow(2, float64(gt.Zoom))
	latRad := math.Atan(math.Sinh(Pi * (1 - (2 * float64(gt.Y) / n))))

	return GP(latRad*180/Pi, (float64(gt.X)/n*360)-180)
}

// Vec returns the Vec for the GeoPoint in the GeoTile.
func (gt GeoTile) Vec(gp GeoPoint, tileSize int) Vec {
	return gp.Vec(tileSize, gt.Zoom).Sub(gt.GeoPoint().Vec(tileSize, gt.Zoom))
}

// Rawurl formats a URL string with Zoom, X and Y.
func (gt GeoTile) Rawurl(format string) string {
	return Sprintf(format, gt.Zoom, gt.X, gt.Y)
}

// AddXY adds x and y.
func (gt GeoTile) AddXY(x, y int) GeoTile {
	return GT(gt.Zoom, gt.X+x, gt.Y+y)
}

// Neighbors returns the neighboring tiles.
func (gt GeoTile) Neighbors() GeoTiles {
	return GeoTiles{
		gt.N(),
		gt.NE(),
		gt.E(),
		gt.SE(),
		gt.S(),
		gt.SW(),
		gt.W(),
		gt.NW(),
	}
}

// N is the tile to the north.
func (gt GeoTile) N() GeoTile {
	if gt.Zoom > 0 && gt.Y > 0 {
		gt.Y--
	}

	return gt
}

// NE is the tile to the northeast.
func (gt GeoTile) NE() GeoTile {
	if gt.Zoom > 0 {
		if gt.Y > 0 {
			gt.Y--
		}

		gt.X++
	}

	return gt
}

// E is the tile to the east.
func (gt GeoTile) E() GeoTile {
	if gt.Zoom > 0 {
		gt.X++
	}

	return gt
}

// SE is the tile to the southeast.
func (gt GeoTile) SE() GeoTile {
	if gt.Zoom > 0 {
		gt.X++
		gt.Y++
	}

	return gt
}

// S is the tile to the south.
func (gt GeoTile) S() GeoTile {
	if gt.Zoom > 0 {
		gt.Y++
	}

	return gt
}

// SW is the tile to the southwest.
func (gt GeoTile) SW() GeoTile {
	if gt.Zoom > 0 {
		if gt.X > 0 {
			gt.X--
		}

		gt.Y++
	}

	return gt
}

// W is the tile to the west.
func (gt GeoTile) W() GeoTile {
	if gt.Zoom > 0 && gt.X > 0 {
		gt.X--
	}

	return gt
}

// NW is the tile to the northwest.
func (gt GeoTile) NW() GeoTile {
	if gt.Zoom > 0 {
		if gt.Y > 0 {
			gt.Y--
		}

		if gt.X > 0 {
			gt.X--
		}
	}

	return gt
}

// GetImage for the tile.
func (gt GeoTile) GetImage(format string) (image.Image, error) {
	return GetImage(gt.Rawurl(format))
}

// Draw the tile on dst.
func (gt GeoTile) Draw(dst draw.Image, gp GeoPoint, src image.Image) {

	Draw(dst, gt.Bounds(dst, gp, src.Bounds().Dx()), src)
}

// Bounds returns an image.Rectangle for the GeoTile based on the dst, gp and tileSize.
func (gt GeoTile) Bounds(dst image.Image, gp GeoPoint, tileSize int) image.Rectangle {
	c := BoundsCenter(dst.Bounds())

	return dst.Bounds().Add(c.Pt()).Sub(gp.In(gt, tileSize).Pt())
}

// GeoTileServer represents a tile server.
type GeoTileServer struct {
	Format string
}

// GTS creates a GeoTileServer.
func GTS(format string) GeoTileServer {
	return GeoTileServer{Format: format}
}

// GetImage for the given GeoTile from the tile server.
func (gts GeoTileServer) GetImage(gt GeoTile) (image.Image, error) {
	return gt.GetImage(gts.Format)
}

// DrawTileAndNeighbors on dst.
func (gts GeoTileServer) DrawTileAndNeighbors(dst draw.Image, gt GeoTile, gp GeoPoint) error {
	if err := gts.DrawTile(dst, gt, gp); err != nil {
		return nil
	}

	return gts.DrawNeighbors(dst, gt, gp)
}

// DrawTile on dst.
func (gts GeoTileServer) DrawTile(dst draw.Image, gt GeoTile, gp GeoPoint) error {
	src, err := gts.GetImage(gt)
	if err != nil {
		return err
	}

	gt.Draw(dst, gp, src)

	return nil
}

// DrawNeighbors on dst.
func (gts GeoTileServer) DrawNeighbors(dst draw.Image, gt GeoTile, gp GeoPoint) error {
	for _, n := range gt.Neighbors() {
		if err := gts.DrawTile(dst, n, gp); err != nil {
			return err
		}
	}

	return nil
}
