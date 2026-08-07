package margo

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	authority "github.com/araihu/margo/internal/authority"
	socialcheck "github.com/araihu/margo/internal/socialcheck"
)

type PublicationMode string

const (
	PublicationPrivate PublicationMode = ""
	PublicationPublic  PublicationMode = "public"
)

type SocialImage struct {
	URL      string
	MIMEType string
	Width    int
	Height   int
	Alt      string
}

type SocialMetadata struct {
	Title        string
	Description  string
	CanonicalURL string
	SiteName     string
	Locale       string
	Image        SocialImage
}

type SocialRenderInput struct {
	Mode      PublicationMode
	Authority AuthorityRecord
	RoutePath string
	Metadata  SocialMetadata
	HeadOwner HeadOwnerSelection
}

type AuthorityRecord = authority.AuthorityRecord
type CanonicalOrigin = authority.CanonicalOrigin
type AuthorityRoutes = authority.AuthorityRoutes
type AuthorityOwner = authority.AuthorityOwner
type AuthorityAsset = authority.AuthorityAsset
type AuthorityEvidence = authority.AuthorityEvidence
type AuthorityDeployment = authority.AuthorityDeployment
type AuthorityReceipt = authority.AuthorityReceipt
type AuthoritySource = authority.AuthoritySource

func fmtInt(value int) string { return strconv.Itoa(value) }

func VerifyAuthorityRecord(data []byte) (AuthorityRecord, error) {
	return authority.VerifyAuthorityRecord(data)
}

func LoadAuthorityRecord(ctx context.Context, source AuthoritySource, location string) (AuthorityRecord, error) {
	return authority.LoadAuthorityRecord(ctx, source, location)
}

func RequireOneCompleteSocialSet(markup string, metadata SocialMetadata) error {
	return socialcheck.RequireOneCompleteSet(markup, metadata.CanonicalURL)
}

func homeSocialMetadata(origin CanonicalOrigin) SocialMetadata {
	base := strings.TrimRight(string(origin), "/")
	return SocialMetadata{
		Title:        "Margo: Markdown for Goshtoso",
		Description:  "Compile one Markdown document into Goshtoso-styled HTML, PDF, and static slide decks with deterministic, offline assets.",
		CanonicalURL: base + "/",
		SiteName:     "Margo",
		Locale:       "en_US",
		Image: SocialImage{
			URL:      base + "/assets/social/margo-v0.0.1.png",
			MIMEType: "image/png",
			Width:    1280,
			Height:   640,
			Alt:      "Margo name and Markdown document motif on a Goshtoso-themed background.",
		},
	}
}

func (m SocialMetadata) Validate(mode PublicationMode, authorityRecord AuthorityRecord, routePath string) error {
	if mode == PublicationPrivate {
		if m.CanonicalURL != "" || m.Image.URL != "" {
			return fmt.Errorf("publication.private_urls_forbidden")
		}
		return nil
	}
	if mode != PublicationPublic {
		return fmt.Errorf("publication.mode_invalid")
	}
	if err := authorityRecord.Validate(); err != nil {
		return err
	}
	if m.Title == "" || m.Description == "" || m.SiteName == "" || m.CanonicalURL == "" || routePath == "" {
		return fmt.Errorf("publication.metadata_required")
	}
	if strings.ContainsAny(m.Title+m.Description+m.SiteName+m.Image.Alt, "<>\x00\n\r") {
		return fmt.Errorf("publication.metadata_unsafe_text")
	}
	canonical, err := parseAbsoluteHTTPS(m.CanonicalURL)
	if err != nil {
		return fmt.Errorf("publication.canonical_invalid: %w", err)
	}
	origin, err := parseAbsoluteHTTPS(string(authorityRecord.CanonicalOrigin))
	if err != nil || canonical.Scheme != origin.Scheme || canonical.Host != origin.Host || canonical.User != nil || canonical.RawQuery != "" || canonical.Fragment != "" || canonical.Path != routePath {
		return fmt.Errorf("publication.canonical_out_of_origin")
	}
	if authorityRecord.Routes.Homepage != routePath && authorityRecord.Routes.Representative != routePath {
		return fmt.Errorf("publication.route_unlisted")
	}
	imageURL, err := parseAbsoluteHTTPS(m.Image.URL)
	if err != nil || imageURL.Scheme != origin.Scheme || imageURL.Host != origin.Host || imageURL.Path != authorityRecord.Routes.Preview || imageURL.RawQuery != "" || imageURL.Fragment != "" {
		return fmt.Errorf("publication.image_out_of_origin")
	}
	if m.Image.MIMEType != "image/png" && m.Image.MIMEType != "image/jpeg" || m.Image.Width != authorityRecord.Asset.Width || m.Image.Height != authorityRecord.Asset.Height || m.Image.Alt == "" {
		return fmt.Errorf("publication.image_invalid")
	}
	return nil
}

func parseAbsoluteHTTPS(value string) (*url.URL, error) {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil {
		return nil, fmt.Errorf("absolute HTTPS URL required")
	}
	return parsed, nil
}

// RenderSocialStandalone consumes the C6 owner selection and adds exactly one
// initial HTML metadata set for public output. Private output is URL-free.
func RenderSocialStandalone(result *RenderResult, input SocialRenderInput) (templ.Component, error) {
	owner := input.HeadOwner
	if owner.SchemaVersion == "" {
		owner = FrozenHeadOwnerSelection()
	}
	if err := owner.Validate(); err != nil {
		return nil, err
	}
	if owner != FrozenHeadOwnerSelection() {
		return nil, fmt.Errorf("publication.head_owner_mutation")
	}
	if err := input.Metadata.Validate(input.Mode, input.Authority, input.RoutePath); err != nil {
		return nil, err
	}
	base, err := RenderStandalone(result)
	if err != nil {
		return nil, err
	}
	baseBytes, err := renderComponentBytes(base)
	if err != nil {
		return nil, err
	}
	if input.Mode == PublicationPrivate {
		return templ.Raw(string(baseBytes)), nil
	}
	if owner.Owner != "margo" || owner.Primitive != "socialMetadataTags" {
		return nil, fmt.Errorf("publication.head_owner_primitive_unavailable")
	}
	tags, err := renderComponentBytes(socialMetadataTags(input.Metadata))
	if err != nil {
		return nil, err
	}
	html := string(baseBytes)
	titleStart := strings.Index(html, "<title>")
	titleEnd := strings.Index(html[titleStart:], "</title>")
	if titleStart < 0 || titleEnd < 0 {
		return nil, fmt.Errorf("publication.standalone_title_missing")
	}
	titleEnd += titleStart + len("</title>")
	html = html[:titleStart] + string(tags) + html[titleEnd:]
	return templ.Raw(html), nil
}
