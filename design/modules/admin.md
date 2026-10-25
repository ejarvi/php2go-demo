# Module: admin

The organizers' area: manage events and see sign-ups. Every `/admin` path requires a session.

## `/admin`: events
- **Lists all events**, including drafts: upcoming ones first (soonest first), then past ones (most recent first).
- **Each row shows:** title, date, **Draft** or **Published**, and the sign-up count as *people / capacity*, or *people* when there's no capacity.
- **New event** opens the event form.

## The event form (`/admin/events/new`, `/admin/events/{id}`)
- **Fields:**
  - **Title** (required, up to 120 characters).
  - **Date** (required).
  - **Start time** (required) and **end time** (optional, must be after the start).
  - **Location** (required).
  - **Description** (up to 5,000 characters).
  - **Capacity** (optional, 1–500).
  - **Published** (a checkbox).
- **Saving** records when the event was changed, and adds an activity entry.
- **Lowering the capacity** below the people already signed up is allowed, but warns: *23 people are already signed up.*
- **Delete** asks for confirmation on a separate page, then removes the event and its sign-ups.

## Sign-ups (on the event's admin page)
- **Shown as a table:** name, email, head count and when they signed up, earliest first, with the total underneath.
- **Remove** deletes one sign-up, for example when someone tells an organizer they can't come.
- **Download CSV** gives the same columns, for a printed check-in list.

## Acceptance tests
1. Every `/admin` path without a session redirects to `/signin`, and every `/admin` POST without a session is refused.
2. Creating a published event makes it appear on `/`. Unticking **Published** removes it from `/`.
3. An end time before the start time is refused, and the form keeps what was typed.
4. Deleting an event removes its sign-ups. Its public page then returns 404.
5. The CSV download has a header row and one row per sign-up, and is served as a file attachment.
6. On a fresh site with no data, an organizer can sign in with the configured password and create the first event.
7. Each change adds an activity entry with the action, and no visitor names or emails.
