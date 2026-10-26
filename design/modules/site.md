# Module: site

What every page shares: layout, errors, headers, the health check and the release ID.

## Health check: `GET /health`
- **When the site can read its storage,** returns 200 with `{"status":"ok","release":"r12+abc1234"}`.
- **Otherwise,** returns 503 with `{"status":"error","release":…}`. No error details are given.
- The deploy's smoke test uses it (D-021).

## Release ID
- The ID has the form `r<N>+<short commit>`, where N is the deploy number.
- It appears in every page's footer, and in `/health` (D-016).

## Response headers on every page (D-019)
- `Content-Security-Policy`: scripts and styles only from the site itself, with no inline scripts and no third-party resources.
- `Referrer-Policy: same-origin`.
- `X-Content-Type-Options: nosniff`.
- Frames aren't allowed (`frame-ancestors 'none'`).
- `Strict-Transport-Security`. HTTPS only; HTTP redirects to HTTPS.

## Forms (D-025)
- **Only same-origin requests are accepted.** A POST that a browser marks as coming from another origin, including another subdomain, is refused. How that's checked is up to the implementation.
- **A refused POST** gets 403 and a page that offers to return to the form and try again. Nothing is changed.

## Errors
- **Unknown paths:** a 404 page in the site's layout.
- **Unexpected errors:** a 500 page saying *Something went wrong. Please try again.* The details are written to the application log, never to the page (and never personal data, see [data.md](../data.md)).

## Acceptance tests
1. `GET /health` returns 200 with `status` `ok` and the current release ID.
2. Every HTML response carries the headers listed above.
3. `GET /no-such-page` returns 404 in the site's layout.
4. The footer of `/` shows the same release ID as `/health`.
5. No page loads a script or stylesheet from another site.
6. A POST to `/signin` with the right password, sent with `Origin` and `Sec-Fetch-Site` showing another site, is refused with 403 and starts no session. The same POST sent the way a browser sends it from another subdomain is refused too (D-025).
