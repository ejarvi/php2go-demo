package site

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ejarvi/php2go-demo/internal/store"
)

// sign makes a value that only this site can produce. The purpose keeps one kind of
// signed value from being passed off as another.
func (a *App) sign(purpose, data string) string {
	mac := hmac.New(sha256.New, a.cfg.SessionSecret)
	mac.Write([]byte(purpose + "|" + data))
	return hex.EncodeToString(mac.Sum(nil))
}

func randomHex(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		panic(err) // the system's random source failing is unrecoverable
	}
	return hex.EncodeToString(b)
}

func setCookie(w http.ResponseWriter, name, value string, expires time.Time) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: value, Path: "/", Expires: expires,
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

func clearCookie(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name: name, Value: "", Path: "/", MaxAge: -1,
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode,
	})
}

// --- The form timer (D-011): when the form was shown, signed so it can't be faked. ---

func (a *App) formTimer() string {
	shown := strconv.FormatInt(a.now().Unix(), 10)
	return shown + "." + a.sign("timer", shown)
}

func (a *App) filledTooFast(timer string) bool {
	shown, signature, ok := strings.Cut(timer, ".")
	if !ok || !hmac.Equal([]byte(a.sign("timer", shown)), []byte(signature)) {
		return true
	}
	t, err := strconv.ParseInt(shown, 10, 64)
	return err != nil || a.now().Unix()-t < int64(a.cfg.FormMinSeconds)
}

// --- The sign-up confirmation: a short-lived signed cookie with the event and head count. ---

var confirmationPattern = regexp.MustCompile(`^(\d+)\.(\d+)\.([0-9a-f]{64})$`)

func (a *App) setConfirmation(w http.ResponseWriter, eventID, people int) {
	value := fmt.Sprintf("%d.%d", eventID, people)
	setCookie(w, "signedup", value+"."+a.sign("signedup", value), a.now().Add(10*time.Minute))
}

// takeConfirmation returns the head count of a sign-up just made for this event, if any.
func (a *App) takeConfirmation(w http.ResponseWriter, r *http.Request, eventID int) (int, bool) {
	c, err := r.Cookie("signedup")
	if err != nil {
		return 0, false
	}
	m := confirmationPattern.FindStringSubmatch(c.Value)
	if m == nil || !hmac.Equal([]byte(a.sign("signedup", m[1]+"."+m[2])), []byte(m[3])) || m[1] != strconv.Itoa(eventID) {
		return 0, false
	}
	clearCookie(w, "signedup")
	people, _ := strconv.Atoi(m[2])
	return people, true
}

// --- Sessions (design/modules/signin.md, D-023). ---

const sessionHours = 8

var sessionCodePattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

// passwordFingerprint ties a session to the password it began with, so a new password
// (and the deploy that brings it) ends every session.
func (a *App) passwordFingerprint() string {
	return a.sign("password", a.cfg.OrganizerPassword)
}

// passwordIsRight compares hashes, which takes the same time whatever is typed.
func (a *App) passwordIsRight(typed string) bool {
	want := sha256.Sum256([]byte(a.cfg.OrganizerPassword))
	got := sha256.Sum256([]byte(typed))
	return subtle.ConstantTimeCompare(want[:], got[:]) == 1
}

func (a *App) startSession(ctx context.Context, w http.ResponseWriter) error {
	code := randomHex(32)
	expires := a.now().Add(sessionHours * time.Hour)
	item := sessionItem{
		PK:          sessionPK(a.sign("session", code)),
		SK:          "-",
		Fingerprint: a.passwordFingerprint(),
		Expires:     expires.Unix(),
	}
	if err := a.st.Put(ctx, item, store.IfNew); err != nil {
		return err
	}
	setCookie(w, "session", code, expires)
	return nil
}

// sessionValid reports whether the request's session cookie names a live session that
// began with the current password.
func (a *App) sessionValid(ctx context.Context, r *http.Request) (bool, error) {
	c, err := r.Cookie("session")
	if err != nil || !sessionCodePattern.MatchString(c.Value) {
		return false, nil
	}
	var item sessionItem
	ok, err := a.st.Get(ctx, sessionPK(a.sign("session", c.Value)), "-", &item)
	if err != nil || !ok {
		return false, err
	}
	return item.Expires > a.now().Unix() && hmac.Equal([]byte(item.Fingerprint), []byte(a.passwordFingerprint())), nil
}

func (a *App) endSession(ctx context.Context, w http.ResponseWriter, r *http.Request) error {
	if c, err := r.Cookie("session"); err == nil && sessionCodePattern.MatchString(c.Value) {
		if err := a.st.Delete(ctx, sessionPK(a.sign("session", c.Value)), "-"); err != nil {
			return err
		}
	}
	clearCookie(w, "session")
	return nil
}
