package site

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/ejarvi/php2go-demo/internal/store"
)

// How the records in design/data.md are kept in one DynamoDB table. Events, sign-ups and
// the id counters share one small partition, so one query loads them.
//
//	PK         SK                        Item
//	SITE       EVENT#<id>                an event, with its sign-up total
//	SITE       SIGNUP#<event>#<id>       a sign-up; deleted by TTL 30 days after its event (D-004)
//	SITE       COUNTER#<kind>            the last id handed out
//	LOG        <time>#<random>           an activity entry; deleted by TTL after a year
//	SESSION#<hash of code>   -          a session; deleted by TTL when it expires
const (
	pkSite = "SITE"
	pkLog  = "LOG"
)

// Times are stored as local wall-clock times in the club's time zone (D-006).
const timeLayout = "2006-01-02T15:04"

type eventItem struct {
	PK          string `dynamodbav:"PK"`
	SK          string `dynamodbav:"SK"`
	ID          int    `dynamodbav:"id"`
	Title       string `dynamodbav:"title"`
	Description string `dynamodbav:"description"`
	StartsAt    string `dynamodbav:"starts_at"`
	EndsAt      string `dynamodbav:"ends_at,omitempty"`
	Location    string `dynamodbav:"location"`
	Capacity    int    `dynamodbav:"capacity"` // 0 means no limit
	Published   bool   `dynamodbav:"published"`
	People      int    `dynamodbav:"people"` // the sign-ups' total head count
	UpdatedAt   string `dynamodbav:"updated_at"`
	Version     int    `dynamodbav:"version"`
}

type signupItem struct {
	PK        string `dynamodbav:"PK"`
	SK        string `dynamodbav:"SK"`
	ID        int    `dynamodbav:"id"`
	EventID   int    `dynamodbav:"event_id"`
	Name      string `dynamodbav:"name"`
	Email     string `dynamodbav:"email"`
	People    int    `dynamodbav:"people"`
	CreatedAt string `dynamodbav:"created_at"`
	Expires   int64  `dynamodbav:"expires"`
}

type counterItem struct {
	PK      string `dynamodbav:"PK"`
	SK      string `dynamodbav:"SK"`
	Value   int    `dynamodbav:"value"`
	Version int    `dynamodbav:"version"`
}

type sessionItem struct {
	PK          string `dynamodbav:"PK"`
	SK          string `dynamodbav:"SK"`
	Fingerprint string `dynamodbav:"fingerprint"`
	Expires     int64  `dynamodbav:"expires"`
}

type activityItem struct {
	PK         string `dynamodbav:"PK"`
	SK         string `dynamodbav:"SK"`
	Action     string `dynamodbav:"action"`
	EventID    int    `dynamodbav:"event_id"`
	EventTitle string `dynamodbav:"event_title"`
	CreatedAt  string `dynamodbav:"created_at"`
	Expires    int64  `dynamodbav:"expires"`
}

func eventSK(id int) string            { return fmt.Sprintf("EVENT#%d", id) }
func signupPrefix(eventID int) string  { return fmt.Sprintf("SIGNUP#%d#", eventID) }
func signupSK(eventID, id int) string  { return fmt.Sprintf("SIGNUP#%d#%d", eventID, id) }
func counterSK(kind string) string     { return "COUNTER#" + kind }
func sessionPK(codeHash string) string { return "SESSION#" + codeHash }

// errRetries means an optimistic write kept conflicting with other writes.
var errRetries = errors.New("site: too many conflicting changes; try again")

// retry runs fn until it doesn't return store.ErrConflict, a few times at most.
func retry(fn func() error) error {
	for range 5 {
		if err := fn(); !errors.Is(err, store.ErrConflict) {
			return err
		}
	}
	return errRetries
}

// event is an event as the pages use it.
type event struct {
	item  eventItem
	Start time.Time
	End   time.Time // zero when there's no end time
}

func (a *App) toEvent(it eventItem) event {
	e := event{item: it}
	e.Start, _ = time.ParseInLocation(timeLayout, it.StartsAt, a.cfg.Location)
	if it.EndsAt != "" {
		e.End, _ = time.ParseInLocation(timeLayout, it.EndsAt, a.cfg.Location)
	}
	return e
}

// lastMoment is when the event ends, or starts if it has no end time.
func (e event) lastMoment() time.Time {
	if e.End.IsZero() {
		return e.Start
	}
	return e.End
}

// placesLeft reports the places left, and false when there's no capacity.
func (e event) placesLeft() (int, bool) {
	if e.item.Capacity == 0 {
		return 0, false
	}
	return max(0, e.item.Capacity-e.item.People), true
}

func (e event) full() bool {
	left, limited := e.placesLeft()
	return limited && left == 0
}

func (a *App) allEvents(ctx context.Context) ([]event, error) {
	var items []eventItem
	if err := a.st.Query(ctx, pkSite, "EVENT#", &items); err != nil {
		return nil, err
	}
	events := make([]event, 0, len(items))
	for _, it := range items {
		events = append(events, a.toEvent(it))
	}
	sort.Slice(events, func(i, j int) bool {
		if !events[i].Start.Equal(events[j].Start) {
			return events[i].Start.Before(events[j].Start)
		}
		return events[i].item.ID < events[j].item.ID
	})
	return events, nil
}

func (a *App) findEvent(ctx context.Context, id int) (event, bool, error) {
	var it eventItem
	ok, err := a.st.Get(ctx, pkSite, eventSK(id), &it)
	if err != nil || !ok {
		return event{}, false, err
	}
	return a.toEvent(it), true, nil
}

func (a *App) eventSignups(ctx context.Context, eventID int) ([]signupItem, error) {
	var items []signupItem
	if err := a.st.Query(ctx, pkSite, signupPrefix(eventID), &items); err != nil {
		return nil, err
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].CreatedAt != items[j].CreatedAt {
			return items[i].CreatedAt < items[j].CreatedAt
		}
		return items[i].ID < items[j].ID
	})
	return items, nil
}

// nextID returns an op that claims the next id of a kind, and the id. The op belongs in
// the same transaction as the item that uses the id, so a failed write wastes nothing.
func (a *App) nextID(ctx context.Context, kind string) (store.Op, int, error) {
	var c counterItem
	if _, err := a.st.Get(ctx, pkSite, counterSK(kind), &c); err != nil {
		return store.Op{}, 0, err
	}
	next := counterItem{PK: pkSite, SK: counterSK(kind), Value: c.Value + 1, Version: c.Version + 1}
	return store.PutOp(next, store.IfVersion(c.Version)), next.Value, nil
}

// signupExpiry is when a sign-up is deleted: 30 days after its event ends (D-004).
func signupExpiry(e event) int64 {
	return e.lastMoment().AddDate(0, 0, 30).Unix()
}

// record adds an activity entry (design/data.md). It never holds visitor names or emails.
func (a *App) record(ctx context.Context, action string, eventID int, eventTitle string) {
	now := a.now()
	entry := activityItem{
		PK:         pkLog,
		SK:         now.UTC().Format(time.RFC3339Nano) + "#" + randomHex(4),
		Action:     action,
		EventID:    eventID,
		EventTitle: eventTitle,
		CreatedAt:  now.Format(timeLayout),
		Expires:    now.AddDate(1, 0, 0).Unix(),
	}
	if err := a.st.Put(ctx, entry, store.Always); err != nil {
		a.log.Error("recording activity", "action", action, "err", err)
	}
}

func (a *App) recentActivity(ctx context.Context, limit int) ([]activityItem, error) {
	var items []activityItem
	if err := a.st.Query(ctx, pkLog, "", &items); err != nil {
		return nil, err
	}
	now := a.now().Unix()
	recent := make([]activityItem, 0, limit)
	for i := len(items) - 1; i >= 0 && len(recent) < limit; i-- {
		if items[i].Expires > now { // TTL deletion can lag, so hide what's past its year
			recent = append(recent, items[i])
		}
	}
	return recent, nil
}

// emailKey is how emails are compared: two sign-ups with the same email, whatever the
// capitalization, are one sign-up (design/modules/signups.md).
func emailKey(email string) string { return strings.ToLower(strings.TrimSpace(email)) }
