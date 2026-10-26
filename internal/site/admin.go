package site

import (
	"context"
	"encoding/csv"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/ejarvi/php2go-demo/internal/store"
)

// design/modules/admin.md. adminGuard has already checked for a session.

func (a *App) adminIndex(w http.ResponseWriter, r *http.Request) {
	events, err := a.allEvents(r.Context())
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	activity, err := a.recentActivity(r.Context(), 20)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	today := a.today()
	var upcoming, past []eventView
	for _, e := range events {
		if e.Start.Before(today) {
			past = append([]eventView{a.view(e)}, past...) // most recent first
		} else {
			upcoming = append(upcoming, a.view(e))
		}
	}
	a.render(w, r, http.StatusOK, "admin_index", map[string]any{
		"Title": "Admin", "Upcoming": upcoming, "Past": past, "Activity": activity,
	})
}

// --- The event form ---

type eventForm struct {
	Title, Date, StartTime, EndTime, Location, Description, Capacity string
	Published                                                         bool
}

func eventFormFrom(r *http.Request) eventForm {
	return eventForm{
		Title:       strings.TrimSpace(r.PostFormValue("title")),
		Date:        strings.TrimSpace(r.PostFormValue("date")),
		StartTime:   strings.TrimSpace(r.PostFormValue("start_time")),
		EndTime:     strings.TrimSpace(r.PostFormValue("end_time")),
		Location:    strings.TrimSpace(r.PostFormValue("location")),
		Description: strings.TrimSpace(strings.ReplaceAll(r.PostFormValue("description"), "\r\n", "\n")),
		Capacity:    strings.TrimSpace(r.PostFormValue("capacity")),
		Published:   r.PostFormValue("published") == "1",
	}
}

func eventFormOf(e event) eventForm {
	f := eventForm{
		Title:       e.item.Title,
		Date:        e.Start.Format("2006-01-02"),
		StartTime:   e.Start.Format("15:04"),
		Location:    e.item.Location,
		Description: e.item.Description,
		Published:   e.item.Published,
	}
	if !e.End.IsZero() {
		f.EndTime = e.End.Format("15:04")
	}
	if e.item.Capacity > 0 {
		f.Capacity = strconv.Itoa(e.item.Capacity)
	}
	return f
}

var hhmm = regexp.MustCompile(`^\d{2}:\d{2}$`)

// validate checks the form, and fills in the event's fields when it's valid.
func (a *App) validate(f eventForm, it *eventItem) map[string]string {
	errs := map[string]string{}
	switch n := utf8.RuneCountInString(f.Title); {
	case n == 0:
		errs["title"] = "Enter a title."
	case n > 120:
		errs["title"] = "Keep the title to 120 characters or fewer."
	}

	start, startOK := a.parseLocal(f.Date, f.StartTime)
	_, dateOK := a.parseLocal(f.Date, "00:00")
	switch {
	case f.Date == "" || !dateOK:
		errs["date"] = "Enter a date."
	case !startOK && hhmm.MatchString(f.StartTime):
		errs["start_time"] = "That time doesn't exist on this date, because the clocks change. Choose another time."
	case !startOK:
		errs["start_time"] = "Enter a start time."
	}

	var end time.Time
	if f.EndTime != "" {
		var endOK bool
		end, endOK = a.parseLocal(f.Date, f.EndTime)
		switch {
		case !endOK:
			errs["end_time"] = "Enter the end time as hours and minutes, or leave it empty."
		case startOK && !end.After(start):
			errs["end_time"] = "The end time must be after the start time."
		}
	}

	switch n := utf8.RuneCountInString(f.Location); {
	case n == 0:
		errs["location"] = "Enter where the event is."
	case n > 200:
		errs["location"] = "Keep the location to 200 characters or fewer."
	}
	if utf8.RuneCountInString(f.Description) > 5000 {
		errs["description"] = "Keep the description to 5,000 characters or fewer."
	}
	capacity := 0
	if f.Capacity != "" {
		n, err := strconv.Atoi(f.Capacity)
		if err != nil || n < 1 || n > 500 {
			errs["capacity"] = "Enter a number from 1 to 500, or leave it empty for no limit."
		}
		capacity = n
	}
	if len(errs) > 0 {
		return errs
	}

	it.Title = f.Title
	it.Description = f.Description
	it.StartsAt = start.Format(timeLayout)
	it.EndsAt = ""
	if f.EndTime != "" {
		it.EndsAt = end.Format(timeLayout)
	}
	it.Location = f.Location
	it.Capacity = capacity
	it.Published = f.Published
	it.UpdatedAt = a.now().Format(timeLayout)
	return nil
}

// parseLocal reads a date and time in the club's time zone. It refuses times that don't
// exist, such as February 30, or 2:30 AM on the night the clocks go forward.
func (a *App) parseLocal(date, clock string) (time.Time, bool) {
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+clock, a.cfg.Location)
	if err != nil || t.Format("2006-01-02 15:04") != date+" "+clock {
		return time.Time{}, false
	}
	return t, true
}

func (a *App) adminEventNew(w http.ResponseWriter, r *http.Request) {
	a.renderEventForm(w, r, nil, eventForm{}, nil, http.StatusOK)
}

func (a *App) adminEventCreate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	form := eventFormFrom(r)
	var it eventItem
	if errs := a.validate(form, &it); errs != nil {
		a.renderEventForm(w, r, nil, form, errs, http.StatusUnprocessableEntity)
		return
	}
	var id int
	err := retry(func() error {
		counter, next, err := a.nextID(ctx, "event")
		if err != nil {
			return err
		}
		id = next
		it.PK, it.SK, it.ID, it.Version = pkSite, eventSK(id), id, 1
		return a.st.Transact(ctx, counter, store.PutOp(it, store.IfNew))
	})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	a.record(ctx, "Event created", id, it.Title)
	if it.Published {
		a.record(ctx, "Event published", id, it.Title)
	}
	redirect(w, r, fmt.Sprintf("/admin/events/%d?saved=1", id))
}

func (a *App) adminEventEdit(w http.ResponseWriter, r *http.Request) {
	e, ok := a.eventOr404(w, r)
	if !ok {
		return
	}
	a.renderEventForm(w, r, &e, eventFormOf(e), nil, http.StatusOK)
}

func (a *App) adminEventUpdate(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	e, ok := a.eventOr404(w, r)
	if !ok {
		return
	}
	form := eventFormFrom(r)
	if errs := a.validate(form, &eventItem{}); errs != nil {
		a.renderEventForm(w, r, &e, form, errs, http.StatusUnprocessableEntity)
		return
	}
	var saved event
	err := retry(func() error {
		current, found, err := a.findEvent(ctx, e.item.ID)
		if err != nil || !found {
			return err
		}
		it := current.item
		a.validate(form, &it)
		it.Version++
		if err := a.st.Put(ctx, it, store.IfVersion(current.item.Version)); err != nil {
			return err
		}
		saved = a.toEvent(it)
		return nil
	})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	if saved.item.ID == 0 {
		a.notFound(w, r)
		return
	}
	// A new date moves when the sign-ups are deleted (D-004).
	if !saved.lastMoment().Equal(e.lastMoment()) {
		if err := a.updateSignupExpiry(ctx, saved); err != nil {
			a.serverError(w, r, err)
			return
		}
	}
	a.record(ctx, "Event updated", saved.item.ID, saved.item.Title)
	if saved.item.Published != e.item.Published {
		action := "Event unpublished"
		if saved.item.Published {
			action = "Event published"
		}
		a.record(ctx, action, saved.item.ID, saved.item.Title)
	}
	redirect(w, r, fmt.Sprintf("/admin/events/%d?saved=1", saved.item.ID))
}

func (a *App) updateSignupExpiry(ctx context.Context, e event) error {
	signups, err := a.eventSignups(ctx, e.item.ID)
	if err != nil {
		return err
	}
	for _, s := range signups {
		s.Expires = signupExpiry(e)
		if err := a.st.Put(ctx, s, store.Always); err != nil {
			return err
		}
	}
	return nil
}

func (a *App) renderEventForm(w http.ResponseWriter, r *http.Request, e *event, form eventForm, errs map[string]string, status int) {
	data := map[string]any{"Title": "New event", "Form": form, "Errors": errs, "Action": "/admin/events"}
	if e != nil {
		signups, err := a.eventSignups(r.Context(), e.item.ID)
		if err != nil {
			a.serverError(w, r, err)
			return
		}
		total := 0
		for _, s := range signups {
			total += s.People
		}
		data["Title"] = "Edit event"
		data["Event"] = a.view(*e)
		data["Action"] = fmt.Sprintf("/admin/events/%d", e.item.ID)
		data["Signups"] = signups
		data["Total"] = total
		data["Saved"] = r.URL.Query().Get("saved") == "1" && len(errs) == 0
		data["OverCapacity"] = e.item.Capacity > 0 && e.item.People > e.item.Capacity
	}
	a.render(w, r, status, "admin_event_form", data)
}

func (a *App) eventOr404(w http.ResponseWriter, r *http.Request) (event, bool) {
	id, ok := a.pathID(w, r)
	if !ok {
		return event{}, false
	}
	e, found, err := a.findEvent(r.Context(), id)
	if err != nil {
		a.serverError(w, r, err)
		return event{}, false
	}
	if !found {
		a.notFound(w, r)
		return event{}, false
	}
	return e, true
}

// The first POST shows a confirmation page; its button posts again with confirm=yes.
func (a *App) adminEventDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	e, ok := a.eventOr404(w, r)
	if !ok {
		return
	}
	if r.PostFormValue("confirm") != "yes" {
		a.render(w, r, http.StatusOK, "admin_event_delete", map[string]any{"Title": "Delete event", "Event": a.view(e)})
		return
	}
	// The event goes first, so its page is gone at once; then its sign-ups (design/data.md).
	if err := a.st.Delete(ctx, pkSite, eventSK(e.item.ID)); err != nil {
		a.serverError(w, r, err)
		return
	}
	signups, err := a.eventSignups(ctx, e.item.ID)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	for _, s := range signups {
		if err := a.st.Delete(ctx, pkSite, s.SK); err != nil {
			a.serverError(w, r, err)
			return
		}
	}
	a.record(ctx, "Event deleted", e.item.ID, e.item.Title)
	redirect(w, r, "/admin")
}

func (a *App) adminSignupsCSV(w http.ResponseWriter, r *http.Request) {
	e, ok := a.eventOr404(w, r)
	if !ok {
		return
	}
	signups, err := a.eventSignups(r.Context(), e.item.ID)
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="signups-%d.csv"`, e.item.ID))
	w.Header().Set("Cache-Control", "no-store")
	out := csv.NewWriter(w)
	_ = out.Write([]string{"Name", "Email", "People", "Signed up"})
	for _, s := range signups {
		_ = out.Write([]string{csvSafe(s.Name), csvSafe(s.Email), strconv.Itoa(s.People), strings.Replace(s.CreatedAt, "T", " ", 1)})
	}
	out.Flush()
}

// csvSafe stops a spreadsheet from running a name like "=HYPERLINK(...)" as a formula.
func csvSafe(v string) string {
	if v != "" && strings.ContainsRune("=+-@\t\r", rune(v[0])) {
		return "'" + v
	}
	return v
}

func (a *App) adminSignupDelete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	id, ok := a.pathID(w, r)
	if !ok {
		return
	}
	var removed signupItem
	err := retry(func() error {
		removed = signupItem{}
		var all []signupItem
		if err := a.st.Query(ctx, pkSite, "SIGNUP#", &all); err != nil {
			return err
		}
		for _, s := range all {
			if s.ID == id {
				removed = s
			}
		}
		if removed.ID == 0 {
			return nil
		}
		ops := []store.Op{store.DeleteOp(pkSite, removed.SK, store.Always)}
		if e, found, err := a.findEvent(ctx, removed.EventID); err != nil {
			return err
		} else if found {
			ev := e.item
			ev.People = max(0, ev.People-removed.People)
			ev.Version++
			ops = append(ops, store.PutOp(ev, store.IfVersion(e.item.Version)))
		}
		return a.st.Transact(ctx, ops...)
	})
	if err != nil {
		a.serverError(w, r, err)
		return
	}
	if removed.ID == 0 {
		a.notFound(w, r)
		return
	}
	title := ""
	if e, found, _ := a.findEvent(ctx, removed.EventID); found {
		title = e.item.Title
	}
	a.record(ctx, "Sign-up removed", removed.EventID, title)
	redirect(w, r, fmt.Sprintf("/admin/events/%d", removed.EventID))
}
