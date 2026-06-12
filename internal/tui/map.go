package tui

import (
	"fmt"
	"strings"

	"github.com/ftl/hamradio/locator"
)

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
	if outW < 20 {
		outW = 20
	}
	if outH < 5 {
		outH = 5
	}

	ownX, ownY := latLonToMap(ownLat, ownLon, outW, outH)
	partnerX, partnerY := latLonToMap(partnerLat, partnerLon, outW, outH)

	ownLabel := inputStyle.Render("[*]")
	partnerLabel := inputStyle.Render("[P]")

	var out strings.Builder
	for y := 0; y < outH; y++ {
		srcY := int(float64(y) / scale)
		if srcY >= mapH {
			srcY = mapH - 1
		}
		srcLine := mapLines[srcY]
		row := make([]byte, outW)
		for x := 0; x < outW; x++ {
			srcX := int(float64(x) / scale)
			if srcX < len(srcLine) {
				row[x] = srcLine[srcX]
			} else {
				row[x] = ' '
			}
		}
		out.Write(row)

		marker := ""
		if y == ownY && ownX >= 0 && ownX < outW {
			marker = ownLabel
		}
		if y == partnerY && partnerX >= 0 && partnerX < outW {
			if marker != "" && ownX == partnerX {
				marker = inputStyle.Render("[B]")
			} else {
				marker = partnerLabel
			}
		}
		if marker != "" {
			out.WriteString(" " + marker)
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

const worldMap = "" +
	"                                ..::::..                                          \n" +
	"                            .::^^~~~~^^^::..                                     \n" +
	"                        ..::^^~~~~~~~~~^^^^^^::..              ....              \n" +
	"                      .::^~~^^^^^^^^~~~~~~~^^^^^^^^^::..    .::^^::..            \n" +
	"                    .::^~~~~~~~~~~~~~~^^^^^^^^^^^^^^^^^^^::..::^^::..           \n" +
	"                   .^~~~^^^~~~~^^^^^^^~~~~~~~~~~~~~~~~~~^^^^^^^^^^::..          \n" +
	"                 .:^~~~~~^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^~~~~~~~~~~~~^^::.       \n" +
	"                .^~^^^^^^^^^^^^^^^^^^::::::::::^^^^^^^^^^^^^~~~~~~~~~^^:.      \n" +
	"               .^~^^^^^^^^^^^^^^^:::....       ..:^^^^^^^^^^^^^^^~~~~~^^:.      \n" +
	"               ^~^^^^^^^^^^^^:::..                ..::^^^^^^^^^^^~~~~~~^:.     \n" +
	"              :~^^^^^^^^^^:::...       .....        ..::^^^^^^^^^^~~~~~^:.    \n" +
	"             .^^^^^^^^^^:::...       .:^^^^::..       ..:^^^^^^^^^~~~~~~^.    \n" +
	"             :^^^^^^^^^:::...        :^^^^^^^^::.       .:^^^^^^^^^~~~~~~^.   \n" +
	"            .^^^^^^^^^^::..         :^^^^^^^^^^::        .:^^^^^^^^^~~~~~~^.  \n" +
	"            :^^^^^^^^^^::..        .^^^^^^^^^^^^:.       .:^^^^^^^^^^~~~~~~^.\n" +
	"           .^^^^^^^^^^^^::..       .^^^^^^^^^^^^:.       .:^^^^^^^^^^^~~~~~~:\n" +
	"            ^^^^^^^^^^^^^::..      .:^^^^^^^^^^::.       .:^^^^^^^^^^^^~~~~~~.\n" +
	"            :^^^^^^^^^^^^^::..      .::^^^^^^::..        .:^^^^^^^^^^^^^~~~~~.\n" +
	"            .^^^^^^^^^^^^^^::...     ..........        ..::^^^^^^^^^^^^^^~~~~:\n" +
	"             :^^^^^^^^^^^^^^:::...                    .::^^^^^^^^^^^^^^^^^^^~^\n" +
	"              ^^^^^^^^^^^^^^^^^::....              ...::^^^^^^^^^^^^^^^^^^^^^^\n" +
	"              .^^^^^^^^^^^^^^^^^^^:::..............:::^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"               .^^^^^^^^^^^^^^^^^^^^^^^^::::::::::^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"                .^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"                 .:^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"                   .:^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"                     .::^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"                        .::^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"                           ...::^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"                                .......::^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"                                   .........:::::^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n" +
	"                                         ...........::::::::::^^^^^^^^^^^^^^\n" +
	"                                              ....................:::::::::::"
