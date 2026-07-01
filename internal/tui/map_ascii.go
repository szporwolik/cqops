package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/ftl/hamradio/locator"
)

const worldMap = "" +
	"          . _..::__:  ,-\"-\"._       |]       ,     _,.__              \n" +
	"  _.___ _ _<_>`!(._`.`-.    /        _._     `_ ,_/  '  '-._.---.-.__ \n" +
	".{     \" \" `-==,',._\\{  \\  / {)     / _ \">_,-' `                 /-/_ \n" +
	" \\_.:--.       `._ )`^-. \"'      , [_/(                       __,/-'  \n" +
	"'\"'     \\         \"    _L       |-_,--'                )     /. (|    \n" +
	"         |           ,'         _)_.\\\\._<> {}              _,' /  '   \n" +
	"         `.         /          [_/_'` `\"(                <'}  )       \n" +
	"          \\\\    .-. )          /   `-'\"..' `:._          _)  '        \n" +
	"   `        \\  (  `(          /         `:\\  > \\  ,-^.  /' '          \n" +
	"             `._,   \"\"        |           \\`'   \\|   ?_)  {\\          \n" +
	"                `=.---.       `._._       ,'     \"`  |' ,- '.         \n" +
	"                  |    `-._        |     /          `:`<_|=--._       \n" +
	"                  (        >       .     | ,          `=.__.`-'\\      \n" +
	"                   `.     /        |     |{|              ,-.,\\     . \n" +
	"                    |   ,'          \\   / `'            ,\"     \\      \n" +
	"                    |  /             |_'                |  __  /      \n" +
	"                    | |                                 '-'  `-'   \\. \n" +
	"                    |/                                        \"    /  \n" +
	"                    \\.                                            '   \n" +
	"                                                                      \n" +
	"                     ,/           ______._.--._ _..---.---------.     \n" +
	"__,-----\"-..?----_/ )\\    . ,-'\"             \"                  (__--/\n" +
	"                      /__/\\/                                        \n"

// NativeMapHeight is the full height of the ASCII world map in rows.
// NativeMapWidth is its full width in columns.
var NativeMapHeight, NativeMapWidth int

func init() {
	lines := strings.Split(strings.TrimRight(worldMap, "\n"), "\n")
	NativeMapHeight = len(lines)
	for _, l := range lines {
		if len(l) > NativeMapWidth {
			NativeMapWidth = len(l)
		}
	}
}

func renderWorldMap(ownLat, ownLon, partnerLat, partnerLon float64, width, height int) string {
	mapLines := strings.Split(strings.TrimRight(worldMap, "\n"), "\n")
	mapH := len(mapLines)
	mapW := 0
	for _, l := range mapLines {
		if len(l) > mapW {
			mapW = len(l)
		}
	}
	if mapH == 0 || mapW == 0 {
		return ""
	}

	outW := mapW
	outH := mapH
	if height < outH {
		return "" // height is non-negotiable — the art can't be cropped vertically
	}

	ownX, ownY := mercatorXY(ownLat, ownLon, outW, outH)
	partnerX, partnerY := mercatorXY(partnerLat, partnerLon, outW, outH)

	ownSty := S.MapOwn
	partnerSty := S.MapPartner
	bothSty := S.MapBoth

	pad := (width - outW) / 2
	if pad < 0 {
		pad = 0
	}
	padStr := strings.Repeat(" ", pad)

	hasOwn := ownY >= 0 && ownY < outH && ownX >= 0 && ownX < outW
	hasPartner := partnerY >= 0 && partnerY < outH && partnerX >= 0 && partnerX < outW
	collide := hasOwn && hasPartner && ownX == partnerX && ownY == partnerY

	var b strings.Builder
	for y := 0; y < outH; y++ {
		b.WriteString(padStr)
		line := mapLines[y]
		for x := 0; x < outW; x++ {
			ownHere := hasOwn && y == ownY && x == ownX
			partnerHere := hasPartner && y == partnerY && x == partnerX

			if ownHere && partnerHere {
				b.WriteString(bothSty.Render("@"))
			} else if ownHere {
				b.WriteString(ownSty.Render("*"))
			} else if partnerHere {
				b.WriteString(partnerSty.Render("P"))
			} else if x < len(line) {
				b.WriteByte(line[x])
			} else {
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	if hasOwn || hasPartner {
		b.WriteString(padStr)
		if collide {
			b.WriteString(bothSty.Render("@"))
			b.WriteString(S.MapGrid.Render("  you & partner"))
		} else {
			b.WriteString(ownSty.Render("*"))
			b.WriteString(S.MapGrid.Render("  you    "))
			b.WriteString(partnerSty.Render("P"))
			b.WriteString(S.MapGrid.Render("  partner"))
		}
	}
	return b.String()
}

func mercatorXY(lat, lon float64, w, h int) (int, int) {
	if lat == 0 && lon == 0 {
		return -1, -1
	}
	x := int((lon+180.0)/360.0*float64(w) + 0.5)
	rLat := lat * math.Pi / 180.0
	yMerc := math.Log(math.Tan(math.Pi/4.0 + rLat/2.0))
	yMin, yMax := -0.6, 1.4
	yRatio := (yMerc - yMin) / (yMax - yMin)
	y := int((1.0-yRatio)*float64(h) + 0.5)
	if x < 0 {
		x = 0
	} else if x >= w {
		x = w - 1
	}
	if y < 0 {
		y = 0
	} else if y >= h {
		y = h - 1
	}
	return x, y
}

func gridToLatLon(grid string) (float64, float64) {
	// ftl/hamradio/locator supports up to 8-char grids (4 pairs).
	// Truncate longer grids to 8 chars — the 5th pair adds ~5m precision,
	// which is below the map resolution anyway.
	if len(grid) > 8 {
		grid = grid[:8]
	}
	loc, err := locator.Parse(grid)
	if err != nil {
		return 0, 0
	}
	ll := locator.ToLatLon(loc)
	lat, lon := float64(ll.Lat), float64(ll.Lon)
	// locator.ToLatLon has a bug for 8-char grids: when the 4th pair
	// is present the code takes the "if" branch for precision 2, adds
	// the digit offset, but falls through without adding the centre
	// offset for either the 3rd pair (subsquare: 2.5′ lon × 1.25′ lat)
	// or the 4th pair (extended: 0.25′ lon × 0.125′ lat). The result
	// is the SW corner of the cell instead of the centre.
	// Add both missing centre offsets back.
	if loc[6] > 0 {
		lon += (2.5 + 0.25) / 60.0 // 3rd-pair centre + 4th-pair centre
		lat += (1.25 + 0.125) / 60.0
	}
	return lat, lon
}

func parseCoord(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}
