# Demo prep: to do

What's left before the demo in [presentation.md](presentation.md) can run. As of 2026-10-25, the design is written and the `lamp` site's code, tests and workflow are written but have never been run.

## Design first

- [x] Write `design/decisions.md`: numbered, platform-neutral decisions, each with a date, status and reason.
- [x] Write one spec per module in `design/modules/`: events listing, event sign-up, organizers' area.
- [x] Write the HTTP-level contract tests in `tests/contract/` that both branches must pass.
- [x] Record a decision on CSRF protection and the referrer policy: D-019 sets `Referrer-Policy: same-origin` and checks a token plus `Origin`.
- [x] Forms accept only same-origin requests, enforced however each platform does it best (D-025). The `lamp` branch checks a token, `Sec-Fetch-Site` and `Origin`; contract tests site 6 and signups 6 check the outcome.

## Before: `lamp`

- [x] Build the PHP 8 + MySQL site from the specs, in `public/`, `src/`, `templates/` and `migrations/`. Setup and choices are in `platform/README.md`.
- [ ] Push and get CI green: PHP syntax check, contract tests, then the Playwright + axe gate. None of it has run yet, because nothing needed is installed locally.
- [ ] Follow the one-time setup in `platform/README.md`, steps A–E: the host, the database, the host's key, the three confined deploy keys, and the `lamp` GitHub environment's variables and secrets.
- [x] Write the GitHub Actions workflow: tests → Playwright + axe gate → rsync deploy → smoke test → release tag.
- [ ] Deploy it, with a dry run first (step F), and confirm `/health` shows the release.

## The move

- [ ] Put the PHP code on the `lamp` branch for archival, and keep its environment and deploys working. On `lamp`, change the workflow to run only by hand (`workflow_dispatch`), and add `lamp` to the `lamp` environment's deployment branches, which currently allow only `main`.
- [ ] Create the `aws` branch and make it the repo's default branch.
- [ ] Have Claude regenerate each module in Go on `aws`, in `cmd/` and `internal/`, until the contract tests pass. Save the session so it can be shown in the demo.

## After: `aws`

- [ ] Write the CDK stack in Go in `infra/`: CloudFront flat-rate Free plan → Lambda (arm64, `provided.al2023`) → DynamoDB, with secrets in SSM Parameter Store.
- [ ] Enforce same-origin forms (D-025) the Go way, for example with Go 1.25's `http.CrossOriginProtection`, and record the choice in `platform/`. Make CloudFront forward the `Origin` and `Sec-Fetch-Site` headers to Lambda. Without them, the Lambda can't tell where a request came from. Its `Host` is also the Lambda's own address, not the site's, so an `Origin` comparison would refuse every form. Contract tests site 6 and signups 6 must pass.
- [ ] Add the cost tests that fail CI on idle-cost resources (NAT gateway, load balancer, public IPv4, Secrets Manager).
- [ ] Set up the GitHub OIDC provider and a deploy role scoped to this repo, the `aws` branch and this stack. This needs the AWS Paid plan.
- [ ] Create the `aws` GitHub environment holding only the role ARN, with no secrets.
- [ ] Write the workflow: tests → Playwright + axe gate → cost gate → OIDC → `cdk deploy` → smoke test → release tag.
- [ ] Set up AWS Budgets with zero-spend alerts and Cost Anomaly Detection.
- [ ] Deploy, and confirm both sites are live side by side with the same sign-up working on each.
- [ ] Check that the Billing console still shows $0.14 total and $0.00 for hosting.

## Notes and docs

- [ ] Check the three "Bugs from the original port" fixes in presentation.md against what the private app actually does. They were written from standard practice, not from its code. The WebKit `<select>` one is the least certain.
- [ ] Fill in the README's *(coming)* sections: the repo layout as built, and how to run it.
- [ ] If the build changes anything the talk claims, update presentation.md to match.

## Rehearsal

- [ ] Open the eight demo tabs listed in presentation.md, and keep a screenshot of each as a fallback.
- [ ] Run through the demo against a timer, aiming for 5 minutes.
