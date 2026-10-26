# Module: signups

Let visitors sign up for an event, and show them it worked.

## The form (on `/events/{id}`)
- **Fields:**
  - **Name** (required, up to 100 characters).
  - **Email** (required, must look like an address). The form says it's only for the organizers to reach you if the event changes.
  - **How many people, including you** (1–6, default 1).
- A hidden **trap field** catches bots (D-011).

## `POST /events/{id}/signup`
- **Checks**, in order:
  1. The event exists, is published, and hasn't started.
  2. The fields are valid.
  3. The trap field is empty, and the form wasn't submitted too fast (D-011).
  4. The head count fits in the places left (D-010).
- **On success:** the sign-up is saved and the visitor is redirected to the event page. It says *You're signed up for {title}, {count} people.*, with the event's date, time and location, and suggests saving or printing the page. No email is sent (D-022).
- **Invalid fields:** the form is shown again, keeping what was typed, with a message next to each field that needs fixing.
- **Not enough places:** *Sorry, only 2 places are left.* when there are some left, or *Sorry, this event is now full.*
- **A trap hit** gets the normal success page, but nothing is saved. Bots can't tell they were caught.
- **Signing up twice** with the same email for the same event replaces the earlier head count instead of adding a second sign-up.
- **To cancel or change** a sign-up, people tell an organizer, who removes it in [admin](admin.md).

## Acceptance tests
1. A valid sign-up returns a redirect to `/events/{id}`, which shows the confirmation. The event's places left drop by the head count.
2. A sign-up for a head count larger than the places left is refused, and nothing is saved.
3. Two sign-ups with the same email for the same event leave one sign-up, with the later head count.
4. A sign-up with the trap field filled in looks successful but saves nothing.
5. A sign-up to a draft, a past event or an unknown id is refused (404 for draft and unknown).
6. A sign-up sent from another site is refused with 403, and nothing is saved (D-025).
