# Design

This folder is the long-lived part of the project. The code in each branch is generated from it, and can be generated again on a different foundation.

## Rules for this folder

- **No platform names.** Nothing here names a programming language, framework, database, web server or hosting provider (D-002). Those choices belong to an implementation, and each branch records its own in `platform/`, outside the design.
- **Data in general terms.** [data.md](data.md) says *what* is kept, for how long and who can see it. It never says how it's laid out in storage.
- **Every choice is a numbered decision** in [decisions.md](decisions.md), with its date, status and reason. A later decision can supersede an earlier one, but nothing is deleted.
- **Each feature is a module** in [modules/](modules/), with its pages, its rules and its acceptance tests. The acceptance tests are written against HTTP requests and responses, so any implementation can be checked against them (D-003).

## Files

| File | What it holds |
|------|---------------|
| [00-vision.md](00-vision.md) | The problem, who uses the site, goals and non-goals |
| [decisions.md](decisions.md) | The decision log |
