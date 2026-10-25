# PHP2Go

**A typical PHP/MySQL shared-hosting site, regenerated in Go on AWS from the same specs with Claude Code.** Both versions are deployed by GitHub Actions. The move cuts hosting cost to a $0 target and replaces SSH deploy keys with short-lived OIDC credentials.

> **Status: work in progress.** This README describes what the repo is for and how it's laid out. The branches are being built, and sections marked *(coming)* will be filled in as they land.

## Why this exists

In October 2026, [PHP2Shell](https://en.wikipedia.org/wiki/PHP2Shell) (CVE-2026-109618 and CVE-2026-109619) exposed a critical remote code execution bug in PHP 5.4 through 8.5. Amazon's honeypots logged an exploit attempt four days before public disclosure, the patch embargo leaked, and the fixes depended on a small, donation-funded upstream team.

For a small site on shared hosting, that left the site's safety in other people's hands. PHP, Apache and Linux get patched on the hosting provider's schedule, and WordPress sites also depend on their plugins' PHP version constraints.

I had already been through this on a private app. I built it with Claude Code on PHP shared hosting, then regenerated it in Go on AWS in about a day and a half, because every requirement was already written down as a platform-neutral decision. This repo repeats that move in public, on a small site anyone can clone, so you can see the steps and reuse them.

## The idea: maintain the specs, regenerate the code

The code isn't the long-lived asset here. The design is:

- **A decision log** (`design/decisions.md`): numbered decisions, each with a date, a status and a reason. None of them mention PHP or Go.
- **One spec per module** (`design/modules/`): what each feature does, written for whoever builds it, whether that's a person or an agent.
- **Contract tests at the HTTP level:** a module is done when it passes them, whatever language it's written in. Both branches pass the same tests.

When the platform underneath becomes a liability, you don't patch and hope. You regenerate the modules on a new foundation and let the contract tests decide when you're done.

## The site

A small community-organization website, the kind found on thousands of shared hosts:
- a public events listing;
- a sign-up form for an event;
- a password-protected organizers' area to manage events and see sign-ups.

It deliberately sends no email, which keeps the demo free of mail setup.

## Two branches

| | `lamp` | `aws` |
|---|---|---|
| Language | PHP 8.x | Go (standard library first) |
| Data | MySQL | DynamoDB (one table, inside the always-free allowance) |
| Hosting | Apache on shared hosting | CloudFront → Lambda (arm64) |
| Infrastructure | The host's control panel | AWS CDK in Go, in the repo |
| Deploy | GitHub Actions → rsync over SSH, with a confined deploy key | GitHub Actions → OIDC (no stored keys) → `cdk deploy` |
| Secrets | GitHub secrets and files outside the web root | SSM Parameter Store |
| What you depend on for patches | PHP, Apache, Linux and the host's IT team | One Go binary with a short, pinned `go.mod`, on Lambda's OS-only runtime |
| Monthly cost | The hosting plan | $0 target, enforced in CI |

Both branches use the same pipeline in the same order: **test → gate → deploy → smoke test → tag a release.**

## Three things to look at

**1. Cost.** "Free tier" is treated as a design constraint, so it has a test. The `aws` branch's infrastructure tests fail the deploy if the planned resources include anything that costs money while idle:

| Resource | Typical cost |
|---|---|
| NAT gateway | ~$32/month |
| Load balancer | $15–30/month |
| Public IPv4 address | $3.60/month |
| Secrets Manager secret | $0.40/month each |

Zero-spend budget alerts and AWS Cost Anomaly Detection are the backstop.

**2. Security.** On `lamp`, a private SSH key lives in GitHub secrets, confined on the server with `rrsync`. On `aws`, there are no stored deploy keys at all. GitHub Actions gets a short-lived role through OIDC, scoped to one repo, one branch and one stack.

**3. The same pipeline.** Moving from SSH to cloud deploys doesn't mean leaving GitHub Actions. The quality gates carry over unchanged: tests, then accessibility checks with Playwright + axe-core, then deploy.

## Lessons from the original port

These came up on the private app this repo is modeled on.

- **On AWS's new Free plan, cost and security pull against each other.** That plan can never charge your card, and it blocked creating the GitHub OIDC provider. Keyless deploys required the Paid plan, which can't be undone, so the cost guards moved into CI and budget alerts.
- **`Referrer-Policy: no-referrer` makes browsers send `Origin: null`.** The CSRF check then refused every sign-in.
- **Warm Lambdas keep secrets they read at cold start.** Changing a secret has no effect until the next deploy.
- **WebKit reports a native `<select>` as a CSP violation** under a strict Content Security Policy.
- **CloudFront's flat-rate Free plan changes distribution settings when you subscribe.** Your infrastructure code has to match, or the next deploy fails.

## Repo layout *(coming)*

```
design/
  decisions.md      numbered, platform-neutral decisions
  modules/          one spec per feature
tests/contract/     HTTP-level tests both branches must pass
.github/workflows/  the deploy pipeline (differs per branch)
```

On `lamp`: `public/` (PHP pages), `migrations/` (SQL).
On `aws`: `cmd/`, `internal/` (Go modules), `infra/` (CDK and its cost tests).

## Running it *(coming)*

## Built with

- **Claude Code** (Claude Opus): writes the specs, the decisions, both implementations, the infrastructure, the workflows and the tests, working design-first.
- **GitHub Actions** with **GitHub OIDC** for CI/CD.
- **PHP 8, MySQL and Apache** on `lamp`.
- **Go, AWS Lambda, CloudFront, DynamoDB, SSM Parameter Store and AWS CDK** on `aws`.
- **Playwright + axe-core** for accessibility checks.

## Talk

This repo goes with a 5-minute demo for AI Tinkerers, "PHP2Go". The notes and sources are in [presentation.md](presentation.md).

## License

[MIT](LICENSE)
