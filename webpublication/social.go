package webpublication

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	authority "github.com/araihu/margo/internal/authority"
	socialcheck "github.com/araihu/margo/internal/socialcheck"
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

type AuthorityRecord = authority.AuthorityRecord
type CanonicalOrigin = authority.CanonicalOrigin
type AuthorityRoutes = authority.AuthorityRoutes
type AuthorityOwner = authority.AuthorityOwner
type AuthorityAsset = authority.AuthorityAsset
type AuthorityEvidence = authority.AuthorityEvidence
type AuthorityDeployment = authority.AuthorityDeployment
type AuthorityReceipt = authority.AuthorityReceipt
type AuthoritySource = authority.AuthoritySource

func VerifyAuthorityRecord(data []byte) (AuthorityRecord, error) {
	return authority.VerifyAuthorityRecord(data)
}

func LoadAuthorityRecord(ctx context.Context, source AuthoritySource, location string) (AuthorityRecord, error) {
	return authority.LoadAuthorityRecord(ctx, source, location)
}

func RequireOneCompleteSocialSet(markup string, metadata SocialMetadata) error {
	return socialcheck.RequireOneCompleteSet(markup, metadata.CanonicalURL)
}

func (m SocialMetadata) Validate(authorityRecord AuthorityRecord, routePath string) error {
	if err := authorityRecord.Validate(); err != nil {
		return err
	}
	if m.Title == "" || m.Description == "" || m.SiteName == "" || m.CanonicalURL == "" || routePath == "" {
		return fmt.Errorf("webpublication.metadata_required")
	}
	if strings.ContainsAny(m.Title+m.Description+m.SiteName+m.Image.Alt, "<>\x00\n\r") {
		return fmt.Errorf("webpublication.metadata_unsafe_text")
	}
	canonical, err := parseAbsoluteHTTPS(m.CanonicalURL)
	if err != nil {
		return fmt.Errorf("webpublication.canonical_invalid: %w", err)
	}
	origin, err := parseAbsoluteHTTPS(string(authorityRecord.CanonicalOrigin))
	if err != nil || canonical.Scheme != origin.Scheme || canonical.Host != origin.Host || canonical.User != nil || canonical.RawQuery != "" || canonical.Fragment != "" || canonical.Path != routePath {
		return fmt.Errorf("webpublication.canonical_out_of_origin")
	}
	if authorityRecord.Routes.Homepage != routePath && authorityRecord.Routes.Representative != routePath {
		return fmt.Errorf("webpublication.route_unlisted")
	}
	imageURL, err := parseAbsoluteHTTPS(m.Image.URL)
	if err != nil || imageURL.Scheme != origin.Scheme || imageURL.Host != origin.Host || imageURL.Path != authorityRecord.Routes.Preview || imageURL.RawQuery != "" || imageURL.Fragment != "" {
		return fmt.Errorf("webpublication.image_out_of_origin")
	}
	if (m.Image.MIMEType != "image/png" && m.Image.MIMEType != "image/jpeg") || m.Image.Width != authorityRecord.Asset.Width || m.Image.Height != authorityRecord.Asset.Height || m.Image.Alt == "" {
		return fmt.Errorf("webpublication.image_invalid")
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
