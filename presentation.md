# PHP2Go: talk notes

**PHP2Go: After PHP2Shell, Regenerating a PHP Site in Go on AWS with Claude, for $0.14**

A 5-minute live demo for AI Tinkerers. There are no slides: everything is shown in the editor, the terminal, GitHub Actions and the AWS console. The demo repo is this one; see the [README](README.md).

I also used Claude to help develop this presentation, including these notes.

## In one paragraph

PHP2Shell showed what PHP on shared hosting really costs a small site. Your security depends on a hosting provider's patch schedule and on a thinly staffed upstream. I'd built a private app on that stack with Claude Code. When PHP2Shell landed, I had Claude regenerate it in Go on AWS from its specs and decision log, in about a day and a half. This repo repeats that move on a typical small PHP site, with a `lamp` branch and an `aws` branch, so the steps are reusable.

The talk rests on three points:
1. **Lower cost:** paid shared hosting → AWS at a $0 target, enforced in CI.
2. **Better security:** a short, visible patch surface, and deploys without stored keys (SSH → OIDC).
3. **The same pipeline:** GitHub Actions deploys both branches, with the same gates in the same order.

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

| Time | On screen | Point |
|------|-----------|-------|
| 0:00–0:30 | The `lamp` site running | PHP2Shell, and the decision to leave PHP instead of waiting for the next patch. |
| 0:30–1:30 | `design/decisions.md` → a module spec → Claude regenerates the module on `aws` → contract tests pass | What I maintain is the specs. None of them mention PHP or Go. |
| 1:30–2:30 | **Security:** what each branch depends on for patches, then the two deploy jobs side by side | `lamp` depends on PHP, Apache, Linux and the host. `aws` depends on one binary and a short `go.mod`. SSH key → OIDC. |
| 2:30–3:30 | **Cost:** the AWS bill, then the CI cost test | Hosting costs $0.00. A NAT gateway would cost $32 a month, so CI refuses to deploy one. |
| 3:30–4:30 | **Same pipeline:** a GitHub Actions run on each branch | Same platform, same gates, new target. |
| 4:30–5:00 | — | Write your decisions down, and the next PHP2Shell is a rebuild, not an emergency. |

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
- My whole AWS bill so far is **$0.14**, for Bedrock playground queries during an AWS credit challenge. Free credits paid for it, and the challenge paid $20 back.

**5. On a cloud account, cost and security can pull against each other.** AWS's new Free plan, the one that can never charge your card, blocked creating the GitHub OIDC provider. Keyless deploys required upgrading to the Paid plan, which can't be undone. I chose security, and replaced the Free plan's spending guarantee with budgets, anomaly detection and the CI cost gate.

**6. Keep the pipeline and change everything under it.** Moving from SSH to cloud deploys didn't mean leaving GitHub Actions. The gates carried over: tests, accessibility checks with Playwright + axe, deploy, smoke test, release tag.

**One honest note on cost:** I'm keeping my shared hosting plan, but only for other things on it; the app itself could be spun off entirely. Email still goes through that plan's SMTP relay for convenience, which has nothing to do with PHP or Go, and Amazon SES is the likely next step.

**Bugs no spec predicted, which carry over to other projects:**
- `Referrer-Policy: no-referrer` makes browsers send `Origin: null`, so the CSRF check refused every sign-in.
- Warm Lambdas keep secrets they read at cold start.
- WebKit reports a native `<select>` as a CSP violation under a strict policy.

**Four things to apply even if you never use this code:**
1. Maintain specs and decisions, not just code, so a platform move is a rebuild rather than a rewrite.
2. Put your cost ceiling in CI, next to your tests.
3. Use OIDC and delete your deploy keys. It works from the CI/CD platform you already have.
4. Count what you actually depend on for patches: language runtime, web server, OS and host. Pick a stack where that list is short and visible.

## Built with

- **Claude Code (Claude Opus) in VS Code:** specs, decision log, both implementations, the CDK stack, workflows, and cost and security tests.
- **GitHub Actions + GitHub OIDC:** CI/CD for both branches.
- **`lamp`:** PHP 8, MySQL and Apache on shared hosting, deployed with rsync over SSH.
- **`aws`:**
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

**About the format**
- AI Tinkerers, "What makes a great demo": https://aitinkerers.org/p/what-makes-a-great-demo-at-ai-tinkerers
