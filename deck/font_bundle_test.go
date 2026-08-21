package deck

import (
	"context"
	"strings"
	"testing"

	"github.com/araihu/margo"
)

func TestFontBundleDigestV1KnownAnswer(t *testing.T) {
	digest, err := FontBundleDigestV1([]FontFaceAsset{
		{Family: "Margo Sans", Weight: 400, Bytes: []byte{0x00, 0x01, 0x02}},
		{Family: "Margo Sans", Weight: 700, Bytes: []byte{0x03, 0x04, 0x05}},
		{Family: "Margo Mono", Weight: 400, Bytes: []byte{0x06, 0x07}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if digest != "ca7de7d8ae3ee43e4984afd2b18a81825ab383bfd379d2d93b986d4c5d59aaa1" {
		t.Fatalf("digest = %q", digest)
	}
}

func TestRenderResolvesImmutableFontBundleRequest(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{Name: "deck.md", Markdown: []byte("# One\n")}, WithValidationRequest(margo.RuntimeValidationRequest{
		ViewportWidth: 800, ViewportHeight: 600, DeviceScaleFactor: 1, Zoom: 1, BrowserProfile: "chromium-deck-v1",
	}))
	if err != nil {
		t.Fatal(err)
	}
	request := result.ValidationRequest()
	if request.ViewportWidth != 800 || request.ViewportHeight != 600 || request.ExpectedFontBundleDigest == "" {
		t.Fatalf("request = %+v", request)
	}
	bad := margo.RuntimeValidationRequest{
		ViewportWidth: 800, ViewportHeight: 600, DeviceScaleFactor: 1, Zoom: 1,
		BrowserProfile: "chromium-deck-v1", ExpectedFontBundleDigest: strings.Repeat("f", 64),
	}
	_, err = Render(context.Background(), margo.New(), RenderInput{Name: "deck.md", Markdown: []byte("# One\n")}, WithValidationRequest(bad))
	if err == nil || !strings.Contains(err.Error(), "deck.validator_profile_mismatch") {
		t.Fatalf("mismatched request error = %v", err)
	}
}

func TestRenderEmbedsCanonicalFontFaces(t *testing.T) {
	result, err := Render(context.Background(), margo.New(), RenderInput{Name: "deck.md", Markdown: []byte("# One\n")})
	if err != nil {
		t.Fatal(err)
	}
	html := string(result.HTML())
	for _, fragment := range []string{
		`data-margo-font-bundle-digest="`,
		`<style data-margo-deck-fonts>`,
		`font-family:'Margo Sans'`,
		`font-family:'Margo Serif'`,
		`font-family:'Margo Mono'`,
		`src:url(data:font/woff2;base64,`,
		`margoGetDeckFontEvidence`,
	} {
		if !strings.Contains(html, fragment) {
			t.Fatalf("HTML missing %q", fragment)
		}
	}
}

func TestFontBundleDigestV1RejectsAmbiguousFaceOrder(t *testing.T) {
	faces := []FontFaceAsset{
		{Family: "Margo Sans", Weight: 700, Bytes: []byte{1}},
		{Family: "Margo Sans", Weight: 400, Bytes: []byte{2}},
	}
	if _, err := FontBundleDigestV1(faces); err == nil {
		t.Fatal("unsorted faces unexpectedly accepted")
	}
	faces[0].Family = ""
	if _, err := FontBundleDigestV1(faces); err == nil {
		t.Fatal("empty family unexpectedly accepted")
	}
	faces[0].Family = "Margo Sans"
	faces[0].Weight = 400
	faces[1].Weight = 700
	faces[0].Bytes = nil
	if _, err := FontBundleDigestV1(faces); err == nil || !strings.Contains(err.Error(), "font_bundle") {
		t.Fatalf("nil bytes error = %v", err)
	}
}

func TestBundledFontFacesUseVersionedWOFF2Assets(t *testing.T) {
	faces, err := bundledFontFaces(margo.ThemeModern)
	if err != nil {
		t.Fatal(err)
	}
	if len(faces) != 6 {
		t.Fatalf("modern face count = %d", len(faces))
	}
	for _, face := range faces {
		if len(face.Bytes) < 4 || string(face.Bytes[:4]) != "wOF2" {
			t.Fatalf("%s %d is not WOFF2", face.Family, face.Weight)
		}
	}
	minimal, err := bundledFontFaces(margo.ThemeMinimal)
	if err != nil {
		t.Fatal(err)
	}
	if len(minimal) != 9 {
		t.Fatalf("minimal face count = %d", len(minimal))
	}
}
