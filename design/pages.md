# Pages

Every page and form, who can reach it, and which module owns it. Paths are part of the contract (D-003): every implementation serves the same ones.

| Method | Path | What | Who | Module |
|--------|------|------|-----|--------|
| GET | `/` | Upcoming published events, soonest first | Everyone | [events](modules/events.md) |
| GET | `/events/{id}` | One event, with its sign-up form | Everyone (drafts: organizers only) | [events](modules/events.md) |
| POST | `/events/{id}/signup` | Sign up for an event | Everyone | [signups](modules/signups.md) |
| GET | `/contact` | Contact form | Everyone | [contact](modules/contact.md) |
| POST | `/contact` | Send a message to the organizers | Everyone | [contact](modules/contact.md) |
| GET | `/signin` | Ask for a sign-in link | Everyone | [signin](modules/signin.md) |
| POST | `/signin` | Email a sign-in link | Everyone | [signin](modules/signin.md) |
| GET | `/signin/confirm?code=…` | "Sign in" button for a link | Whoever has the link | [signin](modules/signin.md) |
| POST | `/signin/confirm` | Use the link and start a session | Whoever has the link | [signin](modules/signin.md) |
| POST | `/signout` | End the session | Organizers | [signin](modules/signin.md) |
| GET | `/admin` | All events, including drafts, with sign-up counts | Organizers | [admin](modules/admin.md) |
| GET | `/admin/events/new` | New event form | Organizers | [admin](modules/admin.md) |
| POST | `/admin/events` | Create an event | Organizers | [admin](modules/admin.md) |
| GET | `/admin/events/{id}` | Edit an event, and see its sign-ups | Organizers | [admin](modules/admin.md) |
| POST | `/admin/events/{id}` | Save changes to an event | Organizers | [admin](modules/admin.md) |
| POST | `/admin/events/{id}/delete` | Delete an event and its sign-ups | Organizers | [admin](modules/admin.md) |
| GET | `/admin/events/{id}/signups.csv` | Download the sign-ups | Organizers | [admin](modules/admin.md) |
| POST | `/admin/signups/{id}/delete` | Remove one sign-up | Organizers | [admin](modules/admin.md) |
| GET | `/admin/organizers` | List organizers | Organizers | [admin](modules/admin.md) |
| POST | `/admin/organizers` | Add an organizer | Organizers | [admin](modules/admin.md) |
| POST | `/admin/organizers/{id}/remove` | Remove an organizer | Organizers | [admin](modules/admin.md) |
| GET | `/health` | Is the site up, and can it reach its storage? | Everyone | [site](modules/site.md) |

## Every page
- **Layout:** a header with the club's name and links to Events and Contact; for organizers, also Admin and Sign out.
- **Footer:** the release ID (D-016).
- **Errors:** unknown paths get a friendly 404 page. Any `/admin` path without a session redirects to `/signin`.
- **No JavaScript needed:** every page and form works without it (D-009).
