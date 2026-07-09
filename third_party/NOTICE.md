# Third-Party Notices

This document lists third-party components used by CQOps that are not covered by the CQOps project license (Apache 2.0). Each component retains its own upstream license.

| Component | Used for | License / Status | Local notice | Upstream source |
|---|---|---|---|---|
| **hessu/aprs-symbols** | APRS symbol sprite graphics (3 PNG sheets embedded for dashboard map markers) | Free for APRS applications per upstream README; mixed per-symbol copyright (VEC-OH7LZB / OH7LZB CC BY-SA 2.0 / PD / brand logos); attribution/source pointer requested | [`aprs-symbols/NOTICE.md`](aprs-symbols/NOTICE.md) | https://github.com/hessu/aprs-symbols |
| **hessu/go-aprs-fap** (algorithm reference) | APRS Mic-E and Base-91 position decoding algorithms (implemented in `internal/aprs/parse.go`; not a linked dependency) | BSD-style (upstream LICENSE) | — | https://github.com/la5nta/go-aprs |
| **farmergreg/adif** | ADIF 3.1.7 parsing and writing | MIT | `../licenses/ADIF-MIT-LICENSE` | https://github.com/farmergreg/adif |
| **farmergreg/spec** | ADIF specification types | BSD-3-Clause | `../licenses/SPEC-BSD3-LICENSE` | https://github.com/farmergreg/spec |
| **ftl/hamradio** | Grid locator, distance math, DXCC prefix lookup | MIT | `../licenses/HAMRADIO-MIT-LICENSE` | https://github.com/ftl/hamradio |
| **k0swe/wsjtx-go** | WSJT-X UDP protocol | MIT | `../licenses/WSJTXGO-MIT-LICENSE` | https://github.com/k0swe/wsjtx-go |
| **Charm stack** (Bubble Tea v2, Bubbles, Lip Gloss) | TUI framework | MIT | `../licenses/BUBBLETEA-MIT-LICENSE`, `../licenses/BUBBLES-MIT-LICENSE`, `../licenses/LIPGLOSS-MIT-LICENSE` | https://charm.land |
| **modernc.org/sqlite** | Pure-Go SQLite database | BSD-3-Clause | `../licenses/SQLITE-BSD3-LICENSE` | https://modernc.org/sqlite |
| **spf13/cobra** | CLI framework | Apache 2.0 | `../licenses/COBRA-APACHE2-LICENSE.txt` | https://github.com/spf13/cobra |
| **gopkg.in/yaml.v3** | Configuration parsing | MIT | `../licenses/YAML-MIT-LICENSE` | https://github.com/go-yaml/yaml |
| **golang.org/x/text** | Unicode normalization for ADIF | BSD-3-Clause | `../licenses/TEXT-BSD3-LICENSE` | https://golang.org/x/text |
| **NimbleMarkets/ntcharts** | Terminal chart rendering | MIT | `../licenses/NTCHARTS-MIT-LICENSE.txt` | https://github.com/NimbleMarkets/ntcharts |
| **gen2brain/beeep** | Desktop notifications | MIT | `../licenses/BEEEP-MIT-LICENSE` | https://github.com/gen2brain/beeep |
| **go.bug.st/serial** | Serial port (GPS NMEA, APRS KISS TNC) | BSD-3-Clause | `../licenses/SERIAL-BSD3-LICENSE` | https://github.com/bugst/go-serial |
| **Leaflet 1.9.4** | Dashboard interactive maps (bundled) | BSD-2-Clause | `../licenses/LEAFLET-BSD2-LICENSE` | https://leafletjs.com |
| **Leaflet.Terminator** | Day/night grayline overlay (bundled) | MIT | `../licenses/LEAFLET-TERMINATOR-MIT-LICENSE` | https://github.com/joergdietrich/Leaflet.Terminator |
| **Open-Meteo** | Weather forecast API (browser-side) | CC BY 4.0 | `../licenses/OPEN-METEO-CC-BY-4.0.txt` | https://open-meteo.com/ |
| **OpenStreetMap tiles** | Dashboard map tiles (optional, browser-side) | ODbL, © OpenStreetMap contributors | — | https://www.openstreetmap.org/copyright |
| **RainViewer** | Weather radar overlay (optional, browser-side) | Free public API; attribution displayed on-map | — | https://www.rainviewer.com |

## APRS Symbol Graphics — Important Note

The APRS symbol graphics embedded in CQOps (`aprs-symbols-24-{0,1,2}.png`) are from the **aprs.fi APRS symbol set** by Heikki Hannikainen, OH7LZB. These graphics have complex, per-symbol copyright status as detailed in [`aprs-symbols/COPYRIGHT.md`](aprs-symbols/COPYRIGHT.md). They are **not** covered by the CQOps Apache 2.0 license.

The APRS position-parsing code in `internal/aprs/parse.go` is an independent implementation based on algorithms described in the go-aprs-fap library and the APRS 1.0.1/1.2 specifications. It is not a copy or fork of go-aprs-fap source code.

**None of the third-party components listed above are relicensed under the CQOps project license.** Each retains its own upstream license terms.
