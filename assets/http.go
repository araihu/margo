package assets

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"
)

// MuambaHTTPHandler serves only the immutable files embedded in this Margo
// build. Paths are their repository-independent embedded paths, for example
// /mermaid/11.16.1/mermaid.esm.min.mjs.
func MuambaHTTPHandler() http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet && request.Method != http.MethodHead {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+request.URL.Path), "/")
		if name == "." || strings.HasSuffix(request.URL.Path, "/") {
			http.NotFound(writer, request)
			return
		}
		data, err := fs.ReadFile(muambaFiles, name)
		if err != nil {
			http.NotFound(writer, request)
			return
		}
		if strings.HasSuffix(name, ".mjs") || strings.HasSuffix(name, ".js") {
			writer.Header().Set("Content-Type", "text/javascript; charset=utf-8")
		}
		writer.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		http.ServeContent(writer, request, path.Base(name), time.Time{}, bytes.NewReader(data))
	})
}
