package margo

import (
	"bytes"
	"context"
	"testing"
)

func TestHeadOwnerSelectionIsFrozenBeforeSocialTask(t *testing.T) {
	selection := FrozenHeadOwnerSelection()
	if selection.SchemaVersion != "margo/head-owner-selection/v1" {
		t.Fatalf("schemaVersion = %q", selection.SchemaVersion)
	}
	if selection.Owner != "goshtoso" && selection.Owner != "margo" {
		t.Fatalf("unexpected owner %q", selection.Owner)
	}
	if selection.Primitive != "head.Metadata" && selection.Primitive != "socialMetadataTags" {
		t.Fatalf("unexpected primitive %q", selection.Primitive)
	}
	if selection.APISourcePath == "" || len(selection.APISourceSHA256) != 64 {
		t.Fatalf("incomplete API evidence: %#v", selection)
	}
	if err := selection.Validate(); err != nil {
		t.Fatalf("frozen selection invalid: %v", err)
	}
}

func TestHeadOwnerSelectionRejectsUnknownAndTrailingFields(t *testing.T) {
	valid := `{"schemaVersion":"margo/head-owner-selection/v1","owner":"margo","primitive":"socialMetadataTags","goshtosoCommit":"module:v0.1.2","goshtosoTree":"module-cache:v0.1.2","apiSourcePath":"components/head/component.go","apiSourceSHA256":"833562eafa47d917587c21e300d28c45006b855a569266b96041123ca870b3fb"}`
	if _, err := ParseHeadOwnerSelection([]byte(valid + ` {"extra":true}`)); err == nil {
		t.Fatal("trailing JSON unexpectedly accepted")
	}
	if _, err := ParseHeadOwnerSelection([]byte(`{"schemaVersion":"margo/head-owner-selection/v1","owner":"margo","primitive":"socialMetadataTags","goshtosoCommit":"module:v0.1.2","goshtosoTree":"module-cache:v0.1.2","apiSourcePath":"components/head/component.go","apiSourceSHA256":"833562eafa47d917587c21e300d28c45006b855a569266b96041123ca870b3fb","extra":true}`)); err == nil {
		t.Fatal("unknown selection field unexpectedly accepted")
	}
}

func TestStandaloneIsOfflineDeterministicAndScoped(t *testing.T) {
	result := mustRenderSource(t, "# Standalone\n\ncontent")
	component, err := RenderStandalone(result, WithPageTitle("Standalone"), WithTheme(ThemeMinimal))
	if err != nil {
		t.Fatal(err)
	}
	var got bytes.Buffer
	if err := component.Render(context.Background(), &got); err != nil {
		t.Fatal(err)
	}
	html := got.String()
	for _, want := range []string{
		"<!doctype html>",
		`data-margo-render-instance="ri-00000000"`,
		`class="goshtoso-document"`,
		"Standalone",
		"--document-font-body",
	} {
		if !bytes.Contains(got.Bytes(), []byte(want)) {
			t.Fatalf("standalone HTML missing %q:\n%s", want, html)
		}
	}
	if bytes.Contains(got.Bytes(), []byte("http://")) || bytes.Contains(got.Bytes(), []byte("https://")) {
		t.Fatalf("offline standalone unexpectedly contains network URLs:\n%s", html)
	}
}
