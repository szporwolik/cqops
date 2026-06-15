// Package assets holds resources embedded into the CQOPS binary.
package assets

import _ "embed"

// WorldMap is the embedded Natural Earth world map raster (equirectangular).
// Compiled into the binary — no external file dependency.
//
// World map raster derived from Natural Earth public domain data.
//
//go:embed map-earth.jpg
var WorldMap []byte
