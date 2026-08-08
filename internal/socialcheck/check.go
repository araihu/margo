package socialcheck

import (
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"os"
	"strings"

	"golang.org/x/net/html"
)

// RequireOneCompleteSet parses initial HTML and requires exactly one value for
// each route-level metadata key. It intentionally ignores client-side scripts.
func RequireOneCompleteSet(markup, canonical string) error {
	tokenizer := html.NewTokenizer(strings.NewReader(markup))
	counts := map[string]int{}
	values := map[string]string{}
	for {
		tokenType := tokenizer.Next()
		if tokenType == html.ErrorToken {
			if tokenizer.Err() == io.EOF {
				break
			}
			return fmt.Errorf("socialcheck: invalid HTML: %w", tokenizer.Err())
		}
		if tokenType != html.StartTagToken && tokenType != html.SelfClosingTagToken {
			continue
		}
		token := tokenizer.Token()
		switch token.Data {
		case "title":
			counts["title"]++
		case "link":
			rel := attr(token, "rel")
			if rel == "canonical" {
				counts["canonical"]++
				values["canonical"] = attr(token, "href")
			}
		case "meta":
			name := attr(token, "name")
			property := attr(token, "property")
			key := name
			if key == "" {
				key = property
			}
			if key == "description" || strings.HasPrefix(key, "og:") || strings.HasPrefix(key, "twitter:") {
				counts[key]++
				values[key] = attr(token, "content")
			}
		}
	}
	required := []string{"title", "description", "canonical", "og:url", "og:type", "og:title", "og:description", "og:site_name", "og:image", "og:image:type", "og:image:width", "og:image:height", "og:image:alt", "twitter:card", "twitter:title", "twitter:description", "twitter:image", "twitter:image:alt"}
	for _, key := range required {
		if counts[key] != 1 || strings.TrimSpace(values[key]) == "" && key != "title" {
			return fmt.Errorf("socialcheck: expected one non-empty %s, got %d", key, counts[key])
		}
	}
	if values["canonical"] != canonical || values["og:url"] != canonical {
		return fmt.Errorf("socialcheck: canonical mismatch")
	}
	return nil
}

func attr(token html.Token, key string) string {
	for _, attribute := range token.Attr {
		if attribute.Key == key {
			return attribute.Val
		}
	}
	return ""
}

// RequirePNG validates the exact preview dimensions and byte bound.
func RequirePNG(path string, width, height int, maxBytes int64) error {
	info, err := os.Stat(path)
	if err != nil {
		return fmt.Errorf("socialcheck: preview stat: %w", err)
	}
	if info.Size() <= 0 || info.Size() >= maxBytes {
		return fmt.Errorf("socialcheck: preview size %d outside bound", info.Size())
	}
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	config, format, err := image.DecodeConfig(file)
	if err != nil {
		return fmt.Errorf("socialcheck: preview decode: %w", err)
	}
	if format != "png" || config.Width != width || config.Height != height {
		return fmt.Errorf("socialcheck: preview = %s %dx%d, want png %dx%d", format, config.Width, config.Height, width, height)
	}
	return nil
}
