package deck

import (
	"embed"
	"encoding/base64"
	"fmt"
	"io/fs"
	"path"
	"strings"
)

var (
	//go:embed assets/deck.css
	deckCSS string

	//go:embed assets/deck.js
	deckJavaScript string

	//go:embed assets/fonts/*.woff2
	deckFontAssets embed.FS
)

func embeddedDeckFont(file string) ([]byte, error) {
	data, err := fs.ReadFile(deckFontAssets, path.Join("assets", "fonts", file))
	if err != nil {
		return nil, fmt.Errorf("deck.fonts_unavailable: %s: %w", file, err)
	}
	if len(data) < 4 || string(data[:4]) != "wOF2" {
		return nil, fmt.Errorf("deck.fonts_unavailable: %s is not a WOFF2 asset", file)
	}
	return append([]byte(nil), data...), nil
}

func embeddedDeckFontCSS() (string, error) {
	type face struct {
		family string
		file   string
		weight int
	}
	faces := []face{
		{family: "Margo Sans", file: "margo-sans.woff2", weight: 400},
		{family: "Margo Sans", file: "margo-sans.woff2", weight: 600},
		{family: "Margo Sans", file: "margo-sans.woff2", weight: 700},
		{family: "Margo Sans", file: "margo-sans.woff2", weight: 800},
		{family: "Margo Serif", file: "margo-serif.woff2", weight: 400},
		{family: "Margo Serif", file: "margo-serif.woff2", weight: 600},
		{family: "Margo Serif", file: "margo-serif.woff2", weight: 700},
		{family: "Margo Mono", file: "margo-mono.woff2", weight: 400},
		{family: "Margo Mono", file: "margo-mono.woff2", weight: 600},
	}
	var output strings.Builder
	for _, item := range faces {
		data, err := embeddedDeckFont(item.file)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(&output, "@font-face{font-family:'%s';font-style:normal;font-display:block;font-weight:%d;src:url(data:font/woff2;base64,%s) format('woff2');}", item.family, item.weight, base64.StdEncoding.EncodeToString(data))
	}
	return output.String(), nil
}
