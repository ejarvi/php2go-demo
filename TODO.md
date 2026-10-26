# Demo prep: to do

What's left before the demo in [presentation.md](presentation.md) can run. As of 2026-10-26, the `lamp` site is live, deployed by GitHub Actions, and its tests pass.

## Design first

- [x] Write `design/decisions.md`: numbered, platform-neutral decisions, each with a date, status and reason.
- [x] Write one spec per module in `design/modules/`: events listing, event sign-up, organizers' area.
- [x] Write the HTTP-level contract tests in `tests/contract/` that both branches must pass.
- [x] Record a decision on CSRF protection and the referrer policy: D-019 sets `Referrer-Policy: same-origin` and checks a token plus `Origin`.
- [x] Forms accept only same-origin requests, enforced however each platform does it best (D-025). The `lamp` branch checks a token, `Sec-Fetch-Site` and `Origin`; contract tests site 6 and signups 6 check the outcome.

## Before: `lamp`

- [x] Build the PHP 8 + MySQL site from the specs, in `public/`, `src/`, `templates/` and `migrations/`. Setup and choices are in `platform/README.md`.
- [x] Push and get CI green: PHP syntax check, contract tests, then the Playwright + axe gate. All passed on 2026-10-26. They're now off by default and switched on when wanted (D-026).
- [x] Follow the one-time setup in `platform/README.md`, steps A–E: the host, the database, the host's key, the three confined deploy keys, and the `lamp` GitHub environment's variables and secrets.
- [x] Write the GitHub Actions workflow: tests → Playwright + axe gate → rsync deploy → smoke test → release tag.
- [x] Deploy it, with a dry run first (steps 21–23), and confirm `/health` shows the release. First live release: r7, on 2026-10-26.
- [ ] Sign in on the live site and create the demo events (step 23).

## The move

- [x] Put the PHP code on the `lamp` branch for archival (tag `lamp-final`). Its workflow now runs only by hand.
- [ ] In the `lamp` environment, add `lamp` under **Deployment branches and tags**, so the archived version can still be deployed by hand.
- [x] `main` becomes the Go version, as in the other project (no separate `aws` branch, so the default branch stays `main`).
- [x] Claude regenerated each module in Go in `cmd/` and `internal/`. It compiled first time; the acceptance tests and gate run against it in CI.

## After: Go on AWS (`main`)

- [x] CDK stacks in Go in `infra/`: `Php2goGitHub` (OIDC provider and deploy role) and `Php2go` (DynamoDB, Lambda arm64, function URL, CloudFront with the secret origin header).
- [x] Same-origin forms (D-025) with Go's `http.CrossOriginProtection`; CloudFront forwards `Origin` and `Sec-Fetch-Site`.
- [x] Cost tests that fail CI on idle-cost resources, plus tests of the deploy role's limits and the Free plan's rules.
- [x] The workflow: tests → gate → OIDC → secrets to SSM → `cdk diff`/`deploy` → smoke test → tag. It skips the deploy until `AWS_ROLE_ARN` exists.
- [x] Followed `platform/README.md` from Part B: a member account, `aws login` with a role-switching profile, `cdk bootstrap`, `cdk deploy Php2goGitHub`, and the `aws` GitHub environment.
- [x] First deploy to AWS on 2026-10-26 (r26), green from tests to tag. Deploy logs now mask the site's and the function's addresses.
- [x] CloudFront on the flat-rate Free plan ($0/month), with its web ACL kept by `FREE_PLAN_WEB_ACL_ARN` (r36).
- [ ] Re-enter the `SITE_URL` secret as exactly the CloudFront address and redeploy: a form post from an older browser (no `Sec-Fetch-Site`) is refused, so the trusted origin doesn't match yet.
- [x] The gate runs in Microsoft's Playwright image, after GitHub's Ubuntu mirror stalled the browser install four times.
- [ ] Confirm both sites are live side by side with the same sign-up working on each.
- [ ] Check that the Billing console still shows $0.14 total and $0.00 for hosting.
- [ ] Save or record the Claude session that regenerated the site, for the demo.

## Notes and docs

- [ ] Check the three "Bugs from the original port" fixes in presentation.md against what the private app actually does. They were written from standard practice, not from its code. The WebKit `<select>` one is the least certain.
- [x] Fill in the README's *(coming)* sections: the repo layout as built, and how to run it.
- [ ] If the build changes anything the talk claims, update presentation.md to match.

## Rehearsal

- [ ] Open the eight demo tabs listed in presentation.md, and keep a screenshot of each as a fallback.
- [ ] Run through the demo against a timer, aiming for 5 minutes.
