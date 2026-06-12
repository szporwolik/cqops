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

	landSty := lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	landFillSty := lipgloss.NewStyle().Foreground(lipgloss.Color("22"))
	seaSty := lipgloss.NewStyle().Foreground(lipgloss.Color("19"))
	iceSty := lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
	playerSty := lipgloss.NewStyle().Foreground(th.Accent).Bold(true)
	partnerSty := lipgloss.NewStyle().Foreground(th.Accent).Bold(true)

	pad := (width - outW) / 2
	if pad < 0 {
		pad = 0
	}
	padStr := strings.Repeat(" ", pad)

	var b strings.Builder
	for y := 0; y < outH; y++ {
		srcLine := mapLines[y]
		isPolar := y <= 2 || y >= outH-2

		leftEdge, rightEdge := -1, -1
		for x := 0; x < outW; x++ {
			if x < len(srcLine) && srcLine[x] != ' ' {
				if leftEdge < 0 {
					leftEdge = x
				}
				rightEdge = x
			}
		}

		isPlayerRow := ownY >= 0 && y == ownY
		isPartnerRow := partnerY >= 0 && y == partnerY

		b.WriteString(padStr)
		for x := 0; x < outW; x++ {
			playerHere := isPlayerRow && x == ownX
			partnerHere := isPartnerRow && x == partnerX

			if playerHere && partnerHere {
				b.WriteString(playerSty.Render("*"))
			} else if playerHere {
				b.WriteString(playerSty.Render("*"))
			} else if partnerHere {
				b.WriteString(partnerSty.Render("P"))
			} else if x < len(srcLine) && srcLine[x] != ' ' {
				if isPolar {
					b.WriteString(iceSty.Render(string(srcLine[x])))
				} else {
					b.WriteString(landSty.Render(string(srcLine[x])))
				}
			} else if x >= leftEdge && x <= rightEdge && leftEdge >= 0 {
				if isPolar {
					b.WriteString(iceSty.Render("·"))
				} else {
					b.WriteString(landFillSty.Render("·"))
				}
			} else {
				b.WriteString(seaSty.Render("·"))
			}
		}
		b.WriteByte('\n')
	}
	if (ownY >= 0 && ownY < outH) || (partnerY >= 0 && partnerY < outH) {
		b.WriteString(padStr)
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("245")).Render("  * you    P partner"))
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
	fmt.Sscanf(strings.TrimSpace(s), "%f", &f)
	return f
}
