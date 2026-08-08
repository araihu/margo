package socialcheck_test

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	margo "github.com/araihu/margo"
	socialcheck "github.com/araihu/margo/internal/socialcheck"
)

func TestPreviewAsset(t *testing.T) {
	if err := socialcheck.RequirePNG("../../assets/social/margo-v0.0.1.png", 1280, 640, 1_000_000); err != nil {
		t.Fatal(err)
	}
}

func TestProductionPageFixture(t *testing.T) {
	html, err := os.ReadFile("../../testdata/social/home.html")
	if err != nil {
		t.Fatal(err)
	}
	if err := socialcheck.RequireOneCompleteSet(string(html), "https://margo.invalid/"); err != nil {
		t.Fatal(err)
	}
}

func loadAuthorityFixtureForTest(t *testing.T) margo.AuthorityRecord {
	t.Helper()
	data, err := os.ReadFile("../../testdata/authority/record.json")
	if err != nil {
		t.Fatal(err)
	}
	record, err := margo.VerifyAuthorityRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	return record
}

func mustRenderSource(t *testing.T, source string) *margo.RenderResult {
	t.Helper()
	compiler := margo.New(margo.WithHostPolicy(margo.Policy{RawHTML: margo.RawHTMLDeny, OutputBytes: margo.MaxOutputBytes}))
	document, err := compiler.Compile(context.Background(), margo.Source{Name: "social.md", Content: []byte(source)})
	if err != nil {
		t.Fatal(err)
	}
	result, err := compiler.Render(context.Background(), document)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func homeSocialMetadata(origin margo.CanonicalOrigin) margo.SocialMetadata {
	base := strings.TrimRight(string(origin), "/")
	return margo.SocialMetadata{
		Title: "Margo: Markdown for Goshtoso", Description: "Compile one Markdown document into Goshtoso-styled HTML, PDF, and static slide decks with deterministic, offline assets.", CanonicalURL: base + "/", SiteName: "Margo", Locale: "en_US",
		Image: margo.SocialImage{URL: base + "/assets/social/margo-v0.0.1.png", MIMEType: "image/png", Width: 1280, Height: 640, Alt: "Margo name and Markdown document motif on a Goshtoso-themed background."},
	}
}

func TestPublicStandaloneSocialMetadata(t *testing.T) {
	authority := loadAuthorityFixtureForTest(t)
	result := mustRenderSource(t, "# Home\n\nWelcome")
	input := margo.SocialRenderInput{Mode: margo.PublicationPublic, Authority: authority, RoutePath: authority.Routes.Homepage, Metadata: homeSocialMetadata(authority.CanonicalOrigin), HeadOwner: margo.FrozenHeadOwnerSelection()}
	component, err := margo.RenderSocialStandalone(result, input)
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := component.Render(context.Background(), &got); err != nil {
		t.Fatal(err)
	}
	if err := socialcheck.RequireOneCompleteSet(got.String(), input.Metadata.CanonicalURL); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"og:url", "og:image", "twitter:card", "Margo: Markdown for Goshtoso"} {
		if !strings.Contains(got.String(), want) {
			t.Fatalf("public HTML missing %q:\n%s", want, got.String())
		}
	}
}

func TestPrivateStandaloneOmitsSocialURLs(t *testing.T) {
	component, err := margo.RenderSocialStandalone(mustRenderSource(t, "# Private"), margo.SocialRenderInput{Mode: margo.PublicationPrivate, HeadOwner: margo.FrozenHeadOwnerSelection()})
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := component.Render(context.Background(), &got); err != nil {
		t.Fatal(err)
	}
	for _, marker := range []string{"rel=\"canonical\"", "og:", "twitter:", `href="https://`, `content="https://`} {
		if strings.Contains(got.String(), marker) {
			t.Fatalf("private HTML contains social marker %q", marker)
		}
	}
}

func TestSocialMetadataValidation(t *testing.T) {
	authority := loadAuthorityFixtureForTest(t)
	metadata := homeSocialMetadata(authority.CanonicalOrigin)
	metadata.CanonicalURL = "https://outside.invalid/"
	if err := metadata.Validate(margo.PublicationPublic, authority, authority.Routes.Homepage); err == nil {
		t.Fatal("out-of-origin canonical URL unexpectedly accepted")
	}
	metadata = homeSocialMetadata(authority.CanonicalOrigin)
	metadata.Image.Width = 1
	if err := metadata.Validate(margo.PublicationPublic, authority, authority.Routes.Homepage); err == nil {
		t.Fatal("invalid preview dimensions unexpectedly accepted")
	}
}
