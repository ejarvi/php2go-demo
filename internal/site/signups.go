package site

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/ejarvi/php2go-demo/internal/store"
)

// design/modules/signups.md

type signupForm struct {
	Name, Email, People string
}

func signupFormFrom(r *http.Request) signupForm {
	return signupForm{
		Name:   strings.TrimSpace(r.PostFormValue("name")),
		Email:  strings.TrimSpace(r.PostFormValue("email")),
		People: strings.TrimSpace(r.PostFormValue("people")),
	}
}

func (f signupForm) validate() map[string]string {
	errs := map[string]string{}
	switch {
	case f.Name == "":
		errs["name"] = "Enter your name."
	case utf8.RuneCountInString(f.Name) > 100:
		errs["name"] = "Keep your name to 100 characters or fewer."
	}
	switch {
	case f.Email == "":
		errs["email"] = "Enter your email address."
	case utf8.RuneCountInString(f.Email) > 254 || !looksLikeEmail(f.Email):
		errs["email"] = "Enter an email address like name@example.com."
	}
	if n, err := strconv.Atoi(f.People); err != nil || n < 1 || n > 6 {
		errs["people"] = "Enter a number from 1 to 6."
	}
	return errs
}

// looksLikeEmail accepts a plain address with a dot in its domain, like name@example.com.
func looksLikeEmail(s string) bool {
	addr, err := mail.ParseAddress(s)
	if err != nil || addr.Address != s {
		return false
	}
	_, domain, _ := strings.Cut(s, "@")
	return strings.Contains(domain, ".")
}

func (a *App) signupCreate(w http.ResponseWriter, r *http.Request) {
	id, ok := a.pathID(w, r)
	if !ok {
		return
	}
	ctx := r.Context()

	// 1. The event exists, is published, and hasn't started.
	e, found, err := a.findEvent(ctx, id)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	if !found || !e.item.Published {
		a.notFound(w, r)
		return
	}
	form := signupFormFrom(r)
	if !e.Start.After(a.now()) {
		a.renderEventPage(w, r, e, form, nil, "Sign-ups for this event have closed.", http.StatusConflict)
		return
	}

	// 2. The fields are valid.
	if errs := form.validate(); len(errs) > 0 {
		a.renderEventPage(w, r, e, form, errs, "", http.StatusUnprocessableEntity)
		return
	}
	people, _ := strconv.Atoi(form.People)

	// 3. The trap field is empty, and the form wasn't filled in too fast (D-011).
	// A bot gets the normal success page, but nothing is saved.
	if r.PostFormValue("website") != "" || a.filledTooFast(r.PostFormValue("timer")) {
		a.setConfirmation(w, id, people)
		redirect(w, r, fmt.Sprintf("/events/%d", id))
		return
	}

	// 4. The head count fits in the places left (D-010).
	problem, err := a.saveSignup(ctx, id, form, people)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	if problem != "" {
		if fresh, ok, _ := a.findEvent(ctx, id); ok {
			e = fresh
		}
		a.renderEventPage(w, r, e, form, nil, problem, http.StatusConflict)
		return
	}
	a.setConfirmation(w, id, people)
	redirect(w, r, fmt.Sprintf("/events/%d", id))
}

// saveSignup saves a sign-up, or replaces the head count of an earlier one with the same
// email. It returns the message to show when there aren't enough places. Every change
// rewrites the event with a new version, so two sign-ups at once can't overfill it.
func (a *App) saveSignup(ctx context.Context, eventID int, form signupForm, people int) (string, error) {
	var problem string
	err := retry(func() error {
		problem = ""
		e, found, err := a.findEvent(ctx, eventID)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("event %d disappeared", eventID)
		}
		signups, err := a.eventSignups(ctx, eventID)
		if err != nil {
			return err
		}
		var earlier *signupItem
		for i := range signups {
			if emailKey(signups[i].Email) == emailKey(form.Email) {
				earlier = &signups[i]
			}
		}
		already := 0
		if earlier != nil {
			already = earlier.People
		}
		if e.item.Capacity > 0 {
			left := max(0, e.item.Capacity-e.item.People+already)
			if people > left {
				switch left {
				case 0:
					problem = "Sorry, this event is now full."
				case 1:
					problem = "Sorry, only 1 place is left."
				default:
					problem = fmt.Sprintf("Sorry, only %d places are left.", left)
				}
				return nil
			}
		}

		ev := e.item
		ev.People = ev.People - already + people
		ev.Version++
		ops := []store.Op{store.PutOp(ev, store.IfVersion(e.item.Version))}
		if earlier != nil {
			updated := *earlier
			updated.Name = form.Name
			updated.People = people
			ops = append(ops, store.PutOp(updated, store.Always))
		} else {
			counter, id, err := a.nextID(ctx, "signup")
			if err != nil {
				return err
			}
			ops = append(ops, counter, store.PutOp(signupItem{
				PK: pkSite, SK: signupSK(eventID, id), ID: id, EventID: eventID,
				Name: form.Name, Email: form.Email, People: people,
				CreatedAt: a.now().Format(timeLayout), Expires: signupExpiry(e),
			}, store.IfNew))
		}
		return a.st.Transact(ctx, ops...)
	})
	return problem, err
}
