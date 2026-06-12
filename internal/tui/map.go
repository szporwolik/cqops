package tui

import (
	"fmt"
	"math"
	"strings"

	"github.com/charmbracelet/lipgloss"
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
	if width < mapW || height < mapH {
		return ""
	}

	outW := mapW
	outH := mapH

	ownX, ownY := mercatorXY(ownLat, ownLon, outW, outH)
	partnerX, partnerY := mercatorXY(partnerLat, partnerLon, outW, outH)

	ownSty := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)
	partnerSty := lipgloss.NewStyle().Foreground(lipgloss.Color("13")).Bold(true)
	bothSty := lipgloss.NewStyle().Foreground(lipgloss.Color("14")).Bold(true)

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
		gray := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
		b.WriteString(padStr)
		if collide {
			b.WriteString(bothSty.Render("@") + gray.Render("  you & partner"))
		} else {
			b.WriteString(ownSty.Render("*") + gray.Render("  you    ") + partnerSty.Render("P") + gray.Render("  partner"))
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
	yMerc := math.Log(math.Tan(math.Pi/4.0+rLat/2.0))
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
	loc, err := locator.Parse(grid)
	if err != nil {
		return 0, 0
	}
	ll := locator.ToLatLon(loc)
	return float64(ll.Lat), float64(ll.Lon)
}

func parseCoord(s string) float64 {
	var f float64
	_, _ = fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}
