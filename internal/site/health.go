package site

import (
	"encoding/json"
	"net/http"

	"github.com/ejarvi/php2go-demo/internal/platform"
)

// GET /health (D-021): can the site read its storage, and which release is running?
func (a *App) health(w http.ResponseWriter, r *http.Request) {
	status, code := "ok", http.StatusOK
	if err := a.st.Ping(r.Context()); err != nil {
		a.log.Error("health check failed", "err", err)
		status, code = "error", http.StatusServiceUnavailable
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": status, "release": platform.Release()})
}
