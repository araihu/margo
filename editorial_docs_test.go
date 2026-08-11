package margo

import (
	"os"
	"strings"
	"testing"
)

func TestHTMLDocumentationNamesDecoupledPublicContract(t *testing.T) {
	readme := readEditorialRepoFile(t, "README.md")
	for _, want := range []string{
		"RenderHTML", "RenderHTMLPage", "HTMLAssetHandler", "HTMLPageInput",
		"Head:", "Header:", "BeforeContent:", "Footer:",
		"/assets/", "/margo-assets/", "/charts/assets/",
		"HTMLDependenciesLocal", "HTMLDependenciesInline",
		"mux.Handle(\"/assets/\", goshtosoassets.Handler())",
		"mux.Handle(\"/margo-assets/\", margo.HTMLAssetHandler())",
		"mux.Handle(chartassets.Prefix, chartassets.Handler())",
		"Margo handler does not serve either dependency mount.",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README missing %q", want)
		}
	}
	for _, forbidden := range []string{"RenderEditorial", "margo.RenderPublication", "margo.PublicationInput", "EditorialAssetHandler", "webpublication"} {
		if strings.Contains(readme, forbidden) {
			t.Fatalf("README retains coupled API %q", forbidden)
		}
	}

	chartsReadme := readEditorialRepoFile(t, "charts/README.md")
	for _, want := range []string{"WithExternalizedControlRuntime(true)", "JavaScript", "accessible data table"} {
		if !strings.Contains(chartsReadme, want) {
			t.Fatalf("charts README missing %q", want)
		}
	}

	rendererReadme := readEditorialRepoFile(t, "charts/tools/optimistic-renderer/README.md")
	for _, want := range []string{"`Expand`", "capability-derived `Export`"} {
		if !strings.Contains(rendererReadme, want) {
			t.Fatalf("chart renderer README missing %q", want)
		}
	}
	if strings.Contains(rendererReadme, "fullscreen") {
		t.Fatal("chart renderer README promises fullscreen control")
	}

	testingDoc := readEditorialRepoFile(t, "docs/testing/editorial-html.md")
	for _, want := range []string{
		"tested", "not a minimum", "PDF deferred", "Manja-compatible fragment", "No duplicate runtime",
		"missing Chromium skips this tagged gate", "Untagged tests may opportunistically", "Chromium is not required for the ordinary suite.",
	} {
		if !strings.Contains(testingDoc, want) {
			t.Fatalf("testing evidence missing %q", want)
		}
	}
	for _, forbidden := range []string{
		"users must install Chrome 151", "requires Chrome 151", "minimum browser version: 151",
		"Chromium is required for the ordinary suite.",
	} {
		if strings.Contains(strings.ToLower(testingDoc), strings.ToLower(forbidden)) {
			t.Fatalf("testing evidence pins user browser policy %q", forbidden)
		}
	}
}

func TestPolicyDocumentationShowsHostOwnedNaturalIframePolicy(t *testing.T) {
	policyDoc := readEditorialRepoFile(t, "docs/policy.md")
	for _, want := range []string{
		"## Library API", "margo.Policy", "margo.WithHostPolicy(policy)",
		"margo.WithCheckPolicy(policy)", "AllowedOrigins", "cannot add an origin",
		"## Natural iframe embeds", "standard HTML", "original HTML bytes are never passed through",
	} {
		if !strings.Contains(policyDoc, want) {
			t.Fatalf("policy documentation missing %q", want)
		}
	}
}

func readEditorialRepoFile(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(name)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
