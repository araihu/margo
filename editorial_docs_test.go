package margo

import (
	"os"
	"strings"
	"testing"
)

func TestHTMLDocumentationNamesDecoupledPublicContract(t *testing.T) {
	readme := readEditorialRepoFile(t, "README.md")
	for _, want := range []string{
		"RenderHTML", "RenderHTMLPage", "HTMLAssetHandler", "webpublication.Render",
		"/assets/", "/margo-assets/", "/charts/assets/",
		"HTMLDependenciesLocal", "HTMLDependenciesInline",
	} {
		if !strings.Contains(readme, want) {
			t.Fatalf("README missing %q", want)
		}
	}
	for _, forbidden := range []string{"RenderEditorial", "margo.RenderPublication", "margo.PublicationInput", "EditorialAssetHandler"} {
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

	testingDoc := readEditorialRepoFile(t, "docs/testing/editorial-html.md")
	for _, want := range []string{"tested", "not a minimum", "PDF deferred", "Manja-compatible fragment", "No duplicate runtime"} {
		if !strings.Contains(testingDoc, want) {
			t.Fatalf("testing evidence missing %q", want)
		}
	}
	for _, forbidden := range []string{"users must install Chrome 151", "requires Chrome 151", "minimum browser version: 151"} {
		if strings.Contains(strings.ToLower(testingDoc), strings.ToLower(forbidden)) {
			t.Fatalf("testing evidence pins user browser policy %q", forbidden)
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
