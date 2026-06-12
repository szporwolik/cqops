package tui

import (
	"bytes"
	"image"
	"image/color"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/eliukblau/pixterm/pkg/ansimage"
)

var httpClient = &http.Client{Timeout: 15 * time.Second}

func downloadImage(url string) (image.Image, error) {
	if url == "" {
		return nil, nil
	}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "CQOps/1.0")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	img, _, decodeErr := image.Decode(bytes.NewReader(data))
	if decodeErr != nil {
		return nil, nil
	}
	return img, nil
}

func downloadAndRenderASCII(url string, maxWidth, maxHeight int) (string, error) {
	img, err := downloadImage(url)
	if err != nil {
		return "", err
	}
	if img == nil {
		return "", nil
	}
	pix, err := ansimage.NewScaledFromImage(img, maxHeight, maxWidth, color.RGBA{30, 30, 30, 255}, ansimage.ScaleModeFit, ansimage.NoDithering)
	if err != nil {
		return "", err
	}
	rendered := pix.Render()
	lines := strings.Split(strings.TrimRight(rendered, "\n"), "\n")
	longest := 0
	for _, l := range lines {
		w := ansiWidth(l)
		if w > longest {
			longest = w
		}
	}
	pad := (maxWidth - longest) / 2
	if pad < 0 {
		pad = 0
	}
	var out strings.Builder
	for _, l := range lines {
		out.WriteString(strings.Repeat(" ", pad))
		out.WriteString(l)
		out.WriteByte('\n')
	}
	return out.String(), nil
}

func ansiWidth(s string) int {
	w := 0
	inEscape := false
	for i := 0; i < len(s); i++ {
		if inEscape {
			if s[i] == 'm' {
				inEscape = false
			}
			continue
		}
		if s[i] == '\x1b' && i+1 < len(s) && s[i+1] == '[' {
			inEscape = true
			i++
			continue
		}
		w++
	}
	return w
}
