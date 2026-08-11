package main

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	margo "github.com/araihu/margo"
	"github.com/araihu/margo/internal/staticimage"
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
			data, decodeErr := staticimage.DecodeDataURL(value, margo.MaxDocumentBytes)
			if decodeErr != nil {
				if errors.Is(decodeErr, staticimage.ErrDataTooLarge) {
					return "", 0, fmt.Errorf("cli.resource_too_large: %w", decodeErr)
				}
				return "", 0, fmt.Errorf("cli.resource_format_unsupported: %w", decodeErr)
			}
			if _, detectErr := imageMediaType(data); detectErr != nil {
				return "", 0, detectErr
			}
			return value, int64(len(data)), nil
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
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(value)), "data:") {
		fields := strings.Fields(strings.TrimSpace(value))
		if len(fields) == 0 || len(fields) > 2 {
			return "", 0, fmt.Errorf("cli.resource_external: srcset is malformed")
		}
		_, size, err := materializeImageURL(fields[0], root)
		if err != nil {
			return "", 0, err
		}
		return value, size, nil
	}
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
	mediaType, err := staticimage.Detect(data)
	if err == nil {
		return mediaType, nil
	}
	var imageErr *staticimage.Error
	if errors.As(err, &imageErr) {
		switch imageErr.Kind {
		case staticimage.SVGInvalid:
			return "", fmt.Errorf("cli.resource_svg_invalid: %s", imageErr.Message)
		case staticimage.SVGActive:
			return "", fmt.Errorf("cli.resource_svg_active: %s", imageErr.Message)
		}
	}
	return "", fmt.Errorf("cli.resource_format_unsupported: %v", err)
}
