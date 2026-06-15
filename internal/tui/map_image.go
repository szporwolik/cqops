// Package tui — embedded world map renderer.
//
// The world map JPEG is compiled into the binary via Go embed, decoded
// at startup, and rendered as half-block ANSI true-colour for terminal display.
// Markers for own/partner station are drawn over the map.
//
// World map raster derived from Natural Earth public domain data.

package tui

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"math"
	"strings"

	"github.com/szporwolik/cqops/assets"
)

// mapImg holds the decoded world map image, loaded once at startup.
var mapImg image.Image
var mapSrcW, mapSrcH int
var mapAspect float64

func init() {
	img, err := jpeg.Decode(bytes.NewReader(assets.WorldMap))
	if err == nil {
		mapImg = img
		b := img.Bounds()
		mapSrcW = b.Dx()
		mapSrcH = b.Dy()
		if mapSrcH > 0 {
			mapAspect = float64(mapSrcW) / float64(mapSrcH)
		}
	}
}

// --- Map renderer -----------------------------------------------------------

type mapRenderer struct {
	cacheW int
	cacheH int
	cached string
}

func newMapRenderer() *mapRenderer { return &mapRenderer{} }

// View renders the map at the given terminal dimensions.
func (mr *mapRenderer) View(ownLat, ownLon, partnerLat, partnerLon float64, mapW, mapAvailH int) string {
	if mapImg == nil || mapW < 20 || mapAvailH < 2 {
		return renderWorldMap(ownLat, ownLon, partnerLat, partnerLon, mapW, mapAvailH)
	}
	if mapW > partnerMapMaxW {
		mapW = partnerMapMaxW
	}

	// Compute height: each terminal row paints 2 source pixels (half-block ▀).
	// mapW / (mapH * 2) = mapAspect  =>  mapH = mapW / (2 * mapAspect)
	mapH := mapW
	if mapAspect > 0 {
		mapH = int(float64(mapW) / (2 * mapAspect))
	}
	if mapH < 1 {
		mapH = 1
	}
	if mapH > mapAvailH {
		mapH = mapAvailH
		mapW = int(float64(mapH) * 2 * mapAspect)
	}
	const maxH = 30
	if mapH > maxH {
		mapH = maxH
		mapW = int(float64(mapH) * 2 * mapAspect)
	}
	if mapW < 20 {
		mapW = 20
	}

	if mr.cacheW == mapW && mr.cacheH == mapH && mr.cached != "" {
		return mr.drawMarkers(mr.cached, ownLat, ownLon, partnerLat, partnerLon, mapW, mapH)
	}

	// Resize source to (mapW × (mapH*2)) pixels.
	pixW, pixH := mapW, mapH*2
	resized := image.NewRGBA(image.Rect(0, 0, pixW, pixH))
	sb := mapImg.Bounds()
	for y := 0; y < pixH; y++ {
		sy := y * sb.Dy() / pixH
		for x := 0; x < pixW; x++ {
			sx := x * sb.Dx() / pixW
			resized.Set(x, y, mapImg.At(sx+sb.Min.X, sy+sb.Min.Y))
		}
	}

	// Convert to half-block ANSI.
	var b strings.Builder
	for y := 0; y < pixH; y += 2 {
		for x := 0; x < pixW; x++ {
			r1, g1, b1, _ := resized.At(x, y).RGBA()
			var r2, g2, b2 uint32
			if y+1 < pixH {
				r2, g2, b2, _ = resized.At(x, y+1).RGBA()
			}
			// RGBA() returns 16-bit values; convert to 8-bit.
			fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm\x1b[48;2;%d;%d;%dm▀",
				r1>>8, g1>>8, b1>>8, r2>>8, g2>>8, b2>>8)
		}
		b.WriteString("\x1b[0m")
		if y+2 < pixH {
			b.WriteByte('\n')
		}
	}

	base := b.String()
	mr.cached = base
	mr.cacheW = mapW
	mr.cacheH = mapH
	return mr.drawMarkers(base, ownLat, ownLon, partnerLat, partnerLon, mapW, mapH)
}

// --- Markers & coordinate mapping -------------------------------------------

func lonLatToMapPixel(lon, lat float64, width, height int) (x, y int) {
	if lon < -180 {
		lon = -180
	}
	if lon > 180 {
		lon = 180
	}
	if lat < -90 {
		lat = -90
	}
	if lat > 90 {
		lat = 90
	}
	xf := (lon + 180.0) / 360.0 * float64(width)
	yf := (90.0 - lat) / 180.0 * float64(height)
	x = int(math.Round(xf))
	y = int(math.Round(yf))
	if x < 0 {
		x = 0
	}
	if x >= width {
		x = width - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= height {
		y = height - 1
	}
	return
}

func (mr *mapRenderer) drawMarkers(rendered string, ownLat, ownLon, partnerLat, partnerLon float64, mapW, mapH int) string {
	srcToCell := func(lat, lon float64) (int, int) {
		px, py := lonLatToMapPixel(lon, lat, mapSrcW, mapSrcH)
		cx := px * mapW / mapSrcW
		cy := py * mapH / mapSrcH
		if cx < 0 {
			cx = 0
		}
		if cx >= mapW {
			cx = mapW - 1
		}
		if cy < 0 {
			cy = 0
		}
		if cy >= mapH {
			cy = mapH - 1
		}
		return cx, cy
	}

	ownX, ownY := srcToCell(ownLat, ownLon)
	partnerX, partnerY := srcToCell(partnerLat, partnerLon)

	// When both stations land on the same cell, nudge partner to the nearest
	// free adjacent cell so both markers remain visible.
	if ownX == partnerX && ownY == partnerY {
		// Try 4-neighbour cells in order: right, left, down, up.
		candidates := [][2]int{
			{partnerX + 1, partnerY},
			{partnerX - 1, partnerY},
			{partnerX, partnerY + 1},
			{partnerX, partnerY - 1},
		}
		for _, c := range candidates {
			cx, cy := c[0], c[1]
			if cx >= 0 && cx < mapW && cy >= 0 && cy < mapH {
				partnerX, partnerY = cx, cy
				break
			}
		}
	}

	ownMark := S.MapOwn.Render("\u25c6")
	partMark := S.MapPartner.Render("\u00d7")

	lines := strings.Split(rendered, "\n")
	for i := range lines {
		if i == ownY {
			lines[i] = replaceANSICell(lines[i], ownX, ownMark)
		}
		if i == partnerY {
			lines[i] = replaceANSICell(lines[i], partnerX, partMark)
		}
	}

	legend := DimStyle.Render(" ") +
		S.MapOwn.Render("\u25c6") + DimStyle.Render(" My station") +
		DimStyle.Render("  ") +
		S.MapPartner.Render("\u00d7") + DimStyle.Render(" Partner")
	lines = append(lines, legend)

	return strings.Join(lines, "\n")
}

func replaceANSICell(line string, col int, marker string) string {
	var b strings.Builder
	pos := 0
	inEsc := false
	placed := false
	for _, r := range line {
		if inEsc {
			b.WriteRune(r)
			if r == 'm' || r == 'H' || r == 'f' || r == 'h' || r == 'l' {
				inEsc = false
			}
			continue
		}
		if r == '\x1b' {
			inEsc = true
			b.WriteRune(r)
			continue
		}
		if pos == col && !placed {
			b.WriteString(marker)
			placed = true
		} else {
			b.WriteRune(r)
		}
		pos++
	}
	return b.String()
}
