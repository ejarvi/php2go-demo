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
