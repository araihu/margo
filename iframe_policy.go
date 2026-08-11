package margo

import (
	"fmt"
	"net"
	"net/netip"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/idna"
)

// Projection selects how an authorized iframe is represented for one target.
type Projection string

const (
	ProjectionDeny        Projection = "deny"
	ProjectionStaticLink  Projection = "static-link"
	ProjectionInteractive Projection = "interactive"
)

type SandboxToken string

const (
	SandboxAllowPresentation SandboxToken = "allow-presentation"
	SandboxAllowScripts      SandboxToken = "allow-scripts"
)

type ReferrerPolicy string

const ReferrerNoReferrer ReferrerPolicy = "no-referrer"

// TargetProjections keeps capability decisions independent per output target.
type TargetProjections struct {
	HTML Projection `json:"html"`
	Site Projection `json:"site"`
	PDF  Projection `json:"pdf"`
	Deck Projection `json:"deck"`
}

// IframePolicy is host-owned. Documents provide only src, title, width, and
// height; they cannot widen these capabilities.
type IframePolicy struct {
	AllowedOrigins []string          `json:"allowedOrigins"`
	Sandbox        []SandboxToken    `json:"sandbox"`
	ReferrerPolicy ReferrerPolicy    `json:"referrerPolicy"`
	Projections    TargetProjections `json:"projections"`
}

func cloneIframePolicy(policy *IframePolicy) *IframePolicy {
	if policy == nil {
		return nil
	}
	cloned := *policy
	cloned.AllowedOrigins = append([]string(nil), policy.AllowedOrigins...)
	cloned.Sandbox = append([]SandboxToken(nil), policy.Sandbox...)
	return &cloned
}

func normalizeIframePolicy(policy IframePolicy) (IframePolicy, error) {
	if len(policy.AllowedOrigins) == 0 || len(policy.AllowedOrigins) > 64 {
		return IframePolicy{}, fmt.Errorf("allowedOrigins must contain 1 to 64 exact HTTPS origins")
	}
	policy.AllowedOrigins = append([]string(nil), policy.AllowedOrigins...)
	for index, origin := range policy.AllowedOrigins {
		normalized, err := canonicalPolicyOrigin(origin)
		if err != nil {
			return IframePolicy{}, err
		}
		policy.AllowedOrigins[index] = normalized
	}
	sort.Strings(policy.AllowedOrigins)
	if duplicateStrings(policy.AllowedOrigins) {
		return IframePolicy{}, fmt.Errorf("allowedOrigins must not contain duplicates after canonicalization")
	}
	policy.Sandbox = append([]SandboxToken(nil), policy.Sandbox...)
	for _, token := range policy.Sandbox {
		if token != SandboxAllowPresentation && token != SandboxAllowScripts {
			return IframePolicy{}, fmt.Errorf("unsupported sandbox token %q", token)
		}
	}
	sort.Slice(policy.Sandbox, func(i, j int) bool { return policy.Sandbox[i] < policy.Sandbox[j] })
	for index := 1; index < len(policy.Sandbox); index++ {
		if policy.Sandbox[index] == policy.Sandbox[index-1] {
			return IframePolicy{}, fmt.Errorf("sandbox must not contain duplicates")
		}
	}
	if policy.ReferrerPolicy == "" {
		policy.ReferrerPolicy = ReferrerNoReferrer
	}
	if policy.ReferrerPolicy != ReferrerNoReferrer {
		return IframePolicy{}, fmt.Errorf("referrerPolicy must be no-referrer")
	}
	defaults := TargetProjections{HTML: ProjectionDeny, Site: ProjectionDeny, PDF: ProjectionStaticLink, Deck: ProjectionDeny}
	if policy.Projections.HTML == "" {
		policy.Projections.HTML = defaults.HTML
	}
	if policy.Projections.Site == "" {
		policy.Projections.Site = defaults.Site
	}
	if policy.Projections.PDF == "" {
		policy.Projections.PDF = defaults.PDF
	}
	if policy.Projections.Deck == "" {
		policy.Projections.Deck = defaults.Deck
	}
	for target, projection := range map[string]Projection{
		"html": policy.Projections.HTML, "site": policy.Projections.Site,
		"pdf": policy.Projections.PDF, "deck": policy.Projections.Deck,
	} {
		if projection != ProjectionDeny && projection != ProjectionStaticLink && projection != ProjectionInteractive {
			return IframePolicy{}, fmt.Errorf("unsupported %s projection %q", target, projection)
		}
		if target == "pdf" && projection == ProjectionInteractive {
			return IframePolicy{}, fmt.Errorf("PDF projection cannot be interactive")
		}
	}
	return policy, nil
}

func canonicalPolicyOrigin(value string) (string, error) {
	if len([]byte(value)) == 0 || len([]byte(value)) > 4096 || strings.Contains(value, `\`) {
		return "", fmt.Errorf("allowed origin must contain 1 to 4096 UTF-8 bytes without backslashes")
	}
	parsed, err := url.Parse(value)
	if err != nil || strings.ToLower(parsed.Scheme) != "https" || parsed.Host == "" || parsed.User != nil || (parsed.Path != "" && parsed.Path != "/") || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", fmt.Errorf("allowed origin %q must be an HTTPS origin without path, query, fragment, or credentials", value)
	}
	hostname, err := canonicalPolicyHost(parsed.Hostname())
	if err != nil {
		return "", fmt.Errorf("allowed origin %q: %w", value, err)
	}
	port := parsed.Port()
	if port != "" {
		number, err := strconv.Atoi(port)
		if err != nil || number < 1 || number > 65535 {
			return "", fmt.Errorf("allowed origin %q has invalid port", value)
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

func canonicalPolicyHost(value string) (string, error) {
	if value == "" || strings.ContainsAny(value, `*\`) {
		return "", fmt.Errorf("hostname is empty or contains wildcard syntax")
	}
	if address, err := netip.ParseAddr(value); err == nil {
		if address.Zone() != "" {
			return "", fmt.Errorf("scoped IP addresses are not allowed")
		}
		return address.String(), nil
	}
	numericLike := true
	for _, character := range value {
		if (character < '0' || character > '9') && character != '.' {
			numericLike = false
			break
		}
	}
	if numericLike {
		return "", fmt.Errorf("noncanonical numeric IP address is not allowed")
	}
	ascii, err := idna.Lookup.ToASCII(value)
	if err != nil {
		return "", fmt.Errorf("hostname is not a valid IDNA domain: %w", err)
	}
	ascii = strings.ToLower(ascii)
	if ascii == "" || strings.ContainsAny(ascii, `*\:`) {
		return "", fmt.Errorf("hostname is invalid")
	}
	return strings.TrimSuffix(ascii, "."), nil
}

func duplicateStrings(values []string) bool {
	for index := 1; index < len(values); index++ {
		if values[index] == values[index-1] {
			return true
		}
	}
	return false
}
