// Package assets holds resources embedded into the CQOps binary.
package assets

import _ "embed"

// WorldMap is the embedded Natural Earth world map raster (equirectangular).
// Compiled into the binary — no external file dependency.
//
// World map raster derived from Natural Earth public domain data.
//
//go:embed map-earth.jpg
var WorldMap []byte

// WorldMapMercator is the same world map reprojected to Web Mercator.
// Used by the Leaflet dashboard fallback maps (Mercator CRS), where the
// equirectangular raster would misplace latitudes.
//
//go:embed map-earth-3857.jpg
var WorldMapMercator []byte

// Logo is the embedded CQOps logo (PNG, 256×256).
// Served via /logo.png on the HTTP dashboard when no custom logo is set.
//
//go:embed other/gh-logo.png
var Logo []byte
