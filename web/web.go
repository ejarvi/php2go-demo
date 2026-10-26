// Package web holds the site's static files, built into the binary.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed static
var files embed.FS

// Assets serves /assets/<release>/<file>. The release in the path makes each deploy's
// files new to browsers and CloudFront, so they can be cached for a year. Any release
// segment is accepted, so a page cached during a deploy still gets its stylesheet.
func Assets() http.Handler {
	sub, err := fs.Sub(files, "static")
	if err != nil {
		panic(err)
	}
	fileServer := http.FileServer(http.FS(sub))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rest := strings.TrimPrefix(r.URL.Path, "/assets/")
		_, name, ok := strings.Cut(rest, "/")
		if !ok || name == "" || strings.Contains(name, "/") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		r2 := r.Clone(r.Context())
		r2.URL.Path = "/" + name
		fileServer.ServeHTTP(w, r2)
	})
}
