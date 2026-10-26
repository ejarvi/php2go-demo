# PHP2Go: talk notes

**PHP2Go: Regeneration as Remediation for PHP2Shell**
*When there's no patch yet, when patching isn't an option, or when it costs more than rebuilding*

A 5-minute live demo for AI Tinkerers. There are no slides: everything is shown in the browser, the editor, the terminal, GitHub and the AWS console. The demo repo is this one; see the [README](README.md).

I also used Claude to help develop this presentation, including these notes.

## In one paragraph

The usual answer to a vulnerability is find, wait, patch. For a small site on shared PHP hosting, that means following patch guidance and waiting: on the PHP project, on Apache and Linux, and on how quickly your host's IT security team rolls out the fix. You're trusting the same dependencies that exposed you to get you out. This talk challenges the assumption that this is the only option. With coding agents, regenerating the code is cheap. If your requirements live in platform-neutral decisions and tests, you can **regenerate** the vulnerable component in a different language, on a different platform, and let the same tests prove it still meets every requirement. When PHP2Shell landed, I did this with a private app: Claude regenerated it in Go on AWS from its specs and decision log, in about a day and a half. This repo repeats that move on a mock club-events site, with the PHP version archived on a `lamp` branch and the Go version on `main`.

## When regeneration beats patching

Regeneration is worth considering whenever waiting for a patch is the riskier or costlier choice:
- **A vulnerability is being exploited, and there's no patch yet.** All you know is that your platform has a hole and someone is using it. With PHP2Shell, Amazon's honeypots logged exploitation four days before disclosure. A site whose requirements live in specs and tests can leave the vulnerable platform without waiting for its fix.
- **Patching isn't an option.** The platform is end-of-life or abandoned, your host won't upgrade, or the fix breaks plugins and code you depend on.
- **Patching costs more than rebuilding.** Upgrading an old stack, retesting it and re-hosting it can take longer than regenerating the whole solution on a platform you choose, once coding is cheap and the tests define "done".

Regeneration doesn't replace patching. It adds a second option, and it only works if your decisions, constraints and tests are good enough to judge the regenerated code.

The talk rests on three points:
1. **Regeneration is a remediation option:** when the platform is the problem, rebuild on another one instead of waiting for its fix. Put your effort into decisions and tests, and the code becomes replaceable.
2. **Better security:** a short, visible dependency list you patch on your own schedule, and deploys without stored keys (SSH → OIDC).
3. **Lower cost, same pipeline:** shared hosting → AWS at $0, enforced in CI, deployed by the same GitHub Actions gates in the same order.

The honest caveat: the new stack has a supply chain too (Go modules, AWS). The claim isn't "no dependencies". It's a shorter list you can see and control, plus the option to move again.

The same approach applies beyond small sites. With more investment in design, constraints, requirements and tests, an enterprise team can prepare its components to be regenerated and redeployed on other dependencies, which makes its software supply chain more secure and resilient (takeaway 8).

## Background: PHP2Shell

- **What it is:** CVE-2026-109618 and CVE-2026-109619, a critical remote code execution bug in PHP 5.4 through 8.5, patched on 2026-10-21.
- **Found with AI help:** Calif found it with an expert steering LLM agents through PHP's memory allocator. Wiz found bugs independently, also using frontier models.
- **The exploit works by shaping PHP's request heap.** Wiz says frontier models helped "develop and adapt working exploits across multiple applications in hours," adapting them to each target's memory layout.
- **The rollout was messy:** the advisories went out early, and Chainguard shipped patched packages before the embargo ended.
- **Exploited before disclosure:** Amazon's MadPot honeypots logged an exploitation attempt on 2026-10-17, four days early.
- **A small upstream:** the PHP Foundation behind the fix has about 13 staff and runs on donations.
- **The aftermath:** GitHub filled up with non-working and fake exploits, and attack traffic has been heavy but mostly unsuccessful.

AI is changing the economics of security on both sides. On the attack side, finding and weaponizing bugs got cheaper. That's the backdrop. **This talk is about the defense side.** There, AI's value often isn't counted as security. It isn't only finding bugs faster; it's being able to replace a component the moment it becomes vulnerable, while still delivering what users need.

## What I'll show

The demo is a before and after: it opens on the PHP site, walks through the move in GitHub, and ends with both sites running side by side and the AWS bill.

| Time | On screen | Point |
|------|-----------|-------|
| 0:00–0:30 | **Before:** the `lamp` site running in the browser | PHP2Shell, and the decision to leave PHP instead of waiting for the next patch. |
| 0:30–1:15 | `design/decisions.md` → a module spec → the finished Claude session that regenerated it in Go on `main` → contract tests pass | What I maintain is the specs. None of them mention PHP or Go. |
| 1:15–2:00 | **The move, in GitHub:** the branches page, then Settings → Environments | The PHP code is archived on the `lamp` branch, and its `lamp` environment still holds its SSH deploy secrets. `main` is now the Go version, and its new `aws` environment holds no deploy keys, only a role ARN that GitHub signs in to with OIDC. `lamp` depends on PHP, Apache, Linux and the host for patches; `main` depends on one binary and a short `go.mod`. |
| 2:00–2:45 | **The pipeline:** the `main` workflow file, then a green run in GitHub Actions: tests → accessibility gate → cost gate → OIDC → `cdk deploy` → smoke test → release tag | Same platform and same gates as `lamp`, new target. The SSH key became a short-lived OIDC role. |
| 2:45–3:45 | **After:** both sites side by side in the browser, the same sign-up submitted on each | Same site, same contract tests, new stack. *Pause here for the audience.* |
| 3:45–4:30 | **Cost:** the AWS Billing and Cost Management console showing $0.14 | Hosting costs $0.00. The whole $0.14 is Bedrock playground queries, paid from free credits, and they completed an AWS credit challenge that paid out $20 in more credit. Net, the account is ahead. |
| 4:30–5:00 | — | When the platform is the vulnerability, you don't have to wait for its patch. Write your decisions down and guard them with tests, and you can regenerate the whole solution: the next PHP2Shell is a rebuild you choose, not an emergency you wait out. |

### Tabs to have open, in order

Five minutes leaves no time to navigate, so open these beforehand and flip left to right:

1. **Browser:** the `lamp` site.
2. **Editor:** `design/decisions.md` and one module spec; a terminal with the contract tests already passing on `main`.
3. **GitHub:** the repo's branches page, showing `main` (now Go) and `lamp` kept for archival, with its `lamp-final` tag.
4. **GitHub:** Settings → Environments, with `lamp` and `aws` listed. Open `aws` to show that it has no deploy keys, only a role ARN.
5. **GitHub:** the deploy workflow on `main`, ideally split beside the one on `lamp`.
6. **GitHub Actions:** the latest green run on `main`, with the job graph expanded.
7. **Browser:** the `lamp` and AWS sites in side-by-side windows.
8. **AWS console:** Billing and Cost Management, showing month-to-date and total spend.

Don't trigger a deploy live: a CDK deploy takes longer than the whole slot. Show a completed run, and keep screenshots of each tab in case the conference Wi-Fi or a sign-in fails.

### If asked

- **"What does the cost gate catch?"** A NAT gateway is the classic example, at about $32 a month while idle. Moving MySQL to RDS would put the database in a VPC. A Lambda in a VPC needs a NAT gateway to reach SSM and other AWS APIs, and CDK's default VPC creates one in each availability zone. DynamoDB keeps the site out of a VPC, and the CI test makes sure no NAT gateway sneaks back in.
- **"Were you hit by the Brotli bug?"** Not in anything that ships. `go.sum` has no Brotli module, the repo vendors no decoder, and the Go code never decodes compressed input. CloudFront compresses responses on AWS's side. Only CI touches it, through Node.js, which installs the newest patch release on every run. Checking took one search, which is the point of keeping the dependency list short (takeaway 7).

## What another builder can take away

**1. Small-site PHP's sweet spot is over, and regeneration is the way out.** For years, PHP on cheap shared hosting was the easy default for a small site. With a coding agent, what you maintain can be the specs and decisions rather than the code. When the platform turns against you, you regenerate on a new one. When rebuilding a small site takes a day and a half instead of months, staying on a vulnerable platform stops being the cheap option. The same skills and specs could be packaged to move most standard PHP hosting to a cheaper stack with less to patch.

**2. The decisions are the asset.** Each feature is a module with a spec and HTTP-level contract tests, and a regenerated module is done when it passes them. On the original private app, about 200 decisions carried over to the new platform unchanged, including privacy rules written without reference to any platform.

**3. Why Go, for AI-assisted development:**
- Static types and a fast compiler give the agent an immediate error loop.
- `gofmt` means there is one style to generate.
- The standard library covers HTTP, templates and crypto.
- It builds one binary that runs locally, in a container or on Lambda.
- It's backed by Google.

Go still has supply-chain risk, but it's *visible*: `go.mod` and `go.sum` pin every module and its hash. You can audit it and patch on your own schedule.

**4. "Free tier" is a design constraint, so test it.**
- Price every AWS resource before writing any infrastructure code.
- Turn the expensive ones into a test that fails CI: NAT gateway ~$32/month, load balancer $15–30, a Secrets Manager secret $0.40 each.
- Sign-in, a database, a CDN and WAF all fit inside the always-free allowances.
- My whole AWS bill so far is **$0.14**, for Bedrock playground queries during an AWS credit challenge. Free credits paid for it, and the challenge paid out $20 in more credit.

**5. On a cloud account, cost and security can pull against each other.** AWS's new Free plan, the one that can never charge your card, blocked creating the GitHub OIDC provider. Keyless deploys required upgrading to the Paid plan, which can't be undone. I chose security, and replaced the Free plan's spending guarantee with budgets, anomaly detection and the CI cost gate.

**6. Keep the pipeline and change everything under it.** Moving from SSH to cloud deploys didn't mean leaving GitHub Actions. The gates carried over: tests, accessibility checks with Playwright + axe, deploy, smoke test, release tag.

**7. A short, visible dependency list makes "are we affected?" a one-minute question.** About a week after PHP2Shell, a vulnerability in the widely bundled Brotli decompression library came out. The advice was to search vendored code, container images, `node_modules`, Python packages and gems for bundled copies, upgrade everything that has a fix, and rebuild anything that bundles one. For the Go version, the check was one search:
- **Shipped code:** no Brotli module in `go.sum`, no vendored decoder in the repo, and nothing in the code reads `Content-Encoding`. Go's standard library has no Brotli decoder.
- **The edge:** CloudFront compresses responses itself, inside AWS's own infrastructure.
- **CI only:** Node.js, which bundles Brotli, runs the tests and the CDK CLI, and installs the newest patch release on every run. It only fetches from trusted sources.

On the PHP version, the same question means asking the host about Apache and its modules, and waiting. Regeneration doesn't make you immune to the next bug, but a stack whose whole dependency list fits on one screen lets you answer quickly, on your own schedule.

**8. The same idea scales to enterprise systems, if the investment moves to design and tests.** A small site can be regenerated whole. A larger system can't, but its components can, one at a time, if each one's behavior is pinned down well enough to rebuild it on different dependencies:
- **Requirements and decisions** written without naming a platform, so they survive a change of language, framework or host.
- **Constraints made explicit:** security headers, data retention, cost ceilings and accessibility, each checked automatically rather than remembered.
- **Contract tests at the component's boundary,** so a rebuilt component is done when it passes them, whatever it's built on.

That makes regeneration a form of preparedness for software supply chain risk. When a runtime, library or vendor becomes a liability, whether it's vulnerable, abandoned, slow to patch or untrusted, a team that has invested in its specs and tests has a credible exit: regenerate and redeploy the component on other dependencies, instead of waiting on the one that failed. The patching process doesn't go away, but it stops being the only option. The work shifts from writing code to keeping decisions, constraints and tests current, and that is what coding agents make worth doing now.

**Bugs from the original port that no spec predicted, which carry over to other projects:**
- `Referrer-Policy: no-referrer` makes browsers send `Origin: null`, so the CSRF check refused every sign-in.
  - **Why it was new:** the PHP version used session tokens for CSRF protection and never read the `Origin` header. The Go version runs on Lambda without server-side sessions, so it checks that a form's `Origin` matches the site instead. The new stack also sent strict security headers, including `no-referrer`. Under that policy, browsers hide the origin of cross-site and same-site form posts alike and send `Origin: null`, so every sign-in looked like a forgery.
  - **The fix, in two parts:**
    - **The header:** send `Referrer-Policy: same-origin` instead (D-019). Pages on the site still get the real `Origin`, so the check works, and other sites still get no referrer at all.
    - **The rule:** write it down as an outcome, not a mechanism (D-025): a form post that a browser marks as coming from another origin, including another subdomain, gets 403 and changes nothing. Each platform enforces it its own way. The PHP version checks a signed token plus the browser's `Sec-Fetch-Site` and `Origin` headers. The Go version can use Go 1.25's built-in `http.CrossOriginProtection`, which needs no tokens or sessions. The shared contract tests check the outcome: same-origin sign-in works, and posts from another site or subdomain are refused. So the next mismatch like this fails a test instead of every sign-in.
- Warm Lambdas keep secrets they read at cold start.
  - **Why it was new:** PHP starts fresh on every request and rereads its config each time, so a changed password or key takes effect right away. A Go binary on Lambda keeps running between requests, and it reads its secrets from SSM once, at startup, to keep requests fast and cheap. After a secret changed, warm instances kept using the old value, and only new instances picked up the new one.
  - **The fix:** treat a secret change as a deploy. Redeploying the function replaces every warm instance, so they all start again with the new value.
- WebKit reports a native `<select>` as a CSP violation under a strict policy.
  - **Why it was new:** the PHP site on shared hosting had no Content Security Policy. The Go version sends a strict one, and the Playwright gate fails on any CSP violation in any browser. Safari's engine, WebKit, styles its built-in dropdown in a way that a strict `style-src` reports as a violation, even though the page's own code does nothing wrong.
  - **The fix:** have the test ignore that one known WebKit report, rather than loosening the policy for every browser. This demo avoids the problem entirely: its only choice from a list, the head count, is a number field instead of a `<select>`, and the gate passes in all three engines with the strict policy.

**Four things to apply even if you never use this code:**
1. Maintain specs and decisions, not just code, so a platform move is a rebuild rather than a rewrite.
2. Put your cost ceiling in CI, next to your tests.
3. Use OIDC and delete your deploy keys. It works from the CI/CD platform you already have.
4. Count what you actually depend on for patches: language runtime, web server, OS and host. Pick a stack where that list is short and visible.

## Built with

- **Claude Code (Claude Opus) in VS Code:** specs, decision log, both implementations, the CDK stack, workflows, and cost and security tests.
- **GitHub Actions + GitHub OIDC:** CI/CD for both branches.
- **`lamp`:** PHP 8, MySQL and Apache on shared hosting, deployed with rsync over SSH.
- **`main` (AWS):**
  - Go.
  - Lambda (arm64, `provided.al2023`) behind CloudFront's flat-rate Free plan.
  - DynamoDB.
  - SSM Parameter Store.
  - AWS CDK in Go.
  - AWS Budgets and Cost Anomaly Detection.
- **Playwright + axe-core:** accessibility and layout checks that gate every deploy.

## References

**PHP2Shell overview**
- Wikipedia, "PHP2Shell": https://en.wikipedia.org/wiki/PHP2Shell
- The PHP Foundation, release announcement (2026-10-21): https://thephp.foundation/blog/2026/10/21/todays-php-security-release-and-how-were-strengthening-php-security/

**Discovery, the exploit's heap shaping, and AI's part in it**
- **Calif, "PHP2Shell: A tale of two PHP remote vulnerabilities"**, by Ryan Austin (2026-10-21), the discoverers' write-up: https://calif.io/php2shell/
  - **How it was found:** an expert who knows PHP's bug history steered LLM agents through source review of PHP's memory manager. "No special harness, or access to models that weren't generally available to everyone were used in the initial bug discovery."
  - **How the exploit works:** it drives a chosen allocation into `memory_limit`, inflates the heap with empty upload fields, then reclaims freed memory: "We therefore control where PHP places a pointer and its associated type information."
- **Wiz Research, "PHP2Shell: AI uncovers critical RCE vulnerabilities affecting 70% of the Internet"** (2026-10-21): https://www.wiz.io/blog/php2shell-php-rce
  - "Frontier models were critical to making this process practical."
  - "LLMs can help adapt an exploit to different PHP versions, configurations and memory layouts."
  - "A single researcher completed in a week what previously would have taken a world-class team months of work."
- **OpenAI, Patch the Planet:** https://openai.com/index/coordinated-disclosure-of-critical-vulnerabilities/

**Exploitation before disclosure**
- **AWS Security Blog, "Amazon Threat Intelligence identifies PHP2Shell exploitation prior to public disclosure in MadPot"**, by CJ Moses (2026-10-21): https://aws.amazon.com/blogs/security/amazon-threat-intelligence-identifies-php2shell-exploitation-prior-to-public-disclosure-in-madpot/ ([archived copy](https://web.archive.org/web/20261021233344/https://aws.amazon.com/blogs/security/amazon-threat-intelligence-identifies-php2shell-exploitation-prior-to-public-disclosure-in-madpot/))
  - The first attempt was logged on 2026-10-17, four days before disclosure.
  - Proofs of concept appeared about 12 hours before the embargo lifted.
  - Detection tip: look for POST form uploads containing `PHP_SESSION_UPLOAD_PROGRESS`.

**Fake and non-working exploits, and failed attack traffic**
- **Flare, "'PoC Yes. RCE No.': What the Underground Actually Thinks of the Public PHP2Shell Exploits"** (2026-10-23): https://flare.io/learn/resources/blog/php2shell-exploits-underground-analysis
  - "Github is saturated with low quality vibe coded untested exploits."
- **The Record, "Reports of PHP bug exploitation attempts grow as experts downplay viability of exploits"** (2026-10-23): https://therecord.media/php-php2shell-exploitation-attempts-pocs
- **AhnLab ASEC**, analysis of the public PoC and "mitigation" repos: https://asec.ahnlab.com/en/95944/
- **CrowdStrike**, "PHP2Shell: Obstacles to Weaponizing Likely Constrain Immediate Exploitation": https://www.crowdstrike.com/en-us/blog/obstacles-to-weaponizing-php2shell-likely-constrain-exploitation/
- **The Hacker News**, "PHP2Shell Exploit Attempts Hit Millions as Attackers Try to Deploy Mirai Malware": https://thehackernews.com/2026/10/php2shell-exploit-attempts-hit-millions.html
- **BankInfoSecurity**, "Waiting for the PHP2Shell Onslaught as PoCs Proliferate": https://www.bankinfosecurity.com/waiting-for-php2shell-onslaught-as-pocs-proliferate-a-33190

**Accessibility and browser testing with Playwright and axe**
- **Playwright**, getting started: https://playwright.dev/docs/intro
  - Drives Chromium, Firefox and WebKit (Safari's engine) from one test suite.
- **Playwright, "Accessibility testing"**, the guide to running axe inside Playwright tests: https://playwright.dev/docs/accessibility-testing
- **axe-core**, Deque's open-source accessibility rules engine: https://github.com/dequelabs/axe-core
- **`@axe-core/playwright`**, the package that connects the two: https://www.npmjs.com/package/@axe-core/playwright
- **Deque University, axe rule descriptions**, explaining each violation axe reports and how to fix it: https://dequeuniversity.com/rules/axe/
- **W3C, "How to Meet WCAG"**, the accessibility guidelines axe checks against: https://www.w3.org/WAI/WCAG22/quickref/

**About the format**
- AI Tinkerers, "What makes a great demo": https://aitinkerers.org/p/what-makes-a-great-demo-at-ai-tinkerers
