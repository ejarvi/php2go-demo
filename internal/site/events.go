package site

import (
	"net/http"
)

// design/modules/events.md

func (a *App) eventsIndex(w http.ResponseWriter, r *http.Request) {
	events, err := a.allEvents(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	today := a.today()
	var upcoming []eventView
	for _, e := range events {
		if e.item.Published && !e.Start.Before(today) {
			upcoming = append(upcoming, a.view(e))
		}
	}
	a.render(w, r, http.StatusOK, "events_index", map[string]any{"Title": "Upcoming events", "Events": upcoming})
}

func (a *App) eventShow(w http.ResponseWriter, r *http.Request) {
	id, ok := a.pathID(w, r)
	if !ok {
		return
	}
	e, found, err := a.findEvent(r.Context(), id)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	if !found || (!e.item.Published && !isOrganizer(r)) {
		a.notFound(w, r)
		return
	}
	a.renderEventPage(w, r, e, signupForm{People: "1"}, nil, "", http.StatusOK)
}

// renderEventPage shows an event, also when a sign-up needs fixing.
func (a *App) renderEventPage(w http.ResponseWriter, r *http.Request, e event, form signupForm, errs map[string]string, notice string, status int) {
	v := a.view(e)
	data := map[string]any{
		"Title":     e.item.Title,
		"Event":     v,
		"Form":      form,
		"Errors":    errs,
		"Notice":    notice,
		"Timer":     a.formTimer(),
		"CanSignUp": v.Published && !v.Started && !v.Full,
	}
	if people, ok := a.takeConfirmation(w, r, e.item.ID); ok {
		data["Confirmed"] = people
	}
	a.render(w, r, status, "event_show", data)
}
