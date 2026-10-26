package site

import (
	"fmt"
	"html/template"
	"regexp"
	"strings"
	"time"
)

func (a *App) funcs() template.FuncMap {
	return template.FuncMap{"shortTime": shortTime, "peopleLabel": peopleLabel}
}

// eventDay is "Sat, Nov 8", with the year added when it isn't this year.
func (a *App) eventDay(e event) string {
	if e.Start.Year() == a.now().Year() {
		return e.Start.Format("Mon, Jan 2")
	}
	return e.Start.Format("Mon, Jan 2, 2006")
}

// eventTimes is "9:00 AM" or "9:00 AM – 12:00 PM".
func eventTimes(e event) string {
	times := e.Start.Format("3:04 PM")
	if !e.End.IsZero() {
		times += " – " + e.End.Format("3:04 PM")
	}
	return times
}

func shortTime(local string) string {
	t, err := time.Parse(timeLayout, local)
	if err != nil {
		return local
	}
	return t.Format("Jan 2, 3:04 PM")
}

func peopleLabel(n int) string {
	if n == 1 {
		return "1 person"
	}
	return fmt.Sprintf("%d people", n)
}

func placesLabel(n int) string {
	if n == 1 {
		return "1 place left"
	}
	return fmt.Sprintf("%d places left", n)
}

var urlPattern = regexp.MustCompile(`https?://[^\s<]+`)

// descriptionHTML shows plain text with line breaks kept and web addresses made into
// links (design/modules/events.md). It escapes first, so a <script> stays text.
func descriptionHTML(text string) template.HTML {
	escaped := template.HTMLEscapeString(text)
	linked := urlPattern.ReplaceAllStringFunc(escaped, func(m string) string {
		u := strings.TrimRight(m, ".,;:!?)")
		return `<a href="` + u + `" rel="nofollow noopener">` + u + `</a>` + m[len(u):]
	})
	return template.HTML(strings.ReplaceAll(linked, "\n", "<br>\n")) //nolint:gosec // built from escaped text
}

// eventView is an event as the templates show it.
type eventView struct {
	ID          int
	Title       string
	Location    string
	Description template.HTML
	HasDesc     bool
	Day, Times  string
	Published   bool
	Limited     bool // has a capacity
	Left        int
	LeftLabel   string
	Full        bool
	Started     bool
	Ended       bool
	People      int
	Capacity    int
}

func (a *App) view(e event) eventView {
	left, limited := e.placesLeft()
	now := a.now()
	return eventView{
		ID:          e.item.ID,
		Title:       e.item.Title,
		Location:    e.item.Location,
		Description: descriptionHTML(e.item.Description),
		HasDesc:     e.item.Description != "",
		Day:         a.eventDay(e),
		Times:       eventTimes(e),
		Published:   e.item.Published,
		Limited:     limited,
		Left:        left,
		LeftLabel:   placesLabel(left),
		Full:        e.full(),
		Started:     !e.Start.After(now),
		Ended:       e.lastMoment().Before(now),
		People:      e.item.People,
		Capacity:    e.item.Capacity,
	}
}
