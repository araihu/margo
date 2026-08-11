package staticimage

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type ErrorKind string

const (
	FormatUnsupported ErrorKind = "format_unsupported"
	SVGInvalid        ErrorKind = "svg_invalid"
	SVGActive         ErrorKind = "svg_active"
)

type Error struct {
	Kind    ErrorKind
	Message string
}

func (err *Error) Error() string {
	if err == nil {
		return "static image validation failed"
	}
	return err.Message
}

var ErrDataTooLarge = errors.New("data image exceeds its byte limit")

func Detect(data []byte) (string, error) {
	return DetectContext(context.Background(), data)
}

func DetectContext(ctx context.Context, data []byte) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	mediaType := strings.Split(http.DetectContentType(data), ";")[0]
	switch mediaType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return mediaType, nil
	}
	trimmed := bytes.TrimSpace(data)
	if mediaType == "text/xml" || bytes.HasPrefix(trimmed, []byte("<svg")) || bytes.HasPrefix(trimmed, []byte("<?xml")) {
		if err := validateSVG(ctx, data); err != nil {
			return "", err
		}
		return "image/svg+xml", nil
	}
	return "", &Error{Kind: FormatUnsupported, Message: fmt.Sprintf("detected %s", mediaType)}
}

func ValidateDataURL(ctx context.Context, value string, limit int64) ([]byte, string, error) {
	data, declared, err := decodeDataURL(value, limit)
	if err != nil {
		return nil, "", err
	}
	detected, err := DetectContext(ctx, data)
	if err != nil {
		return nil, "", err
	}
	if declared != detected {
		return nil, "", &Error{Kind: FormatUnsupported, Message: fmt.Sprintf("declared media type %q does not match detected %q", declared, detected)}
	}
	return data, detected, nil
}

func DecodeDataURL(value string, limit int64) ([]byte, error) {
	data, _, err := decodeDataURL(value, limit)
	return data, err
}

func decodeDataURL(value string, limit int64) ([]byte, string, error) {
	header, payload, found := strings.Cut(value, ",")
	if !found || !strings.HasPrefix(strings.ToLower(header), "data:") {
		return nil, "", &Error{Kind: FormatUnsupported, Message: "malformed data URL"}
	}
	parameters := strings.Split(strings.TrimPrefix(strings.ToLower(header), "data:"), ";")
	declared := strings.TrimSpace(parameters[0])
	var data []byte
	var err error
	base64Encoded := false
	for _, parameter := range parameters[1:] {
		if strings.TrimSpace(parameter) == "base64" {
			base64Encoded = true
		}
	}
	if base64Encoded {
		data, err = base64.StdEncoding.DecodeString(payload)
	} else {
		var decoded string
		decoded, err = url.PathUnescape(payload)
		data = []byte(decoded)
	}
	if err != nil {
		return nil, "", &Error{Kind: FormatUnsupported, Message: fmt.Sprintf("malformed data URL payload: %v", err)}
	}
	if limit < 0 || int64(len(data)) > limit {
		return nil, "", ErrDataTooLarge
	}
	return data, declared, nil
}

func validateSVG(ctx context.Context, data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	rootSeen := false
	styleDepth := 0
	var styleText strings.Builder
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		token, err := decoder.Token()
		if err != nil {
			if err == io.EOF && rootSeen {
				return nil
			}
			if err == io.EOF {
				return &Error{Kind: SVGInvalid, Message: "SVG root element is missing"}
			}
			return &Error{Kind: SVGInvalid, Message: err.Error()}
		}
		switch typed := token.(type) {
		case xml.EndElement:
			if strings.EqualFold(typed.Name.Local, "style") && styleDepth > 0 {
				styleDepth--
				if styleDepth == 0 {
					if err := validateStyleCSS(styleText.String()); err != nil {
						return err
					}
					styleText.Reset()
				}
			}
			continue
		case xml.CharData:
			if styleDepth > 0 {
				styleText.Write(typed)
			}
			continue
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		name := strings.ToLower(start.Name.Local)
		if !rootSeen {
			rootSeen = true
			if name != "svg" {
				return &Error{Kind: FormatUnsupported, Message: fmt.Sprintf("XML root element %s is not svg", name)}
			}
		}
		switch name {
		case "script", "foreignobject", "iframe", "object", "embed":
			return &Error{Kind: SVGActive, Message: fmt.Sprintf("element %s is forbidden", name)}
		}
		if name == "style" {
			if styleDepth == 0 {
				styleText.Reset()
			}
			styleDepth++
		}
		for _, attribute := range start.Attr {
			attributeName := strings.ToLower(attribute.Name.Local)
			value := strings.TrimSpace(attribute.Value)
			lowerValue := strings.ToLower(value)
			if strings.HasPrefix(attributeName, "on") || ((attributeName == "href" || attributeName == "src") && value != "" && !strings.HasPrefix(value, "#") && !strings.HasPrefix(lowerValue, "data:")) {
				return &Error{Kind: SVGActive, Message: fmt.Sprintf("active attribute %s is forbidden", attributeName)}
			}
			if attributeName == "style" {
				if err := validateStyleCSS(value); err != nil {
					return err
				}
			}
		}
	}
}

func validateStyleCSS(value string) error {
	lower := strings.ToLower(value)
	if strings.Contains(value, `\`) || strings.Contains(lower, "@") || strings.Contains(lower, "/*") || strings.Contains(lower, "*/") {
		return &Error{Kind: SVGActive, Message: "external or escaped style syntax is forbidden"}
	}
	lower = strings.Map(func(character rune) rune {
		switch character {
		case ' ', '\t', '\r', '\n', '\f':
			return -1
		default:
			return character
		}
	}, lower)
	for cursor := 0; ; {
		index := strings.Index(lower[cursor:], "url(")
		if index < 0 {
			return nil
		}
		start := cursor + index + len("url(")
		endOffset := strings.IndexByte(lower[start:], ')')
		if endOffset < 0 {
			return &Error{Kind: SVGActive, Message: "malformed style URL is forbidden"}
		}
		end := start + endOffset
		target := strings.Trim(strings.TrimSpace(lower[start:end]), `"'`)
		lowerTarget := strings.ToLower(strings.TrimSpace(target))
		if !strings.HasPrefix(lowerTarget, "#") && !strings.HasPrefix(lowerTarget, "data:") {
			return &Error{Kind: SVGActive, Message: "external style URL is forbidden"}
		}
		cursor = end + 1
	}
}
