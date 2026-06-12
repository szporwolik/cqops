package tui

// downloadAndRenderASCII returns a simple text placeholder instead of
// heavy ANSI image rendering. The previous pixterm-based approach pulled
// in 5+ indirect dependencies. Full image rendering can be reintroduced
// later via an optional build tag if needed.
func downloadAndRenderASCII(url string, maxWidth, maxHeight int) (string, error) {
	if url == "" {
		return "", nil
	}
	return DimStyle.Render("📷 Photo available — open " + url), nil
}
