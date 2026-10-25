# Decision log

Format: **D-NNN — Title** · Date · Status (Proposed / Accepted / Superseded by D-NNN) · Decision · Why

## D-001 — Design before implementation
- Date: 2026-10-25 · Status: Accepted
- Decision: Nothing is built until the design in this folder is agreed. Changes in behavior start as a change here.
- Why: the design is what carries over when the implementation is regenerated.

## D-002 — The design is platform-neutral
- Date: 2026-10-25 · Status: Accepted
- Decision: Nothing in `design/` names a language, framework, database, web server or host. Each implementation records its platform choices in its own `platform/` folder.
- Why: when a platform becomes a liability (cost, security, support), the site can be rebuilt on another one without deciding anything again.

## D-003 — Modules with specs and acceptance tests
- Date: 2026-10-25 · Status: Accepted
- Decision: Each feature is a module with a spec in `design/modules/`. Each spec lists acceptance tests stated as HTTP requests and expected responses. An implementation of a module is done when it passes them. Every implementation is checked against the same tests.
- Why: tests at the HTTP level don't care what the code is written in, so they can judge a regenerated module.

## D-004 — Keep little personal data, and not for long
- Date: 2026-10-25 · Status: Accepted
- Decision: A sign-up keeps only a name, an email address and a head count. It's deleted automatically 30 days after its event ends. See [data.md](data.md).
- Why: the club needs a head count and a way to reach people about a change, nothing more. Data that isn't kept can't leak.

## D-005 — Contact messages are emailed, not stored
- Date: 2026-10-25 · Status: Accepted
- Decision: The contact form sends its message to the organizers' email address and keeps no copy.
- Why: the organizers already read and answer email. A stored inbox would be a second place to check and a second copy of personal data.

## D-006 — One time zone
- Date: 2026-10-25 · Status: Accepted
- Decision: All dates and times are in the club's time zone, set once in the site's configuration.
- Why: everyone who comes to an event is local. Converting time zones would only add room for mistakes.

## D-007 — Organizers sign in with an emailed one-time link
- Date: 2026-10-25 · Status: Accepted
- Decision: No passwords. An organizer asks for a link, and the link is valid for 15 minutes and works once. Opening it shows a **Sign in** button; pressing the button uses up the link. See [modules/signin.md](modules/signin.md).
- Why: there are no passwords to store, leak or reset. The button stops email link scanners from using up the link.

## D-008 — The first organizer comes from configuration
- Date: 2026-10-25 · Status: Accepted
- Decision: The first organizer's email address is a deploy setting, and that organizer is added on first start. After that, organizers add and remove each other on `/admin/organizers`.
- Why: a fresh site needs someone who can sign in, without any manual database work (D-014).

## D-009 — Pages work without JavaScript
- Date: 2026-10-25 · Status: Accepted
- Decision: Every page and form works with JavaScript turned off. Script, if any, only adds convenience.
- Why: the forms are simple, and pages rendered on the server are fast on any phone and easy to test at the HTTP level (D-003).

## D-010 — Capacity is a hard limit, with no waiting list
- Date: 2026-10-25 · Status: Accepted
- Decision: An event's capacity, if set, is never exceeded by a sign-up. A full event shows **Full** and no form.
- Why: a waiting list needs notifications and visitor self-service. That's out of scope (00-vision.md).

## D-011 — Bots are caught quietly
- Date: 2026-10-25 · Status: Accepted
- Decision: Public forms carry a hidden trap field and a minimum time to fill in. A caught submission gets the normal success page, but nothing is saved or sent. There's no CAPTCHA.
- Why: this stops most form spam with no third-party service and no hurdle for real visitors.

## D-012 — Email goes through a configurable mail relay
- Date: 2026-10-25 · Status: Accepted
- Decision: Sign-in links, sign-up confirmations and contact messages are sent through an SMTP relay. Its host, port, user and password are deploy secrets, and the sender address is a setting.
- Why: every host can reach a mail relay. Keeping mail separate from hosting lets either one change without the other.

## D-013 — The repository is the single source of truth, and deploys are automatic
- Date: 2026-10-25 · Status: Accepted
- Decision: The live site is always built from `main`. Every push to `main` deploys through GitHub Actions, one deploy at a time. See [delivery.md](delivery.md).
- Why: what's in the repository is what's running. There's nothing to remember to copy.

## D-014 — No manual server work after one-time setup
- Date: 2026-10-25 · Status: Accepted
- Decision: Hand-done steps are allowed only for one-time setup, and they're written down as a checklist in the implementation's `platform/` folder.
- Why: anything changed by hand is invisible to the next person, and it gets lost when the site moves.

## D-015 — No secrets in the repository; deploy credentials as narrow as possible
- Date: 2026-10-25 · Status: Accepted
- Decision: Secrets live in GitHub secrets or the platform's secret store. Deploys use short-lived credentials where the platform allows them. A stored deploy key, when unavoidable, can deploy this site and nothing else.
- Why: the repository is meant to be public. A leaked deploy credential should be useless, or at least contained.

## D-016 — Release IDs, and rolling back by reverting
- Date: 2026-10-25 · Status: Accepted
- Decision: Each deploy gets a release ID `r<N>+<short commit>`. It's shown in the footer and in `/health`, and the commit is tagged with it. To roll back, revert the commit.
- Why: anyone looking at a page can tell exactly which commit is running.

## D-017 — Pipeline order: test, gate, deploy, smoke test, tag
- Date: 2026-10-25 · Status: Accepted
- Decision: A failure at any step stops the rest, and the live site stays on its last good release.
- Why: the same order works on any platform, so the pipeline can stay the same when the site moves.

## D-018 — Accessible and responsive, checked on every deploy
- Date: 2026-10-25 · Status: Accepted
- Decision: The site meets WCAG 2.2 AA and works at phone, laptop and wide-screen sizes. Automated accessibility and layout checks in three browser engines must pass before any deploy.
- Why: checks that run on every deploy catch what a reviewer would miss on the fifth change.

## D-019 — Strict headers, and CSRF protection on every form
- Date: 2026-10-25 · Status: Accepted
- Decision: Every page sends the headers in [modules/site.md](modules/site.md): a strict Content Security Policy, `Referrer-Policy: same-origin`, and so on. Every POST checks a CSRF token and the `Origin` header.
- Why: these block whole classes of attack for free. The referrer policy is `same-origin`, not `no-referrer`, because with `no-referrer` some browsers send `Origin: null` and the origin check would refuse every legitimate form.

## D-020 — The lowest cost the hosting allows
- Date: 2026-10-25 · Status: Accepted
- Decision: Anything that costs money while the site sits idle needs its own decision. Where hosting is described as code, a test in the pipeline fails the deploy if a resource that costs money while idle appears without one.
- Why: a small club site gets little traffic. It should cost little or nothing, and a test catches an expensive mistake before the bill does.

## D-021 — A health check the smoke test can trust
- Date: 2026-10-25 · Status: Accepted
- Decision: `GET /health` reports whether the site can reach its storage, and which release is running. The deploy's smoke test requires 200 and the new release ID.
- Why: a deploy isn't done until the new release is answering and can reach its data.
