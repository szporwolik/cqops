package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/ftl/hamradio/locator"
)

const worldMap = "" +
	"                                                                                                                        \n" +
	"                                                                                                                        \n" +
	"                                                                                                                        \n" +
	"                   ##                                                                                                   \n" +
	"                  ####                    ##                                                              ##            \n" +
	"                 ######                 ####                                                           #####            \n" +
	"               ##########            ######                    ##                                      #######          \n" +
	"             #############        ########                  #####                                   ##########         \n" +
	"           ###############     ###########                #######                   ##             ############        \n" +
	"         ################   ##############             ##########                 ####           ##############        \n" +
	"       ################   ################          #############               ######        #################       \n" +
	"      ###############   ###################       ###############             ########     ####################       \n" +
	"     ###############   ####################     ##################          ##########    #####################       \n" +
	"    ##############    ######################   ####################       ###########  ########################       \n" +
	"   #############     ########  #############  ######   ############    ############  ########   ###############      \n" +
	"  #############     ######     ############## #####     ###########  #############  ######       ##############      \n" +
	" #############     #####       ############## ###       ############ ############   ####          #############      \n" +
	" ############     #####        ############## #         ############ ###########    ##            ############      \n" +
	" ############    #####         ##############          ############# ##########     #              ###########      \n" +
	" ############    #####         ##############         ############## ########      #               ###########      \n" +
	" ############    ######        ##############       ############### ########    ##                ###########       \n" +
	" #############   ########      ##############     #################  ######   ###                #############      \n" +
	"  #############  ################################ ################   ########                  ##############       \n" +
	"  ###########################  ###################################     ######                  ##############        \n" +
	"    #######################    #################################       ##                    ############           \n" +
	"     #####################      ###############################                             #############            \n" +
	"       ##################        #############################                            ############              \n" +
	"         ##############            #########################                            ###########                 \n" +
	"            ##########               #####################                            ##########                    \n" +
	"               #####                    ################                             ########                       \n" +
	"                ##                        ###########                              #######                          \n" +
	"                                            #######                                  ###                            \n" +
	"                                              ##                                                                      \n" +
	"                                                                                                                        "

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
	scaleX := float64(width) / float64(mapW)
	scaleY := float64(height) / float64(mapH)
	scale := scaleX
	if scaleY < scaleX {
		scale = scaleY
	}
	outW := int(float64(mapW) * scale)
	outH := int(float64(mapH) * scale)
	if outW < 40 {
		outW = 40
	}
	if outH < 8 {
		outH = 8
	}

	ownX, ownY := latLonToMap(ownLat, ownLon, outW, outH)
	partnerX, partnerY := latLonToMap(partnerLat, partnerLon, outW, outH)

	landSty := lipgloss.NewStyle().Foreground(lipgloss.Color("70"))
	seaSty := lipgloss.NewStyle().Foreground(lipgloss.Color("19"))
	ownLabel := inputStyle.Render("[*]")
	partnerLabel := lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("[P]")

	var out strings.Builder
	for y := 0; y < outH; y++ {
		srcY := int(float64(y) / scale)
		if srcY >= mapH {
			srcY = mapH - 1
		}
		srcLine := mapLines[srcY]
		var rowRunes []rune
		for x := 0; x < outW; x++ {
			srcX := int(float64(x) / scale)
			if srcX < len(srcLine) {
				rowRunes = append(rowRunes, rune(srcLine[srcX]))
			} else {
				rowRunes = append(rowRunes, ' ')
			}
		}
		type mark struct{ x int; label string }
		var marks []mark
		if ownX >= 0 && ownX < outW && y == ownY {
			marks = append(marks, mark{ownX, ownLabel})
		}
		if partnerX >= 0 && partnerX < outW && y == partnerY {
			marks = append(marks, mark{partnerX, partnerLabel})
		}
		if len(marks) == 2 && ownX == partnerX && ownY == partnerY {
			marks = []mark{{ownX, inputStyle.Render("[B]")}}
		}
		for _, ch := range rowRunes {
			if ch == '#' {
				out.WriteString(landSty.Render("#"))
			} else {
				out.WriteString(seaSty.Render("·"))
			}
		}
		for _, m := range marks {
			out.WriteString(" " + m.label)
		}
		out.WriteByte('\n')
	}
	return out.String()
}

func latLonToMap(lat, lon float64, w, h int) (int, int) {
	x := int((lon + 180.0) / 360.0 * float64(w-1))
	y := int((90.0 - lat) / 180.0 * float64(h-1))
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
