# Delivery

How a change gets from the repository to the live site. The steps and checks are part of the design. *Where* the site runs is not (D-002).

## The pipeline (D-013, D-017)
Every push to `main` runs one GitHub Actions workflow. Deploys run one at a time in push order, and a running deploy is never cancelled.

1. **Test:** the unit tests, and every module's acceptance tests run against the site started locally with empty storage (D-003).
2. **Gate:** accessibility and layout checks at phone, laptop and wide-screen sizes, in three browser engines (D-018). On an implementation that describes its hosting as code, the cost check runs here too (D-020).
3. **Deploy:** the site and any storage changes it needs. Storage changes run before the new code goes live and must work with both the old and the new code.
4. **Smoke test:** `GET /health` on the live site must return 200 with the new release ID (D-021).
5. **Tag:** the commit is tagged with its release ID, and a GitHub Release is created (D-016).

A failure at any step stops the steps after it. The live site keeps running the last good release.

## Rolling back (D-016)
Revert the commit and push. There's no separate rollback button.

## Secrets and settings (D-015, D-024)
- **Never in the repository.** Secrets are GitHub secrets, scoped to the deploy environment, or the platform's own secret store.
- **The list:** the organizers' password, the secret used for hashing session codes, the club's name and time zone, and whatever the platform needs to accept a deploy.
- **Deploy credentials (D-015):** short-lived credentials are preferred to stored keys. Where a stored key can't be avoided, it can do nothing but deploy this one site.

## Where the site is deployed (D-024)
- **The repository never says where the site runs.** It has no host names, server paths, site addresses, account numbers or regions in its code, workflows, docs or examples.
- **All of that lives in the GitHub environment the deploy job uses:**
  - **Environment variables** for the non-secret settings, such as the server's host name and deploy path, or the cloud account and region.
  - **Environment secrets** for the secret ones.
- **Fail early:** the workflow reads these by name and stops with a clear error if one is missing.
- **Clones and forks:** anyone can deploy their own copy by creating the same environment with their own values.
- **Docs use placeholders.** Setup checklists name the variables, never their values: `DEPLOY_HOST`, not a real server name.
- **The live address** is printed in the deploy log and set as the environment's URL in GitHub, not committed.

## One-time setup (D-014)
Anything that has to be done by hand, such as creating the hosting account or filling in the GitHub environment, is done once. It's written as a numbered checklist in the implementation's `platform/` folder, using placeholders for every host-specific value (D-024). After that, nobody changes the live site except through the pipeline.
