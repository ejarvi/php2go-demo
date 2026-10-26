package site

import (
	"net/http"
	"regexp"
	"time"
)

// design/modules/signin.md

var nextPattern = regexp.MustCompile(`^/admin(/[A-Za-z0-9/._-]*)?$`)

// safeNext only ever sends an organizer on to a page in the organizers' area.
func safeNext(next string) string {
	if nextPattern.MatchString(next) {
		return next
	}
	return "/admin"
}

func (a *App) signinForm(w http.ResponseWriter, r *http.Request) {
	if isOrganizer(r) {
		redirect(w, r, "/admin")
		return
	}
	a.render(w, r, http.StatusOK, "signin", map[string]any{
		"Title": "Organizers", "Next": safeNext(r.URL.Query().Get("next")),
	})
}

func (a *App) signinSubmit(w http.ResponseWriter, r *http.Request) {
	next := safeNext(r.PostFormValue("next"))
	if !a.passwordIsRight(r.PostFormValue("password")) {
		// Answer slowly, to slow down guessing.
		time.Sleep(time.Second)
		a.render(w, r, http.StatusUnprocessableEntity, "signin", map[string]any{
			"Title": "Organizers", "Next": next, "Error": "That password isn't right.",
		})
		return
	}
	if err := a.startSession(r.Context(), w); err != nil {
		a.serverError(w, r, err)
		return
	}
	redirect(w, r, next)
}

func (a *App) signout(w http.ResponseWriter, r *http.Request) {
	if err := a.endSession(r.Context(), w, r); err != nil {
		a.serverError(w, r, err)
		return
	}
	redirect(w, r, "/")
}
