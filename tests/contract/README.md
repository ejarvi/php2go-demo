# Acceptance tests

The acceptance tests from each module spec in [design/modules/](../../design/modules/), written against HTTP requests and responses so that every implementation is checked against the same tests (D-003). Each test is numbered as in its spec.

They need only Node 22, and they test a site that is already running:

```sh
BASE_URL=http://127.0.0.1:8080 ORGANIZER_PASSWORD=… CLUB_TIMEZONE=… \
  node --test --test-concurrency=1 'tests/contract/*.test.mjs'
```

- **The site under test** needs `FORM_MIN_SECONDS=0`, because the tests fill in forms instantly.
- **The password-change test** needs a second copy of the site on the same storage, started with a different organizer password, at `ALT_BASE_URL`. Without it, that test is skipped.
- **Form field names** are part of the contract, like the paths in [pages.md](../../design/pages.md): the tests fill in forms by name. A new implementation keeps the same names.
- **Tests share one copy of the site,** so each uses its own event titles and email addresses, and none assumes the site is otherwise empty.
