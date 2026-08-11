// Package embed provides a host-owned typed extension for narrowly authorized
// remote media. It never accepts arbitrary HTML from a document.
package embed

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"

	margo "github.com/araihu/margo"
	"golang.org/x/net/idna"
	"gopkg.in/yaml.v3"
)

const (
	Fence              = "trusted-embed"
	maxRequestURLBytes = 4096
)

type Projection string

const (
	ProjectionDeny        Projection = "deny"
	ProjectionStaticLink  Projection = "static-link"
	ProjectionInteractive Projection = "interactive"
)

type Kind string

const (
	KindIframe Kind = "iframe"
	KindVideo  Kind = "video"
)

type SandboxToken string

const (
	SandboxAllowPresentation SandboxToken = "allow-presentation"
	SandboxAllowScripts      SandboxToken = "allow-scripts"
)

type ReferrerPolicy string

const (
	ReferrerNoReferrer ReferrerPolicy = "no-referrer"
)

// Policy is immutable after Extension returns. Origins are exact HTTPS origins
// and document requests cannot add kinds, origins, or sandbox capabilities.
type Policy struct {
	Projection     Projection     `json:"projection"`
	AllowedKinds   []Kind         `json:"allowedKinds"`
	AllowedOrigins []string       `json:"allowedOrigins"`
	IframeSandbox  []SandboxToken `json:"iframeSandbox"`
	ReferrerPolicy ReferrerPolicy `json:"referrerPolicy"`
}

type request struct {
	Kind   Kind   `yaml:"kind"`
	URL    string `yaml:"url"`
	Title  string `yaml:"title"`
	Width  int    `yaml:"width,omitempty"`
	Height int    `yaml:"height,omitempty"`
}

type session struct{ policy Policy }

// NormalizePolicy returns a detached deterministic policy value suitable for
// hashing, storage, or reuse across compiler instances.
func NormalizePolicy(policy Policy) (Policy, error) { return normalizePolicy(policy) }

// ConfigurationHash identifies the normalized extension configuration.
func (policy Policy) ConfigurationHash() (string, error) {
	normalized, err := normalizePolicy(policy)
	if err != nil {
		return "", err
	}
	return policyHash(normalized)
}

// Extension reserves the trusted-embed fence under one frozen host policy.
func Extension(policy Policy) margo.ExtensionRegistration {
	normalized, err := normalizePolicy(policy)
	configurationHash := "invalid"
	if err == nil {
		configurationHash, err = policyHash(normalized)
	}
	return margo.ExtensionRegistration{
		Identity: margo.ExtensionIdentity{
			Name: "margo-trusted-embed", Version: "v1", ConfigurationHash: configurationHash,
			Capabilities: []string{"typed-remote-embed", string(normalized.Projection)},
		},
		Fences: []string{Fence},
		Check: func(ctx context.Context, node margo.ExtensionNode) error {
			if err != nil {
				return err
			}
			if checkErr := ctx.Err(); checkErr != nil {
				return checkErr
			}
			parsed, checkErr := parseRequest(node)
			if checkErr != nil {
				return checkErr
			}
			if checkErr = validateRequest(normalized, node, parsed); checkErr != nil {
				return checkErr
			}
			if normalized.Projection == ProjectionDeny {
				return diagnostic(node, "embed.policy_denied", "/embed", "trusted embeds are denied for this output target", "Remove the embed or choose an explicitly authorized output target.")
			}
			return nil
		},
		Factory: func(margo.RenderContext) (margo.ExtensionSession, error) {
			if err != nil {
				return nil, err
			}
			return session{policy: normalized}, nil
		},
	}
}

func (s session) Render(ctx context.Context, node margo.ExtensionNode, output io.Writer) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	parsed, err := parseRequest(node)
	if err != nil {
		return err
	}
	if err := validateRequest(s.policy, node, parsed); err != nil {
		return err
	}
	if s.policy.Projection == ProjectionStaticLink {
		_, err = fmt.Fprintf(output, `<figure class="margo-trusted-embed margo-trusted-embed--static"><a class="margo-trusted-embed__link" href="%s" rel="noreferrer" referrerpolicy="%s">%s</a></figure>`, html.EscapeString(parsed.URL), html.EscapeString(string(s.policy.ReferrerPolicy)), html.EscapeString(parsed.Title))
		return err
	}
	if s.policy.Projection != ProjectionInteractive {
		return diagnostic(node, "embed.policy_denied", "/embed", "trusted embeds are denied for this output target", "Remove the embed or choose an explicitly authorized output target.")
	}
	sandbox := make([]string, len(s.policy.IframeSandbox))
	for index, token := range s.policy.IframeSandbox {
		sandbox[index] = string(token)
	}
	_, err = fmt.Fprintf(output, `<figure class="margo-trusted-embed margo-trusted-embed--iframe"><iframe class="margo-trusted-embed__frame" src="%s" title="%s" width="%d" height="%d" sandbox="%s" referrerpolicy="%s" loading="lazy"></iframe></figure>`,
		html.EscapeString(parsed.URL), html.EscapeString(parsed.Title), parsed.Width, parsed.Height,
		html.EscapeString(strings.Join(sandbox, " ")), html.EscapeString(string(s.policy.ReferrerPolicy)))
	return err
}

func parseRequest(node margo.ExtensionNode) (request, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(node.Payload))
	decoder.KnownFields(true)
	var parsed request
	if err := decoder.Decode(&parsed); err != nil {
		return request{}, diagnostic(node, "embed.request_invalid", "/embed", fmt.Sprintf("trusted embed payload is invalid: %v", err), "Use only kind, url, title, width, and height fields.")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return request{}, diagnostic(node, "embed.request_invalid", "/embed", "trusted embed payload must contain one YAML document", "Remove additional YAML documents.")
	}
	if parsed.Width == 0 {
		parsed.Width = 640
	}
	if parsed.Height == 0 {
		parsed.Height = 360
	}
	return parsed, nil
}

func validateRequest(policy Policy, node margo.ExtensionNode, parsed request) error {
	allowedKind := false
	for _, kind := range policy.AllowedKinds {
		allowedKind = allowedKind || kind == parsed.Kind
	}
	if !allowedKind {
		return diagnostic(node, "embed.kind_denied", "/embed/kind", fmt.Sprintf("embed kind %q is not allowed", parsed.Kind), "Choose a kind allowed by the host policy.")
	}
	if strings.TrimSpace(parsed.Title) == "" || len([]byte(parsed.Title)) > 256 {
		return diagnostic(node, "embed.title_invalid", "/embed/title", "embed title must contain text and be at most 256 UTF-8 bytes", "Add a concise accessible title.")
	}
	if parsed.Width < 1 || parsed.Width > 4096 || parsed.Height < 1 || parsed.Height > 4096 {
		return diagnostic(node, "embed.dimensions_invalid", "/embed", "embed dimensions must be between 1 and 4096", "Use bounded positive width and height values.")
	}
	origin, err := requestOrigin(parsed.URL)
	if err != nil {
		return diagnostic(node, "embed.url_invalid", "/embed/url", err.Error(), "Use an absolute HTTPS URL from an origin allowed by the host policy.")
	}
	index := sort.SearchStrings(policy.AllowedOrigins, origin)
	if index >= len(policy.AllowedOrigins) || policy.AllowedOrigins[index] != origin {
		return diagnostic(node, "embed.origin_denied", "/embed/url", fmt.Sprintf("embed origin %q is not allowed", origin), "Use a URL from an origin allowed by the host policy.")
	}
	return nil
}

func normalizePolicy(policy Policy) (Policy, error) {
	if policy.Projection != ProjectionDeny && policy.Projection != ProjectionStaticLink && policy.Projection != ProjectionInteractive {
		return Policy{}, fmt.Errorf("embed.policy_invalid: unsupported projection %q", policy.Projection)
	}
	if policy.Projection != ProjectionDeny && len(policy.AllowedKinds) == 0 {
		return Policy{}, fmt.Errorf("embed.policy_invalid: a non-deny projection requires at least one allowed kind")
	}
	if policy.Projection != ProjectionDeny && len(policy.AllowedOrigins) == 0 {
		return Policy{}, fmt.Errorf("embed.policy_invalid: a non-deny projection requires at least one allowed origin")
	}
	if policy.ReferrerPolicy == "" {
		policy.ReferrerPolicy = ReferrerNoReferrer
	}
	if policy.ReferrerPolicy != ReferrerNoReferrer {
		return Policy{}, fmt.Errorf("embed.policy_invalid: unsupported referrer policy %q", policy.ReferrerPolicy)
	}
	policy.AllowedKinds = append([]Kind(nil), policy.AllowedKinds...)
	policy.AllowedOrigins = append([]string(nil), policy.AllowedOrigins...)
	policy.IframeSandbox = append([]SandboxToken(nil), policy.IframeSandbox...)
	for _, kind := range policy.AllowedKinds {
		if kind != KindIframe && kind != KindVideo {
			return Policy{}, fmt.Errorf("embed.policy_invalid: unsupported kind %q", kind)
		}
		if policy.Projection == ProjectionInteractive && kind == KindVideo {
			return Policy{}, fmt.Errorf("embed.policy_invalid: interactive video cannot enforce the required no-referrer policy; use static-link")
		}
	}
	for index, origin := range policy.AllowedOrigins {
		normalized, err := policyOrigin(origin)
		if err != nil {
			return Policy{}, fmt.Errorf("embed.policy_invalid: %w", err)
		}
		policy.AllowedOrigins[index] = normalized
	}
	for _, token := range policy.IframeSandbox {
		if token != SandboxAllowPresentation && token != SandboxAllowScripts {
			return Policy{}, fmt.Errorf("embed.policy_invalid: unsupported sandbox token %q", token)
		}
	}
	sort.Slice(policy.AllowedKinds, func(i, j int) bool { return policy.AllowedKinds[i] < policy.AllowedKinds[j] })
	sort.Strings(policy.AllowedOrigins)
	sort.Slice(policy.IframeSandbox, func(i, j int) bool { return policy.IframeSandbox[i] < policy.IframeSandbox[j] })
	if duplicateKinds(policy.AllowedKinds) || duplicateStrings(policy.AllowedOrigins) || duplicateSandbox(policy.IframeSandbox) {
		return Policy{}, fmt.Errorf("embed.policy_invalid: policy lists must not contain duplicates")
	}
	return policy, nil
}

func requestOrigin(value string) (string, error) {
	if len([]byte(value)) == 0 || len([]byte(value)) > maxRequestURLBytes {
		return "", fmt.Errorf("embed URL must contain 1 to %d UTF-8 bytes", maxRequestURLBytes)
	}
	if strings.Contains(value, `\`) {
		return "", fmt.Errorf("embed URL must not contain backslashes")
	}
	parsed, err := url.Parse(value)
	if err != nil || strings.ToLower(parsed.Scheme) != "https" || parsed.Host == "" || parsed.User != nil {
		return "", fmt.Errorf("embed URL must be absolute HTTPS without credentials")
	}
	origin, err := canonicalHTTPSOrigin(parsed)
	if err != nil {
		return "", fmt.Errorf("embed URL has an invalid origin: %w", err)
	}
	return origin, nil
}

func policyOrigin(value string) (string, error) {
	if len([]byte(value)) == 0 || len([]byte(value)) > maxRequestURLBytes {
		return "", fmt.Errorf("allowed origin must contain 1 to %d UTF-8 bytes", maxRequestURLBytes)
	}
	if strings.Contains(value, `\`) {
		return "", fmt.Errorf("allowed origin %q must not contain backslashes", value)
	}
	parsed, err := url.Parse(value)
	if err != nil || strings.ToLower(parsed.Scheme) != "https" || parsed.Host == "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("allowed origin %q must be an HTTPS origin without path, query, fragment, or credentials", value)
	}
	origin, err := canonicalHTTPSOrigin(parsed)
	if err != nil {
		return "", fmt.Errorf("allowed origin %q has an invalid origin: %w", value, err)
	}
	return origin, nil
}

func canonicalHTTPSOrigin(parsed *url.URL) (string, error) {
	hostname, err := canonicalHost(parsed.Hostname())
	if err != nil {
		return "", err
	}
	if hostname == "" {
		return "", fmt.Errorf("hostname is empty")
	}
	port := parsed.Port()
	if port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", fmt.Errorf("port is outside 1 through 65535")
		}
		if number == 443 {
			port = ""
		}
	}
	host := hostname
	if port != "" {
		host = net.JoinHostPort(hostname, port)
	} else if strings.Contains(hostname, ":") {
		host = "[" + hostname + "]"
	}
	return "https://" + host, nil
}

func canonicalHost(value string) (string, error) {
	if value == "" {
		return "", fmt.Errorf("hostname is empty")
	}
	if strings.ContainsAny(value, `*\`) {
		return "", fmt.Errorf("hostname contains wildcard or backslash syntax")
	}
	if address, err := netip.ParseAddr(value); err == nil {
		if address.Zone() != "" {
			return "", fmt.Errorf("scoped IP addresses are not allowed")
		}
		return address.String(), nil
	}
	ascii, err := idna.Lookup.ToASCII(value)
	if err != nil {
		return "", fmt.Errorf("hostname is not a valid IDNA domain: %w", err)
	}
	ascii = strings.ToLower(ascii)
	if ascii == "" || strings.ContainsAny(ascii, `*\`) {
		return "", fmt.Errorf("hostname is invalid")
	}
	if address, err := netip.ParseAddr(ascii); err == nil {
		if address.Zone() != "" {
			return "", fmt.Errorf("scoped IP addresses are not allowed")
		}
		return address.String(), nil
	}
	if looksLikeNoncanonicalIP(ascii) {
		return "", fmt.Errorf("noncanonical IP address syntax is not allowed")
	}
	return ascii, nil
}

func looksLikeNoncanonicalIP(value string) bool {
	if strings.Contains(value, ":") {
		return true
	}
	value = strings.TrimSuffix(value, ".")
	parts := strings.Split(strings.ToLower(value), ".")
	if len(parts) == 0 || len(parts) > 4 {
		return false
	}
	for _, part := range parts {
		if part == "" {
			return false
		}
		base := 10
		digits := part
		if strings.HasPrefix(part, "0x") {
			base = 16
			digits = strings.TrimPrefix(part, "0x")
		}
		if digits == "" {
			return false
		}
		if _, err := strconv.ParseUint(digits, base, 32); err != nil {
			return false
		}
	}
	return true
}

func policyHash(policy Policy) (string, error) {
	data, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("embed.policy_invalid: %w", err)
	}
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:]), nil
}

func duplicateKinds(values []Kind) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
}

func duplicateStrings(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
}

func duplicateSandbox(values []SandboxToken) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
}

func diagnostic(node margo.ExtensionNode, code, pointer, message, hint string) error {
	return &margo.DiagnosticError{Diagnostics: []margo.Diagnostic{{
		Code: code, Severity: margo.SeverityError, Source: node.Source.Source,
		Line: node.Source.Line, Column: node.Source.Column, Pointer: pointer,
		Message: message, Hint: hint,
	}}}
}
