# Vision

## Problem
A small community club runs a handful of events a month: a park clean-up, a potluck, a workshop. Today the events are announced by word of mouth and sign-ups are kept on paper, so organizers don't know how many people to expect, and newcomers have no way to find out what's on or to get in touch.

## The site
A small public website for the club, of the kind found on thousands of shared hosts:
- **Upcoming events:** what's on, when and where.
- **Sign up:** a short form on each event, with a confirmation email.
- **Contact:** a form that emails the organizers.
- **Organizers' area:** sign in, add and edit events, and see who has signed up.

## Users
- **Visitors:** anyone. They don't have accounts.
- **Organizers:** a few club volunteers who manage events. All organizers have the same permissions.

## Goals
- **Simple to use:** on a phone first, but it also works well on a laptop.
- **Accessible:** meets WCAG 2.2 AA (D-013).
- **Keeps little personal data, briefly:** only what an event needs, deleted after it's over (D-011).
- **The repository is the single source of truth:** every change goes through it and is deployed automatically (D-004).
- **No manual server work after one-time setup** (D-005).
- **Cheap to run:** the lowest cost the hosting allows (D-015).
- **Can move to a new platform:** the design outlives any one implementation (D-002, D-003).

## Non-goals
- Payments or tickets.
- Visitor accounts, profiles or passwords.
- A waiting list, or visitors cancelling their own sign-up (they use the contact form instead).
- A mailing list or newsletter.
- More than one language.
