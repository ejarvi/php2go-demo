# Module: signin

How organizers sign in. There are no passwords: an organizer asks for a link, and the link arrives by email (D-007).

## `/signin`
- One field, **Email**, and a **Send me a link** button.
- **After any submit,** the page says *If that address belongs to an organizer, a sign-in link is on its way.* It says the same whether or not the address is known, so the form can't be used to find out who the organizers are.
- **For an active organizer**, a link is created and emailed:
  - It holds a random one-time code; only a hash of the code is stored.
  - It's valid for **15 minutes**.
  - Several links can be active at once.

## `/signin/confirm?code=…`
- Opening the link **doesn't sign anyone in.** It shows a page with a **Sign in** button. Email security scanners that open links therefore can't use up the link.
- **Pressing Sign in** (`POST /signin/confirm`) checks the code. It must be unused, unexpired, and for an organizer who is still active. Then it:
  - marks the code used;
  - starts a session lasting **8 hours**;
  - redirects to `/admin`.
- **An expired, used or unknown code** shows *This link has expired or was already used* and a link to `/signin`.

## Sessions
- **The cookie:** `Secure`, `HttpOnly` and `SameSite=Lax`. It holds a random code, and only its hash is stored.
- **`POST /signout`** ends the session and redirects to `/`.
- **Removing an organizer** ends their sessions at once.

## Acceptance tests
1. `POST /signin` with an unknown address and with an organizer's address return the same page. Only the organizer gets an email.
2. `GET /signin/confirm?code=…` doesn't set a session cookie.
3. `POST /signin/confirm` with a valid code sets the session cookie and redirects to `/admin`. Posting the same code again is refused.
4. A code older than 15 minutes is refused.
5. With a session, `GET /admin` returns 200. After `POST /signout`, it redirects to `/signin`.
6. A session for an organizer who has since been removed is refused.
7. Sign-in works when the browser sends a same-origin `Referer` and `Origin` (D-019).
