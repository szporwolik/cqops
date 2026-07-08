// Package tui — embedded world map renderer.
//
// The world map JPEG is compiled into the binary via Go embed, decoded
// at startup, and rendered as half-block ANSI true-colour for terminal display.
// Markers for own/partner station are drawn over the map.
//
// When the Kitty graphics protocol is available, the map is rendered at full
// pixel resolution via the Kitty APC escape sequence instead of half-block ANSI.
// This gives much sharper maps on kitty-compatible terminals (kitty, Ghostty,
// WezTerm).
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

	tea "charm.land/bubbletea/v2"
	"github.com/NimbleMarkets/ntcharts/v2/picture"
	"github.com/szporwolik/cqops/assets"
	"github.com/szporwolik/cqops/internal/applog"
)

const kittyMapImageID = 42070 // partner map
const kittyPSKImageID = 42071 // PSK Reporter map

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

	// Kitty graphics protocol support — partner map.
	kittyPic picture.Model
	kittyOn  bool
	// kittyPending holds SetImage/SetSize cmds dispatched during View(),
	// consumed by Update() on the next frame.
	kittyPending tea.Cmd
	// kittyDirty is set on Invalidate() and cleared in renderKitty
	// after dispatching a new SetImage/SetSize.
	kittyDirty bool
	// kittyW/kittyH track the last SetSize sent to kittyPic so we
	// avoid re-dispatching when dimensions are stable.
	kittyW, kittyH int

	// Kitty graphics protocol support — PSK Reporter map.
	pskKittyPic     picture.Model
	pskKittyPending tea.Cmd
	pskKittyW       int
	pskKittyH       int
	pskKittySig     string // last mapSig sent to SetPSKImage; avoids re-dispatch
}

func newMapRenderer() *mapRenderer {
	return &mapRenderer{
		kittyPic: picture.NewWithConfig(picture.Config{
			KittyID: kittyMapImageID,
		}),
		pskKittyPic: picture.NewWithConfig(picture.Config{
			KittyID: kittyPSKImageID,
		}),
	}
}

// Invalidate clears the base image cache so the next View() fully rebuilds.
func (mr *mapRenderer) Invalidate() {
	mr.cached = ""
	mr.kittyDirty = true
}

// View renders the map at the given terminal dimensions.
// drawGrayline enables the day/night terminator overlay (CPU-friendly —
// computed only on cache miss, i.e. dimension change or UTC-minute tick).
//
// When Kitty graphics protocol is active, the map is rendered at full
// pixel resolution; otherwise the half-block ANSI path is used.
func (mr *mapRenderer) View(ownLat, ownLon, partnerLat, partnerLon float64, mapW, mapAvailH int, drawGrayline bool) string {
	if mr.kittyOn && picture.KittySupported() == picture.KittyCapabilitySupported {
		return mr.renderKitty(ownLat, ownLon, partnerLat, partnerLon, mapW, mapAvailH, drawGrayline)
	}
	base := mr.renderBase(mapW, mapAvailH, drawGrayline)
	if base == "" {
		return ""
	}
	// Use the cached (possibly adjusted) dimensions — not the requested ones.
	return mr.drawMarkers(base, ownLat, ownLon, partnerLat, partnerLon, mr.cacheW, mr.cacheH)
}

// BaseImage returns the raw ANSI map image without any markers or legend.
func (mr *mapRenderer) BaseImage(mapW, mapAvailH int, drawGrayline bool) string {
	return mr.renderBase(mapW, mapAvailH, drawGrayline)
}

// --- Kitty graphics protocol support ---------------------------------------

// SetKittyEnabled enables or disables Kitty graphics protocol rendering.
// Returns a Cmd that the caller must batch into their Update() return.
func (mr *mapRenderer) SetKittyEnabled(enabled bool) tea.Cmd {
	if enabled && picture.KittySupported() != picture.KittyCapabilitySupported {
		return nil // terminal doesn't support it; don't toggle
	}
	if mr.kittyOn == enabled {
		return nil
	}
	mr.kittyOn = enabled
	mr.kittyDirty = true
	mr.pskKittySig = "" // force re-dispatch after mode switch
	applog.Debug("PSK SetKittyEnabled", "on", enabled, "kittyMode", mr.pskKittyPic.Mode())
	return tea.Batch(mr.kittyPic.Toggle(), mr.pskKittyPic.Toggle())
}

// Update forwards messages to the embedded picture.Model and returns any
// pending SetImage/SetSize Cmds stored by the last View() call.
// The caller MUST call this on every Update() frame for Kitty graphics to work.
func (mr *mapRenderer) Update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if c := mr.kittyPic.Update(msg); c != nil {
		cmd = tea.Batch(cmd, c)
	}
	if mr.kittyPending != nil {
		cmd = tea.Batch(cmd, mr.kittyPending)
		mr.kittyPending = nil
	}
	return cmd
}

// renderKitty prepares the map image with pixel-level markers, dispatches
// it to the Kitty picture model, and returns the current Kitty render output
// with a text legend appended below. Caches are used to avoid redundant
// PNG encodes.
func (mr *mapRenderer) renderKitty(ownLat, ownLon, partnerLat, partnerLon float64, mapW, mapAvailH int, drawGrayline bool) string {
	// Compute dimensions — same logic as renderBase.
	if mapImg == nil || mapW < 20 || mapAvailH < 2 {
		return ""
	}
	if mapW > partnerMapMaxW {
		mapW = partnerMapMaxW
	}
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

	now := time.Now().UTC()
	graySlot := now.Hour()*12 + now.Minute()/5

	// Cache check: dimensions, grayline, AND kitty frame readiness must
	// all match. kittyW/H are only set when a valid kitty frame was
	// previously displayed; 0 means "no kitty frame yet."
	if !mr.kittyDirty && mr.kittyW == mapW && mr.kittyH == mapH &&
		mr.graylineOn == drawGrayline && mr.graylineSlot == graySlot &&
		mr.kittyW > 0 {
		kittyOut := mr.kittyPic.View().Content
		if kittyOut != "" && !isGlyphFallback(kittyOut) {
			return mr.appendKittyLegend(kittyOut)
		}
		// Glyph fallback — fall through to rebuild.
	}

	// Render at the picture model's cell-pixel resolution so no
	// upscaling is needed — avoids blur from CatmullRom enlargement.
	cpw, cph := mr.kittyPic.CellPixelSize()
	pixW, pixH := mapW*cpw, mapH*cph

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

	// Draw pixel-level station markers with precise lat/lon math.
	mr.drawPixelMarker(resized, pixW, pixH, ownLat, ownLon, partnerLat, partnerLon)

	cmd1 := mr.kittyPic.SetSize(mapW, mapH)
	cmd2 := mr.kittyPic.SetImage(resized)
	mr.kittyPending = tea.Sequence(cmd1, cmd2)

	// Always mark grayline slot so the ANSI fallback path (renderBase)
	// shares the same timing anchor and doesn't disagree on cache.
	mr.graylineOn = drawGrayline
	mr.graylineSlot = graySlot

	kittyOut := mr.kittyPic.View().Content
	if kittyOut == "" || isGlyphFallback(kittyOut) {
		// Kitty frame not ready yet — use the ANSI map as a
		// height-stable fallback. Do NOT set kittyW/H;
		// mr.kittyDirty stays false so the next frame hits
		// the ANSI fallback again until kittyOut is ready.
		return mr.drawMarkers(mr.renderBase(mapW, mapAvailH, drawGrayline),
			ownLat, ownLon, partnerLat, partnerLon, mr.cacheW, mr.cacheH)
	}

	// Kitty frame arrived — persist the cache dimensions so future
	// frames short-circuit on the cheap cache check.
	mr.kittyDirty = false
	mr.kittyW = mapW
	mr.kittyH = mapH
	return mr.appendKittyLegend(kittyOut)
}

// appendKittyLegend appends a text legend row below the Kitty image output.
func (mr *mapRenderer) appendKittyLegend(kittyOut string) string {
	legend := DimStyle.Render(" ") +
		S.MapOwn.Render("\u25c6") + DimStyle.Render(" My station") +
		DimStyle.Render("  ") +
		S.MapPartner.Render("\u00d7") + DimStyle.Render(" Partner") +
		DimStyle.Render("  ") +
		S.MapBoth.Render("\u25ce") + DimStyle.Render(" Merged")
	if kittyOut == "" {
		return legend
	}
	return kittyOut + "\n" + legend
}

// --- PSK Reporter Kitty support -------------------------------------------

// PSKUpdate forwards messages to the PSK picture model and returns any
// pending SetImage/SetSize Cmds. Must be called on every Update() frame.
func (mr *mapRenderer) PSKUpdate(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	if c := mr.pskKittyPic.Update(msg); c != nil {
		cmd = tea.Batch(cmd, c)
	}
	if mr.pskKittyPending != nil {
		applog.Debug("PSK PSKUpdate: returning pending cmd")
		cmd = tea.Batch(cmd, mr.pskKittyPending)
		mr.pskKittyPending = nil
	}
	return cmd
}

// BaseImageRGBA returns the raw RGBA map image (resized, grayline applied)
// plus its pixel and cell dimensions. Used by PSK Reporter to draw spots
// directly onto the bitmap when Kitty mode is active.
func (mr *mapRenderer) BaseImageRGBA(mapW, mapAvailH int, drawGrayline bool) (*image.RGBA, int, int, int, int) {
	if mapImg == nil || mapW < 20 || mapAvailH < 2 {
		return nil, 0, 0, 0, 0
	}
	if mapW > partnerMapMaxW {
		mapW = partnerMapMaxW
	}
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

	cpw, cph := mr.pskKittyPic.CellPixelSize()
	pixW, pixH := mapW*cpw, mapH*cph

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
		blendGrayline(resized, pixW, pixH, time.Now().UTC())
	}

	return resized, pixW, pixH, mapW, mapH
}

// SetPSKImage dispatches the marked-up PSK map to the PSK Kitty picture
// model. Call after drawing spots onto the RGBA. Only re-dispatches when
// the sig changes — avoids flooding the picture model with SetImage calls
// that would increment the sequence and stall the async Kitty frame.
func (mr *mapRenderer) SetPSKImage(rgba *image.RGBA, mapW, mapH int, sig string) tea.Cmd {
	if sig == mr.pskKittySig {
		return nil // already dispatched for this data
	}
	applog.Debug("PSK SetPSKImage: dispatching", "mapW", mapW, "mapH", mapH, "sig", sig)
	mr.pskKittySig = sig
	mr.pskKittyPending = tea.Sequence(
		mr.pskKittyPic.SetSize(mapW, mapH),
		mr.pskKittyPic.SetImage(rgba),
	)
	return mr.pskKittyPending
}

// PSKView returns the current PSK Kitty picture model output.
func (mr *mapRenderer) PSKView() string {
	return mr.pskKittyPic.View().Content
}

// PSKMode returns the PSK picture model's current rendering mode.
func (mr *mapRenderer) PSKMode() picture.PictureMode { return mr.pskKittyPic.Mode() }

// PSKKittyReady reports whether the real Kitty grid (not the glyph
// fallback) has arrived from the async PNG encode.
func (mr *mapRenderer) PSKKittyReady() bool {
	return mr.pskKittyPic.View().Content != "" && !isGlyphFallback(mr.pskKittyPic.View().Content)
}

// KittyOn reports whether Kitty graphics mode is active on this renderer.
func (mr *mapRenderer) KittyOn() bool { return mr.kittyOn }

// KittyContent returns the current partner-map picture model output.
func (mr *mapRenderer) KittyContent() string { return mr.kittyPic.View().Content }

// isGlyphFallback detects the picture model's half-block glyph fallback
// (▀ U+2580 or ▄ U+2584) which means the real Kitty grid hasn't arrived.
func isGlyphFallback(s string) bool {
	return strings.Contains(s, "\u2580") || strings.Contains(s, "\u2584")
}

// PSKDrawDot draws a filled circle at the given lat/lon on the RGBA image.
// pixW/pixH are the image pixel dimensions.
func (mr *mapRenderer) PSKDrawDot(img *image.RGBA, pixW, pixH int, lat, lon float64, col color.RGBA) {
	xf := (lon + 180.0) / 360.0 * float64(pixW)
	yf := (90.0 - lat) / 180.0 * float64(pixH)
	cx, cy := clampPixel(int(math.Round(xf)), int(math.Round(yf)), pixW, pixH)

	r := pixW / 200 // radius ~0.5% of image width
	if r < 2 {
		r = 2
	}
	if r > 4 {
		r = 4
	}
	for dy := -r; dy <= r; dy++ {
		for dx := -r; dx <= r; dx++ {
			if dx*dx+dy*dy > r*r {
				continue
			}
			x, y := cx+dx, cy+dy
			if x >= 0 && x < pixW && y >= 0 && y < pixH {
				img.Set(x, y, col)
			}
		}
	}
}

// drawPixelMarker paints own/partner station dots directly onto the RGBA
// image using direct lat/lon → pixel math (no double integer division).
// pixW and pixH are the pixel dimensions of the RGBA image.
func (mr *mapRenderer) drawPixelMarker(img *image.RGBA, pixW, pixH int, ownLat, ownLon, partnerLat, partnerLon float64) {
	latLonToPixel := func(lat, lon float64) (px, py int) {
		xf := (lon + 180.0) / 360.0 * float64(pixW)
		yf := (90.0 - lat) / 180.0 * float64(pixH)
		return clampPixel(int(math.Round(xf)), int(math.Round(yf)), pixW, pixH)
	}
	ownX, ownY := latLonToPixel(ownLat, ownLon)
	partnerX, partnerY := latLonToPixel(partnerLat, partnerLon)

	// Marker radius: ~0.5% of image width, min 2px, max 6px.
	r := pixW / 200
	if r < 2 {
		r = 2
	}
	if r > 6 {
		r = 6
	}

	drawDot := func(cx, cy int, col color.RGBA) {
		for dy := -r; dy <= r; dy++ {
			for dx := -r; dx <= r; dx++ {
				if dx*dx+dy*dy > r*r {
					continue
				}
				x, y := cx+dx, cy+dy
				if x >= 0 && x < pixW && y >= 0 && y < pixH {
					img.Set(x, y, col)
				}
			}
		}
	}

	sameCell := ownX == partnerX && ownY == partnerY
	if sameCell {
		drawDot(ownX, ownY, color.RGBA{R: 255, G: 255, B: 0, A: 255}) // yellow — merged
	} else {
		drawDot(ownX, ownY, color.RGBA{R: 0, G: 255, B: 0, A: 255})           // green — own
		drawDot(partnerX, partnerY, color.RGBA{R: 255, G: 50, B: 50, A: 255}) // red — partner
	}
}

// clampPixel clamps (x, y) to [0, w) and [0, h).
func clampPixel(x, y, w, h int) (int, int) {
	if x < 0 {
		x = 0
	}
	if x >= w {
		x = w - 1
	}
	if y < 0 {
		y = 0
	}
	if y >= h {
		y = h - 1
	}
	return x, y
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
