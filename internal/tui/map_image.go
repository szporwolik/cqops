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
	"image/color"
	"image/jpeg"
	"math"
	"strings"
	"time"

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
	cacheW       int
	cacheH       int
	cached       string
	graylineOn   bool
	graylineSlot int // UTC minute-of-day (0–1439)
}

func newMapRenderer() *mapRenderer { return &mapRenderer{} }

// View renders the map at the given terminal dimensions.
// drawGrayline enables the day/night terminator overlay (CPU-friendly —
// computed only on cache miss, i.e. dimension change or UTC-minute tick).
func (mr *mapRenderer) View(ownLat, ownLon, partnerLat, partnerLon float64, mapW, mapAvailH int, drawGrayline bool) string {
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

	// Grayline slot: UTC minute-of-day, changes every 60 s.
	now := time.Now().UTC()
	graySlot := now.Hour()*60 + now.Minute()

	if mr.cacheW == mapW && mr.cacheH == mapH && mr.graylineOn == drawGrayline &&
		(!drawGrayline || mr.graylineSlot == graySlot) && mr.cached != "" {
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

	// Blend day/night terminator (grayline) into the resized image before
	// ANSI conversion — cheap because it only runs on cache miss.
	if drawGrayline {
		blendGrayline(resized, pixW, pixH, now)
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
	mr.graylineOn = drawGrayline
	mr.graylineSlot = graySlot
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

	// When both stations land on the same cell, render a merged marker.
	sameCell := ownX == partnerX && ownY == partnerY

	ownMark := S.MapOwn.Render("\u25c6")
	partMark := S.MapPartner.Render("\u00d7")
	bothMark := S.MapBoth.Render("\u25ce")

	lines := strings.Split(rendered, "\n")
	for i := range lines {
		if sameCell && i == ownY {
			lines[i] = replaceANSICell(lines[i], ownX, bothMark)
		} else {
			if i == ownY {
				lines[i] = replaceANSICell(lines[i], ownX, ownMark)
			}
			if i == partnerY {
				lines[i] = replaceANSICell(lines[i], partnerX, partMark)
			}
		}
	}

	legend := DimStyle.Render(" ") +
		S.MapOwn.Render("\u25c6") + DimStyle.Render(" My station") +
		DimStyle.Render("  ") +
		S.MapPartner.Render("\u00d7") + DimStyle.Render(" Partner") +
		DimStyle.Render("  ") +
		S.MapBoth.Render("\u25ce") + DimStyle.Render(" Merged")
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

// --- Day/night terminator (grayline) ---------------------------------------

// solarElevation returns the sun's elevation in degrees at a given lat/lon
// and UTC time.  Positive = sun above horizon, negative = below.
// Uses standard solar declination + hour-angle formula.
func solarElevation(lat, lon float64, t time.Time) float64 {
	t = t.UTC()
	doy := float64(t.YearDay())
	// Solar declination (approx, ±0.5° accuracy).
	decl := 23.44 * math.Sin(2*math.Pi*(doy-80)/365.25) * math.Pi / 180

	utcHours := float64(t.Hour()) + float64(t.Minute())/60.0 + float64(t.Second())/3600.0
	// Hour angle: how far the sun is from the local meridian.
	ha := (lon/15.0 - utcHours) * 15.0 * math.Pi / 180

	latRad := lat * math.Pi / 180
	sinAlt := math.Sin(latRad)*math.Sin(decl) + math.Cos(latRad)*math.Cos(decl)*math.Cos(ha)
	return math.Asin(sinAlt) * 180 / math.Pi
}

// blendGrayline darkens the night side of the map and adds a twilight
// transition band.  Day side (elevation > 6°) is unchanged.
func blendGrayline(img *image.RGBA, w, h int, t time.Time) {
	// srcW/srcH are the source map pixel dimensions.
	if mapSrcW == 0 || mapSrcH == 0 {
		return
	}
	const (
		nightAlpha = 0.65 // how much to darken night side
		twilightLo = -6.0 // civil twilight start (degrees)
		twilightHi = 6.0  // civil twilight end
	)
	for py := 0; py < h; py++ {
		// py → latitude: equirectangular, y=0 is top (90°N).
		lat := 90.0 - float64(py)*180.0/float64(h-1)
		if h == 1 {
			lat = 0
		}
		for px := 0; px < w; px++ {
			lon := float64(px)*360.0/float64(w-1) - 180.0
			if w == 1 {
				lon = 0
			}
			elev := solarElevation(lat, lon, t)
			var factor float64
			switch {
			case elev > twilightHi:
				continue // full day — unchanged
			case elev < twilightLo:
				factor = nightAlpha // full night
			default:
				// Linear blend across the twilight band.
				t := (twilightHi - elev) / (twilightHi - twilightLo)
				factor = t * nightAlpha
			}
			r, g, b, a := img.At(px, py).RGBA()
			f := 1.0 - factor
			nr := uint8(float64(r>>8)*f + 0.5)
			ng := uint8(float64(g>>8)*f + 0.5)
			nb := uint8(float64(b>>8)*f + 0.5)
			img.Set(px, py, color.RGBA{R: nr, G: ng, B: nb, A: uint8(a >> 8)})
		}
	}
}
