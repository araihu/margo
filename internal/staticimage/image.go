package staticimage

import (
	"bytes"
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
	mediaType := strings.Split(http.DetectContentType(data), ";")[0]
	switch mediaType {
	case "image/png", "image/jpeg", "image/gif", "image/webp":
		return mediaType, nil
	}
	trimmed := bytes.TrimSpace(data)
	if mediaType == "text/xml" || bytes.HasPrefix(trimmed, []byte("<svg")) || bytes.HasPrefix(trimmed, []byte("<?xml")) {
		if err := validateSVG(data); err != nil {
			return "", err
		}
		return "image/svg+xml", nil
	}
	return "", &Error{Kind: FormatUnsupported, Message: fmt.Sprintf("detected %s", mediaType)}
}

func DecodeDataURL(value string, limit int64) ([]byte, error) {
	header, payload, found := strings.Cut(value, ",")
	if !found || !strings.HasPrefix(strings.ToLower(header), "data:") {
		return nil, &Error{Kind: FormatUnsupported, Message: "malformed data URL"}
	}
	var data []byte
	var err error
	if strings.HasSuffix(strings.ToLower(header), ";base64") {
		data, err = base64.StdEncoding.DecodeString(payload)
	} else {
		var decoded string
		decoded, err = url.PathUnescape(payload)
		data = []byte(decoded)
	}
	if err != nil {
		return nil, &Error{Kind: FormatUnsupported, Message: fmt.Sprintf("malformed data URL payload: %v", err)}
	}
	if limit < 0 || int64(len(data)) > limit {
		return nil, ErrDataTooLarge
	}
	return data, nil
}

func validateSVG(data []byte) error {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	rootSeen := false
	for {
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
		for _, attribute := range start.Attr {
			attributeName := strings.ToLower(attribute.Name.Local)
			value := strings.TrimSpace(attribute.Value)
			lowerValue := strings.ToLower(value)
			if strings.HasPrefix(attributeName, "on") || ((attributeName == "href" || attributeName == "src") && value != "" && !strings.HasPrefix(value, "#") && !strings.HasPrefix(lowerValue, "data:")) {
				return &Error{Kind: SVGActive, Message: fmt.Sprintf("active attribute %s is forbidden", attributeName)}
			}
			if attributeName == "style" && (strings.Contains(lowerValue, "url(http:") || strings.Contains(lowerValue, "url(https:")) {
				return &Error{Kind: SVGActive, Message: "external style URL is forbidden"}
			}
		}
	}
}
