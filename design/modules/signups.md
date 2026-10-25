# Module: signups

Let visitors sign up for an event, and tell them it worked.

## The form (on `/events/{id}`)
- **Fields:**
  - **Name** (required, up to 100 characters).
  - **Email** (required, must look like an address).
  - **How many people, including you** (1–6, default 1).
- A hidden **trap field** catches bots (D-011).

## `POST /events/{id}/signup`
- **Checks**, in order:
  1. The event exists, is published, and hasn't started.
  2. The fields are valid.
  3. The trap field is empty, and the form wasn't submitted too fast (D-011).
  4. The head count fits in the places left (D-010).
- **On success:**
  - The sign-up is saved and the visitor is redirected to the event page, which says *You're signed up. We've sent a confirmation to {email}.*
  - A **confirmation email** is sent with the event's title, date, time and location. It says to use the contact form to cancel or change the sign-up.
  - If the email can't be sent, the sign-up still stands, and the page says *We couldn't send a confirmation email, but you're signed up.*
- **Invalid fields:** the form is shown again, keeping what was typed, with a message next to each field that needs fixing.
- **Not enough places:** *Sorry, only 2 places are left.* when there are some left, or *Sorry, this event is now full.*
- **A trap hit** gets the normal success page, but nothing is saved or sent. Bots can't tell they were caught.
- **Signing up twice** with the same email for the same event replaces the earlier head count instead of adding a second sign-up.

## Acceptance tests
1. A valid sign-up returns a redirect to `/events/{id}`. The event's places left drop by the head count, and one confirmation email is sent.
2. A sign-up for a head count larger than the places left is refused, and nothing is saved.
3. Two sign-ups with the same email for the same event leave one sign-up, with the later head count.
4. A sign-up with the trap field filled in looks successful but saves and sends nothing.
5. A sign-up to a draft, a past event or an unknown id is refused (404 for draft and unknown).
6. A sign-up without the form's CSRF token is refused with 403 (D-019).
