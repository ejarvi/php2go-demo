# Platform: Go on AWS

This branch's implementation of the design in [design/](../design/). The design doesn't name a platform (D-002); this folder records the choices this implementation makes, and the one-time setup it needs (D-014). The PHP version it replaced is on the `lamp` branch.

## Choices

| | |
|---|---|
| **Language** | Go, standard library first: `net/http`, `html/template`, `log/slog`, plus the AWS SDK and the Lambda adapter |
| **Storage** | One DynamoDB table, provisioned 5 + 5 units, inside the always-free allowance |
| **Hosting** | CloudFront (flat-rate Free plan) → a Lambda function URL (arm64, not in a VPC) |
| **Infrastructure** | AWS CDK in Go, in `infra/`, with tests that fail on anything that costs money while idle (D-020) |
| **Deploy** | GitHub Actions → GitHub OIDC (no stored keys) → `cdk deploy` |
| **Secrets** | GitHub environment secrets, copied by each deploy into SSM Parameter Store (free standard parameters). The Lambda reads them at start-up |
| **Same-origin forms (D-025)** | Go's built-in `http.CrossOriginProtection`, which reads the browser's `Sec-Fetch-Site` header (or compares `Origin`) and needs no tokens. CloudFront forwards both headers |
| **Clean-up** | DynamoDB's TTL deletes expired sessions, sign-ups 30 days after their event, and activity after a year |

## Layout

```
cmd/site/        the site: a local web server, or a Lambda behind CloudFront
cmd/secrets/     copies the GitHub secrets into SSM during a deploy
internal/site/   every page and form; one file per module, templates in templates/
internal/store/  DynamoDB, plus an in-memory store for Go tests
internal/platform/  settings, security headers, the CloudFront origin check, release ID
web/static/      the stylesheet, built into the binary
infra/           AWS CDK in Go: the GitHub sign-in stack, the site stack, and their tests
tests/contract/  the acceptance tests from design/modules/ (D-003), shared with the lamp branch
tests/gate/      the accessibility and layout checks (D-018)
```

## How it runs

```
Browser → https://dxxxxxxxxxxxx.cloudfront.net   (CloudFront Free plan: HTTPS, WAF, caches /assets/)
            │  adds a secret header, so only CloudFront can reach the function
            ▼
          Lambda function URL → Go binary (arm64, 128 MB)
            ├── DynamoDB table php2go-data
            └── SSM Parameter Store /php2go/: organizer password, session secret
```

The same binary runs locally (`go run ./cmd/site`) with an in-memory store, or with DynamoDB Local in CI.

## How a deploy works

On every push to `main`:
1. **Test and gate:** Go tests and the infrastructure tests (cost traps, the deploy role's limits, the Free plan's rules). Then the shared acceptance tests and the accessibility gate run against two copies of the site sharing DynamoDB Local (D-027 lets you switch these off).
2. **Build** a Linux arm64 binary with the release ID built in (D-016).
3. **Sign in to AWS** with GitHub OIDC, and check the account matches `AWS_ACCOUNT_ID`.
4. **Copy the secrets** into SSM Parameter Store.
5. **`cdk diff`, then `cdk deploy`** the `Php2go` stack.
6. **Smoke test:** `/health` reports the new release, the headers are there, and the function URL refuses direct requests (403).
7. **Tag** the commit with its release ID.

Until the `aws` environment has `AWS_ROLE_ARN`, the deploy job skips itself with a notice and the run stays green. With `.github/dry-run` in the commit, only `cdk diff` runs.

## The aws GitHub environment

Every value here can be either a variable or a secret; when both exist, the secret wins. GitHub shows a secret's value as `***` in every log. A public repo's Actions logs are public too, so anything that identifies your account belongs in secrets.

| Name | Store as | Value |
|---|---|---|
| `AWS_ROLE_ARN` | secret | The deploy role's ARN, printed by step E3 |
| `AWS_ACCOUNT_ID` | secret | The 12-digit account ID (step B1) |
| `AWS_REGION` | variable | The region, such as `us-east-2` |
| `ORIGIN_SECRET` | secret | 64 random hex characters (step F2). CloudFront sends it; the site refuses requests without it |
| `ORGANIZER_PASSWORD` | secret | The organizers' shared password (D-023) |
| `SESSION_SECRET` | secret | 64 random hex characters (step F2) |
| `CLUB_NAME` | variable | Shown in the header and page titles |
| `CLUB_TIMEZONE` | variable | A [tz database name](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones), such as `America/Los_Angeles` |
| `SITE_URL` | variable or secret | The CloudFront address, once the first deploy prints it (step G3). Optional |
| `FREE_PLAN_WEB_ACL_ARN` | secret | The web ACL CloudFront's Free plan attaches (step G4). Set it before the next deploy after subscribing |

## One-time setup

Steps marked 🔒 involve credentials: do them yourself, and never paste secret values into a chat or a file. Every value in angle brackets is your own and never belongs in the repository (D-024).

**Already have an AWS organization on the Paid plan with advanced features on?** Skip Part A and start at **Part B**: a new member account, its own GitHub OIDC provider and deploy role, and a new `aws` environment.

### Part A. A new AWS account (skip if you have one)

AWS changes its sign-up often. Check https://aws.amazon.com/free on the day.

1. 🔒 **Prepare:**
   - an email address only you read, for the account's owner;
   - a password manager;
   - an MFA device: a passkey, or an authenticator app.
2. 🔒 **Sign up** at https://aws.amazon.com/start-here/ (**Sign up for AWS**). It creates an **AWS Builder ID** (your personal sign-in) and a **project** holding one AWS account, on the **Free plan**, with credits. Note the project's **region**.
3. 🔒 **Secure the Builder ID:** register MFA, and add a recovery email that has MFA too.
4. **Optional, before upgrading:** the console's credit activities can add up to $100 more. Delete anything an activity creates once it's done, so nothing keeps running.
5. 🔒 **Upgrade to the Paid plan, then activate advanced features.** The Free plan's guardrails deny creating a GitHub OIDC provider, so keyless deploys need both.
   - Upgrading can't be undone, and remaining credits keep paying the bill.
   - Activation makes you the owner of the organization's **management account**: set its root password from the **Sign in using root user email → Forgot your password?** flow, then add two MFA devices.
   - The spend limit is removed. Budgets and Cost Anomaly Detection replace it (step 7).
   - Use AWS's own documentation for these steps ([Upgrade your account](https://docs.aws.amazon.com/accounts/latest/reference/upgrade-account.html), [Activate advanced features](https://docs.aws.amazon.com/accounts/latest/reference/activate-advanced-features.html)). AI assistants have contradicted it.
6. **Check the guardrails.** In the management account, open **AWS Organizations → Policies → Service control policies**:
   - **Keep** the policy that blocks leaving the organization, and the region restriction.
   - **Allow CloudFront's flat-rate plans:** the region restriction's `us-east-1` statement must list `pricingplanmanager:*` in its `NotAction`, after `pricing:*`. Without it, subscribing to the Free plan (step G4) fails.
   - **Leave** the dormant spend-limit policies unattached. If a deploy ever fails with "explicit deny in a service control policy", check whether one got attached.
7. **Cost guards,** in the management account's **Billing and Cost Management** (they cover every account in the organization):
   - **Budgets → Create budget → Use a template → Zero spend budget**, to your everyday email;
   - **Billing preferences → Alert preferences**: turn on Free Tier alerts;
   - **Cost Anomaly Detection:** keep the default services monitor, with an email alert at $1;
   - **Account → Alternate contacts:** your everyday email for Billing, Operations and Security.

### Part B. A new account for this site

With advanced features on, a new project in your organization is a **member account**. AWS Settings → **Create project** would instead create a whole new organization, with its own billing and Free plan, so don't use it.

1. **Create the member account,** in a **management account** session signed in with your Builder ID (not root):
   - **AWS Organizations → AWS accounts → Add an AWS account → Create an AWS account.**
   - **Account name:** for example `demo`. It's only a label; the code uses the account ID.
   - **Email:** one no other AWS account uses, even a closed one, which keeps its email for 90 days. A plus-address such as `you+awsdemo@example.com` works with most mail providers.
   - **IAM role name:** leave it as `OrganizationAccountAccessRole`.
   - Refresh **AWS accounts** until the new account shows **Active**. Its account ID is `AWS_ACCOUNT_ID`. It isn't a credential, but keep it out of public places.
   - The account sits under the organization root, so the organization's guardrails and zero-spend budget cover it.
2. **Open it with Switch role.** A member account created this way isn't listed in your Builder ID's sessions, and an Identity Center assignment copied from an older project won't work.
   - Optional: in the account menu (top right), **Turn on multi-session support** to keep several accounts open in one browser.
   - In the management account session: account menu → **Switch role**.
   - **Account ID:** the new account's. **IAM role name:** `OrganizationAccountAccessRole`. **Display name:** `demo`, with a color.
   - The account menu then shows `demo` in that color, with administrator access. The consoles look identical, so check it before changing anything. Next time, `demo` is listed under **Switch role**.

### Part C. Tools on your PC

1. **Install,** in an **admin** PowerShell, then reopen the terminal and VS Code:
   ```powershell
   winget install --id GoLang.Go -e
   winget install --id OpenJS.NodeJS.LTS -e
   winget install --id Amazon.AWSCLI -e
   ```
   Then install the CDK CLI: `npm install -g aws-cdk@2.1146.1`.
   - Run the installs from an admin terminal. A silent install started from a normal one can hang invisibly and block later installs with exit code 1618 until a restart.
2. **Sign the CLI in to the management account** with your Builder ID. At the account prompt, choose the management account:
   ```powershell
   aws login --profile jarvi --region us-east-2
   ```
   If `aws login` offers to set up an AI agent's access to the account, answer **n**.
3. **Add a profile that switches into the new account.** Open `notepad $HOME\.aws\config` and add:
   ```ini
   [profile php2go]
   source_profile = jarvi
   role_arn = arn:aws:iam::<new account ID>:role/OrganizationAccountAccessRole
   region = us-east-2
   ```
   Check it shows the **new** account's ID: `aws sts get-caller-identity --profile php2go`.
4. **Give CDK the credentials.** CDK doesn't renew these itself, so before each CDK command, in the same terminal:
   ```powershell
   aws configure export-credentials --profile php2go --format powershell | Invoke-Expression
   ```
   These are temporary credentials (about an hour), held only in that terminal. Never print `$env:AWS_SECRET_ACCESS_KEY`. If they expire, run `aws login --profile jarvi` again, then this step.

### Part D. Bootstrap CDK

Once per account and region. It creates a small S3 bucket for uploading the Lambda, and the roles CDK deploys through (under $0.01 a month):
```powershell
cdk bootstrap aws://<account ID>/<region>
```
Watch it under **CloudFormation → Stacks → CDKToolkit**.

### Part E. 🔒 Connect GitHub to AWS

1. **Find the GitHub subject the role will trust.** This repo uses GitHub's immutable subject format, which adds the owner's and the repo's numeric IDs:
   ```powershell
   gh api repos/<owner>/<repo>/actions/oidc/customization/sub
   ```
   - If it shows `"use_immutable_subject":true` with a `sub_claim_prefix`, the subject is that prefix plus `:environment:aws`, for example `repo:<owner>@<owner id>/<repo>@<repo id>:environment:aws`.
   - Otherwise it's `repo:<owner>/<repo>:environment:aws`.
   - In the other project, the plain form was refused on the first CI run because the repo used the immutable format, so check rather than guess.
2. **Deploy the GitHub stack from your PC,** in the terminal from step C4. CI can't create the role it signs in with, so this one stack is deployed by hand:
   ```powershell
   cd infra
   $env:AWS_ACCOUNT_ID = '<account ID>'
   $env:AWS_REGION = '<region>'
   $env:GITHUB_OIDC_SUBJECT = '<subject from step 1>'
   cdk deploy Php2goGitHub
   ```
   - It creates the account's **GitHub OIDC provider** and the role **`php2go-github-deploy`**.
   - The role trusts only that subject, so only jobs in this repo's `aws` environment can use it. Sessions last at most an hour.
   - The role can do just two things: hand off to CDK's bootstrap roles, and write `/php2go/` parameters in SSM. `infra/cost_test.go` checks this.
   - **If the account already has a GitHub OIDC provider,** creating a second one fails, since an account can have only one. Set `$env:EXISTING_OIDC_PROVIDER_ARN` to its ARN (**IAM → Identity providers**) and deploy again.
3. **Note the output `DeployRoleArn`.** That's `AWS_ROLE_ARN`.

### Part F. 🔒 The aws GitHub environment

1. **Create it:** repo → **Settings → Environments → New environment** → `aws`. Under **Deployment branches and tags**, choose **Selected branches** and add `main`. If the first push already created an `aws` environment, open that one instead.
2. **Generate the two random secrets** straight to the clipboard, one at a time, and paste each into its secret:
   ```powershell
   -join ((1..32) | ForEach-Object { '{0:x2}' -f (Get-Random -Maximum 256) }) | Set-Clipboard
   ```
   Run it once for `ORIGIN_SECRET` and again for `SESSION_SECRET`.
3. **Add the rest** from [the table above](#the-aws-github-environment): `AWS_ROLE_ARN`, `AWS_ACCOUNT_ID` and `ORGANIZER_PASSWORD` as secrets, and `AWS_REGION`, `CLUB_NAME` and `CLUB_TIMEZONE` as variables.

### Part G. The first deploy

1. **Optional dry run:** add an empty `.github/dry-run`, then commit and push. The run signs in, checks the account and shows `cdk diff`, but changes nothing. Then delete the file, and commit and push.
2. **The real deploy:** push to `main`, or run **Actions → Deploy → Run workflow**. The first `cdk deploy` takes several minutes, mostly creating the CloudFront distribution.
3. **The address:** the run's summary prints `https://dxxxxxxxxxxxx.cloudfront.net`. Open it, then sign in from the **Organizers** link with `ORGANIZER_PASSWORD`. Add the address as `SITE_URL`.
4. **Put CloudFront on the Free plan** ($0 a month, with no overage charges):
   - In the project's console: **CloudFront → the distribution → Pricing plan** (or **Flat-rate plans**) **→ Free**.
   - Subscribing attaches an AWS WAF web ACL named `CreatedByCloudFront-…`, which a plan distribution must keep. Copy its ARN from **WAF & Shield → Web ACLs** (choose **Global (CloudFront)**) into the secret `FREE_PLAN_WEB_ACL_ARN`, **before the next deploy**. Otherwise that deploy tries to detach it and fails.
   - Subscribing also requires the price class **All**, which the stack already uses. Another project's deploy failed on exactly this: "Distributions with the Free pricing plan can't have the following features: Price class".
   - After any console change to the distribution, compare it with `infra/site.go`.
   - If the plan won't subscribe, check the guardrail in step A6. Pay-as-you-go's always-free allowance (1 TB and 10 million requests a month) also covers the site.
5. **Redeploy** (push, or **Run workflow**) so the stack picks up `SITE_URL` and the web ACL.
6. **Check the bill the next day:** **Billing → Bills** should show $0.00 for this project.

## If something goes wrong

- **"explicit deny in a service control policy":** an organization guardrail blocked the action. See step A6. Before activation, the Free plan's guardrail is what blocks the OIDC provider.
- **The deploy can't assume the role** (`Not authorized to perform sts:AssumeRoleWithWebIdentity`): the role's subject doesn't match. Check step E1, then redeploy `Php2goGitHub`.
- **CDK says no credentials are configured:** run steps C2 and C4 again; the exported credentials last about an hour.
- **The smoke test sees the old release:** CloudFront and Lambda usually switch within a minute. The test waits three. Run it again before digging deeper.
- **The function URL answers 200 instead of 403:** the Lambda isn't checking `ORIGIN_SECRET`. Make sure the secret is set, then redeploy.

## Changing the organizers' password

Change the `ORGANIZER_PASSWORD` secret and run the Deploy workflow. The deploy writes the new value to SSM and replaces the Lambda, which reads secrets once at start-up. Every existing session ends (D-023).

## Running it locally

You need Go (version in `go.mod`) and Node 24:

```sh
CLUB_NAME="Test Club" CLUB_TIMEZONE=America/Chicago ORGANIZER_PASSWORD=secret \
  SESSION_SECRET=0123456789abcdef0123456789abcdef go run ./cmd/site    # in-memory store, http://127.0.0.1:8080
go test ./... && (cd infra && go test ./...)                            # Go and infrastructure tests
```

The acceptance tests and gate need two copies of the site sharing DynamoDB Local, as in `.github/workflows/deploy.yml`.
