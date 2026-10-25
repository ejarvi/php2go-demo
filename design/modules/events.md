# Module: events (public)

The public face of the site: what's on, and the details of each event.

## `/`: upcoming events
- Lists **published** events whose start is today or later, soonest first.
- **Each event shows:** title, day and date (*Sat, Nov 8*), start time (and end time, if set), location, and **Full** when it has reached capacity.
- Each title links to `/events/{id}`.
- When there are no upcoming events: *No events are planned right now. Check back soon, or [get in touch](/contact).*

## `/events/{id}`: one event
- **Shows:** title, date and times, location, and the description. The description is plain text, with line breaks kept and links made clickable.
- **Spaces left:** when the event has a capacity, *12 places left*, or **Full**.
- **The sign-up form**, from [signups](signups.md). It isn't shown when the event is full or has already started; the page explains why instead.
- **Drafts and unknown ids** give 404 to visitors. Organizers see drafts, marked **Draft: not visible to the public**.
- **Past events** stay viewable at their address, marked *This event has ended*.

## Acceptance tests
1. `GET /` with two published future events and one draft lists exactly the two published ones, soonest first.
2. `GET /` doesn't list a published event that started yesterday.
3. `GET /events/{id}` for a draft returns 404 without a session, and 200 with an organizer session.
4. `GET /events/{id}` for an unknown id returns 404.
5. `GET /events/{id}` for a full event shows **Full** and no sign-up form.
6. A description containing `<script>` is shown as text, not run.
