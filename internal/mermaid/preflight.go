package mermaid

import (
	"fmt"
	"strings"
)

const ConfigurationForbiddenCode = "mermaid.configuration_forbidden"

// DiagnosticError reports one stable Mermaid compilation failure without
// coupling the internal compiler to the root diagnostic package.
type DiagnosticError struct {
	Code    string
	Message string
	Offset  int
}

func (e *DiagnosticError) Error() string {
	if e == nil {
		return "mermaid: diagnostic error"
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Preflight rejects every document-controlled Mermaid configuration channel.
// It runs before any task descriptor is constructed.
func Preflight(source []byte) error {
	if offset, ok := leadingYAMLFrontmatter(source); ok {
		return configurationForbidden("Mermaid fence YAML frontmatter is unsupported", offset)
	}
	if directive, offset, ok := forbiddenDirective(source); ok {
		return configurationForbidden("Mermaid %%{"+directive+":...}%% directives are unsupported", offset)
	}
	return nil
}

func configurationForbidden(message string, offset int) error {
	return &DiagnosticError{Code: ConfigurationForbiddenCode, Message: message, Offset: offset}
}

func leadingYAMLFrontmatter(source []byte) (int, bool) {
	offset := 0
	if len(source) >= 3 && source[0] == 0xef && source[1] == 0xbb && source[2] == 0xbf {
		offset = 3
	}
	for offset < len(source) {
		switch source[offset] {
		case ' ', '\t', '\r', '\n':
			offset++
		default:
			return offset, isDelimiterLine(source[offset:])
		}
	}
	return 0, false
}

func isDelimiterLine(source []byte) bool {
	if len(source) < 3 || source[0] != '-' || source[1] != '-' || source[2] != '-' {
		return false
	}
	for i := 3; i < len(source); i++ {
		switch source[i] {
		case ' ', '\t':
			continue
		case '\r':
			return i+1 == len(source) || source[i+1] == '\n'
		case '\n':
			return true
		default:
			return false
		}
	}
	return true
}

func forbiddenDirective(source []byte) (string, int, bool) {
	for offset := 0; offset+3 <= len(source); offset++ {
		if source[offset] != '%' || source[offset+1] != '%' || source[offset+2] != '{' {
			continue
		}
		cursor := offset + 3
		for cursor < len(source) && isDirectiveSpace(source[cursor]) {
			cursor++
		}
		start := cursor
		for cursor < len(source) && isASCIILetter(source[cursor]) {
			cursor++
		}
		name := strings.ToLower(string(source[start:cursor]))
		for cursor < len(source) && isDirectiveSpace(source[cursor]) {
			cursor++
		}
		if cursor < len(source) && source[cursor] == ':' && (name == "init" || name == "initialize") {
			return name, offset, true
		}
	}
	return "", 0, false
}

func isDirectiveSpace(value byte) bool {
	switch value {
	case ' ', '\t', '\r', '\n':
		return true
	default:
		return false
	}
}

func isASCIILetter(value byte) bool {
	return value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}
