package margo

import (
	"strings"
	"testing"
)

func TestRenderSocialStandaloneDelegatesToPublicationComposition(t *testing.T) {
	authority := loadPublicationAuthority(t)
	metadata := SocialMetadata{
		Title: "Shared social title", Description: "Shared social description.",
		CanonicalURL: string(authority.CanonicalOrigin) + authority.Routes.Representative,
		SiteName:     "Margo", Locale: "en_US",
		Image: SocialImage{
			URL:      string(authority.CanonicalOrigin) + authority.Routes.Preview,
			MIMEType: authority.Asset.MIMEType, Width: authority.Asset.Width, Height: authority.Asset.Height,
			Alt: "Editorial preview fixture.",
		},
	}
	component, err := RenderSocialStandalone(mustRenderSource(t, "# Body title\n"), SocialRenderInput{
		Mode: PublicationPublic, Authority: authority, RoutePath: authority.Routes.Representative,
		Metadata: metadata, HeadOwner: FrozenHeadOwnerSelection(),
	})
	if err != nil {
		t.Fatal(err)
	}
	markup := renderComponent(t, component)
	for _, want := range []string{
		`<title>Shared social title</title>`,
		`data-margo-requirement="goshtoso.styles"`,
		`data-margo-editorial-fingerprint=`,
	} {
		if !strings.Contains(markup, want) {
			t.Fatalf("delegated social output missing %q: %s", want, markup)
		}
	}
	if strings.Count(markup, "<title>") != 1 || strings.Contains(markup, "Body title</title>") {
		t.Fatalf("head metadata was not composed directly: %s", markup)
	}
}
