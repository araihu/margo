package devserver

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/araihu/margo/site"
)

func TestSnapshotCopiesArtifacts(t *testing.T) {
	source := []byte("<html><body>home</body></html>")
	snapshot := NewSnapshot(site.Result{
		Artifacts: []site.Artifact{{Path: "index.html", Content: source}},
		Site:      site.SiteManifest{BasePath: "/docs"},
	})
	source[0] = 'X'

	artifact, ok := snapshot.artifacts["index.html"]
	if !ok || !bytes.Equal(artifact, []byte("<html><body>home</body></html>")) {
		t.Fatalf("snapshot artifact = %q, present = %v", artifact, ok)
	}
}

func TestHandlerServesBasePathRoutesWithoutMutatingSnapshot(t *testing.T) {
	original := []byte("<html><body>guide</body></html>")
	store := NewSnapshotStore()
	store.Replace(NewSnapshot(site.Result{
		Artifacts: []site.Artifact{
			{Path: "index.html", Content: []byte("<html><body>home</body></html>")},
			{Path: "guide.html", Content: original},
			{Path: "margo-assets/site.css", Content: []byte("body{}")},
		},
		Site: site.SiteManifest{BasePath: "/docs"},
	}))
	handler := NewHandler(store, NewBroker())

	redirect := httptest.NewRecorder()
	handler.ServeHTTP(redirect, httptest.NewRequest(http.MethodGet, "/", nil))
	if redirect.Code != http.StatusTemporaryRedirect || redirect.Header().Get("Location") != "/docs/" {
		t.Fatalf("redirect = %d %q", redirect.Code, redirect.Header().Get("Location"))
	}

	basePathRedirect := httptest.NewRecorder()
	handler.ServeHTTP(basePathRedirect, httptest.NewRequest(http.MethodGet, "/docs", nil))
	if basePathRedirect.Code != http.StatusTemporaryRedirect || basePathRedirect.Header().Get("Location") != "/docs/" {
		t.Fatalf("base path redirect = %d %q", basePathRedirect.Code, basePathRedirect.Header().Get("Location"))
	}

	landing := httptest.NewRecorder()
	handler.ServeHTTP(landing, httptest.NewRequest(http.MethodGet, "/docs/", nil))
	if landing.Code != http.StatusOK || !strings.Contains(landing.Body.String(), "home") {
		t.Fatalf("base path landing = %d %q", landing.Code, landing.Body.String())
	}

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/docs/guide.html", nil))
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d", response.Code)
	}
	if response.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("content type = %q", response.Header().Get("Content-Type"))
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("cache control = %q", response.Header().Get("Cache-Control"))
	}
	body := response.Body.String()
	if !strings.Contains(body, liveReloadClient) || !strings.Contains(body, liveReloadPath) {
		t.Fatalf("HTML missing reload client: %s", body)
	}
	if !strings.Contains(body, liveReloadClient+"</body>") {
		t.Fatalf("reload client not injected before body close: %s", body)
	}
	if !bytes.Equal(original, []byte("<html><body>guide</body></html>")) {
		t.Fatalf("source artifact mutated: %q", original)
	}

	asset := httptest.NewRecorder()
	handler.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/docs/margo-assets/site.css", nil))
	if asset.Header().Get("Content-Type") != "text/css; charset=utf-8" || asset.Body.String() != "body{}" {
		t.Fatalf("asset = %q %q", asset.Header().Get("Content-Type"), asset.Body.String())
	}

	missing := httptest.NewRecorder()
	handler.ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/outside.txt", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing status = %d", missing.Code)
	}
}

func TestHandlerResolvesDirectoryIndex(t *testing.T) {
	store := NewSnapshotStore()
	store.Replace(NewSnapshot(site.Result{Artifacts: []site.Artifact{{Path: "guide/index.html", Content: []byte("<html><body>nested</body></html>")}}}))

	response := httptest.NewRecorder()
	NewHandler(store, NewBroker()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/guide/", nil))
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "nested") {
		t.Fatalf("response = %d %q", response.Code, response.Body.String())
	}
}

func TestHandlerEscapesInitialBuildError(t *testing.T) {
	store := NewSnapshotStore()
	store.SetError(errors.New(`site.config_invalid: bad <script>alert("x")</script>`))

	response := httptest.NewRecorder()
	NewHandler(store, NewBroker()).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/", nil))
	body := response.Body.String()
	if response.Code != http.StatusServiceUnavailable || strings.Contains(body, "<script>alert") || !strings.Contains(body, "&lt;script&gt;") {
		t.Fatalf("error response = %d %s", response.Code, body)
	}
	if !strings.Contains(body, liveReloadClient) {
		t.Fatal("initial error page cannot recover through live reload")
	}
}

func TestHandlerLeavesHTMLWithoutBodyCloseAndNonHTMLUnchanged(t *testing.T) {
	store := NewSnapshotStore()
	store.Replace(NewSnapshot(site.Result{Artifacts: []site.Artifact{
		{Path: "fragment.html", Content: []byte("<main>fragment</main>")},
		{Path: "llms.txt", Content: []byte("# docs\n")},
	}}))
	handler := NewHandler(store, NewBroker())

	for requestPath, want := range map[string]string{"/fragment.html": "<main>fragment</main>", "/llms.txt": "# docs\n"} {
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, requestPath, nil))
		if response.Body.String() != want {
			t.Fatalf("%s body = %q, want %q", requestPath, response.Body.String(), want)
		}
	}
}
