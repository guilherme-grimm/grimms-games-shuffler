package http

import (
	"io/fs"
	"net/http"
)

// _placeholder is served when no frontend build is embedded (dev binaries
// built without `pnpm build`).
const _placeholder = `<!doctype html>
<meta charset="utf-8">
<title>GGS</title>
<style>
  body { background:#000; color:#33ff33; font-family:monospace;
         display:grid; place-items:center; height:100vh; margin:0 }
</style>
<pre>
  GGS :: GRIMM'S GAMES SHUFFLER
  NO FRONTEND BUILD EMBEDDED
  RUN: cd web && pnpm build
</pre>`

// spaHandler serves the embedded frontend, falling back to index.html for
// client-side routes and to a placeholder page when no build is embedded.
func (s *Server) spaHandler() http.Handler {
	index, indexErr := fs.ReadFile(s.dist, "index.html")
	fileServer := http.FileServerFS(s.dist)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if indexErr != nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			_, _ = w.Write([]byte(_placeholder))
			return
		}
		if r.URL.Path != "/" {
			if _, err := fs.Stat(s.dist, r.URL.Path[1:]); err == nil {
				fileServer.ServeHTTP(w, r)
				return
			}
		}
		// SPA fallback: unknown paths get index.html, router takes over.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(index)
	})
}
