package assets

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMuambaHTTPHandlerServesRuntimeAndRelativeChunks(t *testing.T) {
	handler := MuambaHTTPHandler()
	for _, path := range []string{
		"/mermaid/11.16.1/mermaid.esm.min.mjs",
		"/mermaid/11.16.1/chunks/mermaid.esm.min/flowDiagram-BWE6NHOH.mjs",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		if response.Code != http.StatusOK || response.Body.Len() == 0 {
			t.Fatalf("GET %s = %d, %d bytes", path, response.Code, response.Body.Len())
		}
		if contentType := response.Header().Get("Content-Type"); !strings.Contains(contentType, "javascript") {
			t.Fatalf("GET %s content type = %q", path, contentType)
		}
	}
}

func TestMuambaHTTPHandlerDoesNotListDirectories(t *testing.T) {
	request := httptest.NewRequest(http.MethodGet, "/mermaid/11.16.1/", nil)
	response := httptest.NewRecorder()
	MuambaHTTPHandler().ServeHTTP(response, request)
	if response.Code != http.StatusNotFound {
		t.Fatalf("directory response = %d", response.Code)
	}
}
