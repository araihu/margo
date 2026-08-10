package main

import (
	"bytes"
	"encoding/base64"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	margo "github.com/araihu/margo"
	nethtml "golang.org/x/net/html"
)

func materializeLocalImages(document []byte, inputName, workingDirectory string) ([]byte, error) {
	root, err := resourceRoot(inputName, workingDirectory)
	if err != nil {
		return nil, err
	}
	node, err := nethtml.Parse(bytes.NewReader(document))
	if err != nil {
		return nil, fmt.Errorf("cli.html_parse: %w", err)
	}
	var total int64
	var walk func(*nethtml.Node) error
	walk = func(current *nethtml.Node) error {
		if current.Type == nethtml.ElementNode && (current.Data == "img" || current.Data == "source") {
			for index := range current.Attr {
				attribute := &current.Attr[index]
				switch attribute.Key {
				case "src":
					materialized, size, err := materializeImageURL(attribute.Val, root)
					if err != nil {
						return err
					}
					attribute.Val = materialized
					total += size
				case "srcset":
					if strings.HasPrefix(strings.ToLower(strings.TrimSpace(attribute.Val)), "data:") {
						continue
					}
					materialized, size, err := materializeSourceSet(attribute.Val, root)
					if err != nil {
						return err
					}
					attribute.Val = materialized
					total += size
				}
				if err := margo.ValidateResourceSize(total, margo.ResourceLimits{DocumentBytes: margo.MaxDocumentBytes}); err != nil {
					return fmt.Errorf("cli.resource_too_large: %w", err)
				}
			}
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	if err := walk(node); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := nethtml.Render(&output, node); err != nil {
		return nil, fmt.Errorf("cli.html_serialize: %w", err)
	}
	return output.Bytes(), nil
}

func resourceRoot(inputName, workingDirectory string) (string, error) {
	root := workingDirectory
	if inputName != "" && inputName != "<stdin>" {
		absolute, err := filepath.Abs(inputName)
		if err != nil {
			return "", fmt.Errorf("cli.resource_root_invalid: %w", err)
		}
		root = filepath.Dir(absolute)
	}
	if root == "" {
		return "", fmt.Errorf("cli.resource_root_invalid: working directory is unavailable")
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("cli.resource_root_invalid: %w", err)
	}
	return absolute, nil
}

func materializeImageURL(value, root string) (string, int64, error) {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme != "" || parsed.Host != "" || strings.HasPrefix(value, "//") || filepath.IsAbs(parsed.Path) {
		if err == nil && strings.EqualFold(parsed.Scheme, "data") {
			return value, 0, nil
		}
		return "", 0, fmt.Errorf("cli.resource_external: image source %q is not a local relative path", value)
	}
	if parsed.Path == "" {
		return "", 0, fmt.Errorf("cli.resource_external: image source is empty")
	}
	target := filepath.Clean(filepath.Join(root, filepath.FromSlash(parsed.Path)))
	relative, err := filepath.Rel(root, target)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", 0, fmt.Errorf("cli.resource_external: image source escapes its input root")
	}
	realRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", 0, fmt.Errorf("cli.resource_read: %w", err)
	}
	realTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", 0, fmt.Errorf("cli.resource_read: %w", err)
	}
	realRelative, err := filepath.Rel(realRoot, realTarget)
	if err != nil || realRelative == ".." || strings.HasPrefix(realRelative, ".."+string(filepath.Separator)) {
		return "", 0, fmt.Errorf("cli.resource_external: image symlink escapes its input root")
	}
	data, err := os.ReadFile(realTarget)
	if err != nil {
		return "", 0, fmt.Errorf("cli.resource_read: %w", err)
	}
	mediaType, err := imageMediaType(data)
	if err != nil {
		return "", 0, err
	}
	return "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data), int64(len(data)), nil
}

func materializeSourceSet(value, root string) (string, int64, error) {
	parts := strings.Split(value, ",")
	var total int64
	for index, part := range parts {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) == 0 || len(fields) > 2 {
			return "", 0, fmt.Errorf("cli.resource_external: srcset is malformed")
		}
		materialized, size, err := materializeImageURL(fields[0], root)
		if err != nil {
			return "", 0, err
		}
		fields[0] = materialized
		parts[index] = strings.Join(fields, " ")
		total += size
	}
	return strings.Join(parts, ", "), total, nil
}

func imageMediaType(data []byte) (string, error) {
	mediaType := strings.Split(http.DetectContentType(data), ";")[0]
	switch mediaType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return mediaType, nil
	}
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("<svg")) {
		if err := validateStaticSVG(data); err != nil {
			return "", err
		}
		return "image/svg+xml", nil
	}
	return "", fmt.Errorf("cli.resource_format_unsupported: detected %s", mediaType)
}

func validateStaticSVG(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	for {
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return fmt.Errorf("cli.resource_svg_invalid: %w", err)
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		name := strings.ToLower(start.Name.Local)
		switch name {
		case "script", "foreignobject", "iframe", "object", "embed":
			return fmt.Errorf("cli.resource_svg_active: element %s is forbidden", name)
		}
		for _, attribute := range start.Attr {
			attributeName := strings.ToLower(attribute.Name.Local)
			value := strings.TrimSpace(attribute.Value)
			if strings.HasPrefix(attributeName, "on") || ((attributeName == "href" || attributeName == "src") && value != "" && !strings.HasPrefix(value, "#") && !strings.HasPrefix(strings.ToLower(value), "data:")) {
				return fmt.Errorf("cli.resource_svg_active: active attribute %s is forbidden", attributeName)
			}
			lowerValue := strings.ToLower(value)
			if attributeName == "style" && (strings.Contains(lowerValue, "url(http:") || strings.Contains(lowerValue, "url(https:")) {
				return fmt.Errorf("cli.resource_svg_active: external style URL is forbidden")
			}
		}
	}
}
