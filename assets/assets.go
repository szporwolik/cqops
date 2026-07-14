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

// Logo is the embedded CQOps logo (PNG, 256×256).
// Served via /logo.png on the HTTP dashboard when no custom logo is set.
//
//go:embed other/gh-logo.png
var Logo []byte
