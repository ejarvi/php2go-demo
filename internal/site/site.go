// Package site is the club's events site: every page and form in design/pages.md.
// Each module in design/modules/ has its own file here.
package site

import (
	"bytes"
	"context"
	"embed"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ejarvi/php2go-demo/internal/platform"
	"github.com/ejarvi/php2go-demo/internal/store"
	"github.com/ejarvi/php2go-demo/web"
)

//go:embed templates
var templateFiles embed.FS

// Options is what New needs.
type Options struct {
	Store  store.Store
	Config platform.Config
	Log    *slog.Logger
	Now    func() time.Time // for tests; defaults to time.Now
}

// App serves the site.
type App struct {
	st    store.Store
	cfg   platform.Config
	log   *slog.Logger
	clock func() time.Time
	pages map[string]*template.Template
}

// The pages, each rendered inside layout.html.
var pageNames = []string{"events_index", "event_show", "signin", "admin_index", "admin_event_form", "admin_event_delete", "error"}

// New returns the site's HTTP handler, with every shared check in front of the pages.
func New(o Options) (http.Handler, error) {
	a := &App{st: o.Store, cfg: o.Config, log: o.Log, clock: o.Now, pages: map[string]*template.Template{}}
	if a.clock == nil {
		a.clock = time.Now
	}
	if a.log == nil {
		a.log = slog.Default()
	}
	for _, name := range pageNames {
		t, err := template.New("").Funcs(a.funcs()).ParseFS(templateFiles, "templates/layout.html", "templates/"+name+".html")
		if err != nil {
			return nil, fmt.Errorf("parsing %s: %w", name, err)
		}
		a.pages[name] = t
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", a.eventsIndex)
	mux.HandleFunc("GET /events/{id}", a.eventShow)
	mux.HandleFunc("POST /events/{id}/signup", a.signupCreate)
	mux.HandleFunc("GET /signin", a.signinForm)
	mux.HandleFunc("POST /signin", a.signinSubmit)
	mux.HandleFunc("POST /signout", a.signout)
	mux.HandleFunc("GET /admin", a.adminIndex)
	mux.HandleFunc("GET /admin/events/new", a.adminEventNew)
	mux.HandleFunc("POST /admin/events", a.adminEventCreate)
	mux.HandleFunc("GET /admin/events/{id}", a.adminEventEdit)
	mux.HandleFunc("POST /admin/events/{id}", a.adminEventUpdate)
	mux.HandleFunc("POST /admin/events/{id}/delete", a.adminEventDelete)
	mux.HandleFunc("GET /admin/events/{id}/signups.csv", a.adminSignupsCSV)
	mux.HandleFunc("POST /admin/signups/{id}/delete", a.adminSignupDelete)
	mux.HandleFunc("GET /health", a.health)
	mux.Handle("GET /assets/", web.Assets())
	mux.HandleFunc("/", a.notFound)

	// Forms accept only same-origin requests (D-025). Go's built-in protection reads the
	// browser's Sec-Fetch-Site header, or compares Origin with the host, and needs no tokens.
	sameOrigin := http.NewCrossOriginProtection()
	if a.cfg.SiteURL != "" {
		if err := sameOrigin.AddTrustedOrigin(a.cfg.SiteURL); err != nil {
			return nil, fmt.Errorf("SITE_URL: %w", err)
		}
	}
	sameOrigin.SetDenyHandler(http.HandlerFunc(a.crossOrigin))

	var h http.Handler = mux
	h = a.adminGuard(h)
	h = sameOrigin.Handler(h)
	h = a.withSession(h)
	h = a.recoverPanics(h)
	h = platform.SecurityHeaders(h)
	return h, nil
}

func (a *App) now() time.Time { return a.clock().In(a.cfg.Location) }

// today is the start of the current day in the club's time zone.
func (a *App) today() time.Time {
	y, m, d := a.now().Date()
	return time.Date(y, m, d, 0, 0, 0, 0, a.cfg.Location)
}

// --- Middleware ---

type ctxKey int

const organizerKey ctxKey = 0

// withSession looks up the session once per request.
func (a *App) withSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ok, err := a.sessionValid(r.Context(), r)
		if err != nil {
			a.log.Error("checking the session", "err", err)
		}
		next.ServeHTTP(w, r.WithContext(context.WithValue(r.Context(), organizerKey, ok)))
	})
}

func isOrganizer(r *http.Request) bool {
	ok, _ := r.Context().Value(organizerKey).(bool)
	return ok
}

// adminGuard: any /admin path needs a session. Pages redirect to sign in; changes are refused.
func (a *App) adminGuard(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if (p == "/admin" || strings.HasPrefix(p, "/admin/")) && !isOrganizer(r) {
			if r.Method == http.MethodGet || r.Method == http.MethodHead {
				http.Redirect(w, r, "/signin?next="+url.QueryEscape(p), http.StatusSeeOther)
				return
			}
			a.errorPage(w, r, http.StatusForbidden, "Please sign in",
				"Your session has ended. Sign in again, then repeat the change.", "/signin", "Sign in")
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (a *App) recoverPanics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if v := recover(); v != nil {
				if v == http.ErrAbortHandler {
					panic(v)
				}
				a.log.Error("panic", "path", r.URL.Path, "panic", fmt.Sprint(v))
				a.serverError(w, r, nil)
			}
		}()
		next.ServeHTTP(w, r)
	})
}

// --- Rendering ---

// render writes a page inside the layout. It renders into a buffer first, so a template
// error becomes a clean 500 page instead of half a page.
func (a *App) render(w http.ResponseWriter, r *http.Request, status int, page string, data map[string]any) {
	data["ClubName"] = a.cfg.ClubName
	data["Organizer"] = isOrganizer(r)
	data["Release"] = platform.Release()
	var buf bytes.Buffer
	if err := a.pages[page].ExecuteTemplate(&buf, "layout", data); err != nil {
		a.log.Error("rendering", "page", page, "err", err)
		if page != "error" {
			a.serverError(w, r, nil)
		}
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)
	_, _ = buf.WriteTo(w)
}

func (a *App) errorPage(w http.ResponseWriter, r *http.Request, status int, title, message, linkHref, linkText string) {
	a.render(w, r, status, "error", map[string]any{
		"Title": title, "Message": message, "LinkHref": linkHref, "LinkText": linkText,
	})
}

func (a *App) notFound(w http.ResponseWriter, r *http.Request) {
	a.errorPage(w, r, http.StatusNotFound, "Page not found", "We couldn't find that page.", "/", "See upcoming events")
}

// serverError logs the details, never to the page and never personal data (design/modules/site.md).
func (a *App) serverError(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		a.log.Error("request failed", "path", r.URL.Path, "err", err)
	}
	a.errorPage(w, r, http.StatusInternalServerError, "Something went wrong",
		"Something went wrong. Please try again.", "/", "Go to the home page")
}

// crossOrigin answers a form post from another origin (D-025).
func (a *App) crossOrigin(w http.ResponseWriter, r *http.Request) {
	a.errorPage(w, r, http.StatusForbidden, "Please try again",
		"This form came from another site, or the page had expired. Go back, reload the page and try again.",
		backPath(r), "Go back to the form")
}

// backPath is the page the visitor came from, if it was on this site.
func backPath(r *http.Request) string {
	ref, err := url.Parse(r.Referer())
	if err != nil || ref.Host == "" || !strings.EqualFold(ref.Host, r.Host) || ref.Path == "" {
		return "/"
	}
	return ref.Path
}

// pathID reads a numeric id from the path, or answers 404.
func (a *App) pathID(w http.ResponseWriter, r *http.Request) (int, bool) {
	id, err := strconv.Atoi(r.PathValue("id"))
	if err != nil || id <= 0 {
		a.notFound(w, r)
		return 0, false
	}
	return id, true
}

func redirect(w http.ResponseWriter, r *http.Request, to string) {
	http.Redirect(w, r, to, http.StatusSeeOther)
}
