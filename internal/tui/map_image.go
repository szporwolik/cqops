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
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"strconv"
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
	graylineSlot int // UTC 5-minute slot (0–287); changes every 300 s

	// Reusable RGBA buffer — avoids allocation on every cache miss.
	buf    *image.RGBA
	bufCap int // total pixels (W×H) the buffer can hold
}

func newMapRenderer() *mapRenderer { return &mapRenderer{} }

// Invalidate clears the base image cache so the next View() fully rebuilds.
func (mr *mapRenderer) Invalidate() {
	mr.cached = ""
}

// View renders the map at the given terminal dimensions.
// drawGrayline enables the day/night terminator overlay (CPU-friendly —
// computed only on cache miss, i.e. dimension change or UTC-minute tick).
func (mr *mapRenderer) View(ownLat, ownLon, partnerLat, partnerLon float64, mapW, mapAvailH int, drawGrayline bool) string {
	base := mr.renderBase(mapW, mapAvailH, drawGrayline)
	if base == "" {
		return renderWorldMap(ownLat, ownLon, partnerLat, partnerLon, mapW, mapAvailH)
	}
	// Use the cached (possibly adjusted) dimensions — not the requested ones.
	return mr.drawMarkers(base, ownLat, ownLon, partnerLat, partnerLon, mr.cacheW, mr.cacheH)
}

// BaseImage returns the raw ANSI map image without any markers or legend.
func (mr *mapRenderer) BaseImage(mapW, mapAvailH int, drawGrayline bool) string {
	return mr.renderBase(mapW, mapAvailH, drawGrayline)
}

// renderBase computes dimensions, resizes the source image, and caches the
// ANSI output. Returns the cached base image (no markers, no legend).
func (mr *mapRenderer) renderBase(mapW, mapAvailH int, drawGrayline bool) string {
	if mapImg == nil || mapW < 20 || mapAvailH < 2 {
		return ""
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

	// Grayline slot: 5-minute bucket.
	now := time.Now().UTC()
	graySlot := now.Hour()*12 + now.Minute()/5

	if mr.cacheW == mapW && mr.cacheH == mapH && mr.graylineOn == drawGrayline &&
		(!drawGrayline || mr.graylineSlot == graySlot) && mr.cached != "" {
		return mr.cached
	}

	// Resize source to (mapW × (mapH*2)) pixels.
	// Reuse a pre-allocated RGBA buffer to avoid per-frame allocation.
	pixW, pixH := mapW, mapH*2
	need := pixW * pixH
	if mr.buf == nil || mr.bufCap < need {
		mr.buf = image.NewRGBA(image.Rect(0, 0, pixW, pixH))
		mr.bufCap = need
	}
	resized := mr.buf.SubImage(image.Rect(0, 0, pixW, pixH)).(*image.RGBA)
	sb := mapImg.Bounds()
	for y := 0; y < pixH; y++ {
		sy := y * sb.Dy() / pixH
		for x := 0; x < pixW; x++ {
			sx := x * sb.Dx() / pixW
			resized.Set(x, y, mapImg.At(sx+sb.Min.X, sy+sb.Min.Y))
		}
	}

	if drawGrayline {
		blendGrayline(resized, pixW, pixH, now)
	}

	// Convert to half-block ANSI using a []byte buffer and strconv to avoid
	// fmt.Fprintf per cell (~1200+ calls on a typical terminal).
	// Pre-allocate: each cell is ~48 bytes of escape + separator + newline.
	cellCount := (pixH / 2) * pixW
	buf := make([]byte, 0, cellCount*48+64)
	fgPrefix := []byte("\x1b[38;2;")
	bgPrefix := []byte("\x1b[48;2;")
	semi := []byte(";")
	endm := []byte("m")
	reset := []byte("\x1b[0m")
	block := []byte("▀")
	for y := 0; y < pixH; y += 2 {
		for x := 0; x < pixW; x++ {
			r1, g1, b1, _ := resized.At(x, y).RGBA()
			var r2, g2, b2 uint32
			if y+1 < pixH {
				r2, g2, b2, _ = resized.At(x, y+1).RGBA()
			}
			buf = append(buf, fgPrefix...)
			buf = strconv.AppendUint(buf, uint64(r1>>8), 10)
			buf = append(buf, semi...)
			buf = strconv.AppendUint(buf, uint64(g1>>8), 10)
			buf = append(buf, semi...)
			buf = strconv.AppendUint(buf, uint64(b1>>8), 10)
			buf = append(buf, endm...)
			buf = append(buf, bgPrefix...)
			buf = strconv.AppendUint(buf, uint64(r2>>8), 10)
			buf = append(buf, semi...)
			buf = strconv.AppendUint(buf, uint64(g2>>8), 10)
			buf = append(buf, semi...)
			buf = strconv.AppendUint(buf, uint64(b2>>8), 10)
			buf = append(buf, endm...)
			buf = append(buf, block...)
		}
		buf = append(buf, reset...)
		if y+2 < pixH {
			buf = append(buf, '\n')
		}
	}

	base := string(buf)
	mr.cached = base
	mr.cacheW = mapW
	mr.cacheH = mapH
	mr.graylineOn = drawGrayline
	mr.graylineSlot = graySlot
	return base
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

// equationOfTime returns the correction in minutes to convert clock time to
// solar time.  Range ±16 min.  Accuracy ~±30 s.
func equationOfTime(doy float64) float64 {
	b := 2 * math.Pi * (doy - 81) / 365.0
	return 9.87*math.Sin(2*b) - 7.53*math.Cos(b) - 1.5*math.Sin(b)
}

// solarElevationFast returns the sun's elevation in degrees at a given
// lat/lon, given pre-computed solar declination (radians) and effective
// UTC hours (already corrected for the equation of time).
func solarElevationFast(lat, lon, decl, effUTCHours float64) float64 {
	// Subsolar longitude: at effUTCHours, the sun is overhead at
	// lon = (12 − effUTCHours) × 15°.  Hour angle is the difference
	// between the point's longitude and the subsolar longitude.
	subsolarLon := (12.0 - effUTCHours) * 15.0
	ha := (lon - subsolarLon) * math.Pi / 180
	latRad := lat * math.Pi / 180
	sinAlt := math.Sin(latRad)*math.Sin(decl) + math.Cos(latRad)*math.Cos(decl)*math.Cos(ha)
	return math.Asin(sinAlt) * 180 / math.Pi
}

// blendGrayline darkens the night side of the map and adds a twilight
// transition band.  Day side (elevation > 6°) is unchanged.
//
// Solar declination and equation-of-time are computed once (not per-pixel).
// The equation of time corrects for Earth's elliptical orbit, improving
// grayline position accuracy from ~±4° to ~±0.5° longitude.
func blendGrayline(img *image.RGBA, w, h int, t time.Time) {
	if mapSrcW == 0 || mapSrcH == 0 {
		return
	}
	t = t.UTC()
	doy := float64(t.YearDay())

	// Solar declination for this day (±23.44° over the year).
	decl := 23.44 * math.Sin(2*math.Pi*(doy-80)/365.25) * math.Pi / 180

	// Equation of time: corrects clock-UTC to solar-UTC.
	eotMin := equationOfTime(doy)
	utcHours := float64(t.Hour()) + float64(t.Minute())/60.0 + float64(t.Second())/3600.0
	effHours := utcHours + eotMin/60.0

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
			elev := solarElevationFast(lat, lon, decl, effHours)
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
