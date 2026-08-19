package devserver

import (
	"bytes"
	"fmt"
	"html"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
)

type siteHandler struct {
	store  *SnapshotStore
	broker *Broker
}

// NewHandler serves the current in-memory snapshot and live-reload stream.
func NewHandler(store *SnapshotStore, broker *Broker) http.Handler {
	if store == nil {
		store = NewSnapshotStore()
	}
	if broker == nil {
		broker = NewBroker()
	}
	return &siteHandler{store: store, broker: broker}
}

func (handler *siteHandler) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")
	if request.URL.Path == liveReloadPath {
		handler.broker.ServeHTTP(writer, request)
		return
	}
	if request.Method != http.MethodGet && request.Method != http.MethodHead {
		writer.Header().Set("Allow", "GET, HEAD")
		http.Error(writer, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	state := handler.store.load()
	if state.snapshot == nil {
		handler.serveBuildError(writer, request, state.failure)
		return
	}
	snapshot := state.snapshot
	requestPath := path.Clean("/" + strings.TrimPrefix(request.URL.Path, "/"))
	if snapshot.basePath != "" {
		if requestPath == "/" || requestPath == snapshot.basePath {
			http.Redirect(writer, request, snapshot.basePath+"/", http.StatusTemporaryRedirect)
			return
		}
		prefix := snapshot.basePath + "/"
		if !strings.HasPrefix(requestPath, prefix) {
			http.NotFound(writer, request)
			return
		}
		requestPath = strings.TrimPrefix(requestPath, snapshot.basePath)
	}
	relative := strings.TrimPrefix(requestPath, "/")
	if relative == "" || strings.HasSuffix(request.URL.Path, "/") {
		relative = path.Join(relative, "index.html")
	}
	if relative == "." || relative == ".." || strings.HasPrefix(relative, "../") || strings.ContainsRune(relative, '\\') {
		http.NotFound(writer, request)
		return
	}
	content, ok := snapshot.artifacts[relative]
	if !ok {
		http.NotFound(writer, request)
		return
	}
	handler.serveArtifact(writer, request, relative, content)
}

func (handler *siteHandler) serveArtifact(writer http.ResponseWriter, request *http.Request, name string, content []byte) {
	contentType := artifactContentType(name)
	if contentType == "text/html; charset=utf-8" {
		content = injectLiveReload(content)
	}
	writer.Header().Set("Content-Type", contentType)
	writer.Header().Set("Content-Length", strconv.Itoa(len(content)))
	writer.WriteHeader(http.StatusOK)
	if request.Method != http.MethodHead {
		_, _ = writer.Write(content)
	}
}

func (handler *siteHandler) serveBuildError(writer http.ResponseWriter, request *http.Request, failure error) {
	message := "Waiting for the first successful site build."
	if failure != nil {
		message = failure.Error()
	}
	content := []byte(fmt.Sprintf("<!doctype html><html><head><meta charset=\"utf-8\"><title>Margo development build failed</title></head><body><main><h1>Development build failed</h1><pre>%s</pre></main></body></html>", html.EscapeString(message)))
	content = injectLiveReload(content)
	writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	writer.Header().Set("Content-Length", strconv.Itoa(len(content)))
	writer.WriteHeader(http.StatusServiceUnavailable)
	if request.Method != http.MethodHead {
		_, _ = writer.Write(content)
	}
}

func injectLiveReload(content []byte) []byte {
	index := bytes.LastIndex(content, []byte("</body>"))
	if index < 0 {
		return content
	}
	result := make([]byte, 0, len(content)+len(liveReloadClient))
	result = append(result, content[:index]...)
	result = append(result, liveReloadClient...)
	result = append(result, content[index:]...)
	return result
}

func artifactContentType(name string) string {
	switch strings.ToLower(path.Ext(name)) {
	case ".html", ".htm":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js", ".mjs":
		return "text/javascript; charset=utf-8"
	case ".json":
		return "application/json"
	case ".xml":
		return "application/xml"
	case ".txt", ".md":
		return "text/plain; charset=utf-8"
	case ".svg":
		return "image/svg+xml"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	}
	if contentType := mime.TypeByExtension(path.Ext(name)); contentType != "" {
		return contentType
	}
	return "application/octet-stream"
}
