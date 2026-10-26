package platform

import (
	"crypto/subtle"
	"net/http"
)

// SecurityHeaders sends the headers every response carries (D-019, design/modules/site.md).
func SecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; "+
			"form-action 'self'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'")
		// same-origin, not no-referrer: with no-referrer, browsers send "Origin: null" (D-019).
		h.Set("Referrer-Policy", "same-origin")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Strict-Transport-Security", "max-age=31536000")
		next.ServeHTTP(w, r)
	})
}

// OriginHeader carries the secret CloudFront adds to every request it forwards.
const OriginHeader = "X-Php2go-Origin"

// OriginGuard refuses requests that didn't come through CloudFront, such as requests sent
// straight to the Lambda's function URL. An empty secret turns the check off, for local runs.
func OriginGuard(secret string, next http.Handler) http.Handler {
	if secret == "" {
		return next
	}
	want := []byte(secret)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if subtle.ConstantTimeCompare([]byte(r.Header.Get(OriginHeader)), want) != 1 {
			http.Error(w, "Forbidden", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	})
}
