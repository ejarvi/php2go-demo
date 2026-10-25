# Module: signin

How organizers sign in. Organizers share one password, set as a deploy secret (D-023). There are no organizer accounts.

## `/signin`
- **One field,** **Password**, and a **Sign in** button.
- **The right password** starts a session lasting **8 hours** and redirects to `/admin`, or back to the `/admin` page the organizer was trying to open.
- **A wrong password** shows *That password isn't right.* and waits about a second before answering, to slow down guessing. The password is compared in a way that takes the same time whatever is typed.
- **Already signed in:** `/signin` redirects to `/admin`.

## Sessions
- **The cookie:** `Secure`, `HttpOnly` and `SameSite=Lax`. It holds a random code, and only its hash is stored.
- **`POST /signout`** ends the session and redirects to `/`.
- **Changing the password** (a new secret, then a deploy) ends every existing session.

## Acceptance tests
1. `POST /signin` with the right password sets the session cookie and redirects to `/admin`.
2. `POST /signin` with a wrong password sets no cookie and shows the error.
3. With a session, `GET /admin` returns 200. After `POST /signout`, it redirects to `/signin`.
4. A session started before the password changed is refused.
5. The session cookie is `Secure`, `HttpOnly` and `SameSite=Lax`.
6. Sign-in works when the browser sends a same-origin `Referer` and `Origin` (D-019).
