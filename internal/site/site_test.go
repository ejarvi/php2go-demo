package site

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/ejarvi/php2go-demo/internal/platform"
	"github.com/ejarvi/php2go-demo/internal/store"
)

// These tests drive the site through HTTP with the in-memory store. The shared acceptance
// tests in tests/contract/ check the same behavior against a running server.

type client struct {
	t       *testing.T
	h       http.Handler
	cookies map[string]string
}

type response struct {
	status   int
	location string
	body     string
	header   http.Header
}

func newTestSite(t *testing.T) http.Handler {
	t.Helper()
	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	h, err := New(Options{
		Store: store.NewMemory(),
		Config: platform.Config{
			ClubName: "Test Club", Location: loc,
			OrganizerPassword: "right-password", SessionSecret: []byte("a-test-session-secret-0123456789"),
			FormMinSeconds: 3,
		},
		Log: slog.New(slog.NewTextHandler(io.Discard, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	return h
}

func (c *client) do(method, path string, form url.Values, headers map[string]string) response {
	c.t.Helper()
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req := httptest.NewRequest(method, "http://site.test"+path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Sec-Fetch-Site", "same-origin")
		req.Header.Set("Origin", "http://site.test")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	for name, value := range c.cookies {
		req.AddCookie(&http.Cookie{Name: name, Value: value})
	}
	rec := httptest.NewRecorder()
	c.h.ServeHTTP(rec, req)
	for _, ck := range rec.Result().Cookies() {
		if ck.MaxAge < 0 || ck.Value == "" {
			delete(c.cookies, ck.Name)
		} else {
			c.cookies[ck.Name] = ck.Value
		}
	}
	return response{rec.Code, rec.Header().Get("Location"), rec.Body.String(), rec.Header()}
}

func (c *client) get(path string) response { return c.do("GET", path, nil, nil) }
func (c *client) post(path string, form url.Values) response {
	return c.do("POST", path, form, nil)
}

func newClient(t *testing.T, h http.Handler) *client {
	return &client{t: t, h: h, cookies: map[string]string{}}
}

func signedIn(t *testing.T, h http.Handler) *client {
	c := newClient(t, h)
	if r := c.post("/signin", url.Values{"password": {"right-password"}}); r.status != http.StatusSeeOther || r.location != "/admin" {
		t.Fatalf("sign-in: %d %q", r.status, r.location)
	}
	return c
}

var idPattern = regexp.MustCompile(`^/admin/events/(\d+)`)

func createEvent(t *testing.T, c *client, fields url.Values) string {
	t.Helper()
	form := url.Values{
		"title": {"Park clean-up"}, "date": {time.Now().AddDate(0, 0, 7).Format("2006-01-02")},
		"start_time": {"18:00"}, "location": {"Hall"}, "published": {"1"},
	}
	for k, v := range fields {
		form[k] = v
	}
	r := c.post("/admin/events", form)
	m := idPattern.FindStringSubmatch(r.location)
	if r.status != http.StatusSeeOther || m == nil {
		t.Fatalf("creating an event: %d %q\n%s", r.status, r.location, r.body)
	}
	return m[1]
}

func TestHomeAndHeaders(t *testing.T) {
	c := newClient(t, newTestSite(t))
	r := c.get("/")
	if r.status != http.StatusOK || !strings.Contains(r.body, "No events are planned right now.") {
		t.Fatalf("home: %d\n%s", r.status, r.body)
	}
	if got := r.header.Get("Referrer-Policy"); got != "same-origin" {
		t.Errorf("Referrer-Policy %q", got)
	}
	if !strings.Contains(r.header.Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Error("CSP missing frame-ancestors")
	}
	if r := c.get("/no-such-page"); r.status != http.StatusNotFound || !strings.Contains(r.body, "<footer") {
		t.Errorf("404 page: %d", r.status)
	}
	if r := c.get("/health"); r.status != http.StatusOK || !strings.Contains(r.body, `"status":"ok"`) {
		t.Errorf("health: %d %s", r.status, r.body)
	}
}

func TestAdminNeedsSession(t *testing.T) {
	c := newClient(t, newTestSite(t))
	if r := c.get("/admin/events/new"); r.status != http.StatusSeeOther || !strings.HasPrefix(r.location, "/signin") {
		t.Errorf("GET without session: %d %q", r.status, r.location)
	}
	if r := c.post("/admin/events", url.Values{"title": {"x"}}); r.status != http.StatusForbidden {
		t.Errorf("POST without session: %d", r.status)
	}
	if r := c.post("/signin", url.Values{"password": {"wrong"}}); r.status != http.StatusUnprocessableEntity || c.cookies["session"] != "" {
		t.Errorf("wrong password: %d", r.status)
	}
}

func TestSignupFlow(t *testing.T) {
	h := newTestSite(t)
	admin := signedIn(t, h)
	id := createEvent(t, admin, url.Values{"capacity": {"10"}})

	visitor := newClient(t, h)
	page := visitor.get("/events/" + id)
	timer := regexp.MustCompile(`name="timer" value="([^"]+)"`).FindStringSubmatch(page.body)
	if timer == nil {
		t.Fatalf("no timer on the event page:\n%s", page.body)
	}
	// Filled in instantly: treated as a bot (D-011), so nothing is saved.
	r := visitor.post("/events/"+id+"/signup", url.Values{"name": {"Ana"}, "email": {"ana@example.com"}, "people": {"3"}, "timer": {timer[1]}})
	if r.status != http.StatusSeeOther {
		t.Fatalf("too-fast sign-up: %d\n%s", r.status, r.body)
	}
	if strings.Contains(admin.get("/admin/events/"+id).body, "ana@example.com") {
		t.Fatal("a sign-up filled in too fast was saved")
	}
}

func TestCrossOriginRefused(t *testing.T) {
	h := newTestSite(t)
	c := newClient(t, h)
	r := c.do("POST", "/signin", url.Values{"password": {"right-password"}},
		map[string]string{"Sec-Fetch-Site": "cross-site", "Origin": "https://attacker.example"})
	if r.status != http.StatusForbidden || c.cookies["session"] != "" {
		t.Fatalf("cross-site sign-in: %d", r.status)
	}
}

func TestEventValidationKeepsInput(t *testing.T) {
	admin := signedIn(t, newTestSite(t))
	r := admin.post("/admin/events", url.Values{
		"title": {"Backwards"}, "date": {"2030-05-01"}, "start_time": {"18:00"}, "end_time": {"17:00"}, "location": {"Hall"},
	})
	if r.status != http.StatusUnprocessableEntity || !strings.Contains(r.body, `value="Backwards"`) ||
		!strings.Contains(r.body, "The end time must be after the start time.") {
		t.Fatalf("validation: %d\n%s", r.status, r.body)
	}
}

func TestDescriptionIsText(t *testing.T) {
	got := string(descriptionHTML("Hi <script>x</script>\nSee https://example.org."))
	if strings.Contains(got, "<script>") || !strings.Contains(got, `<a href="https://example.org"`) || !strings.Contains(got, "<br>") {
		t.Errorf("descriptionHTML: %s", got)
	}
}
