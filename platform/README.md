# Platform: LAMP on shared hosting

This branch's implementation of the design in [design/](../design/). The design doesn't name a platform (D-002); this folder records the choices this implementation makes, and the one-time setup it needs (D-014).

## Choices

| | |
|---|---|
| **Language** | PHP 8.1 or later, with no framework and no Composer packages |
| **Storage** | MySQL 8, through PDO |
| **Web server** | Apache, with `mod_rewrite` sending every request to `index.php` |
| **Hosting** | Shared hosting with SSH access, where PHP runs as your own account (PHP-FPM or suEXEC, as on most control-panel hosts) |
| **Deploy** | GitHub Actions → rsync over SSH, with three keys that can each do one job and nothing else |
| **Secrets** | The GitHub environment `lamp`. The deploy writes them to `config.local.json` in a private folder outside the web root, readable only by your account |
| **Same-origin forms (D-025)** | Every form carries a CSRF token signed from a random cookie, so it needs no session. A POST is also refused if the browser's `Sec-Fetch-Site` or `Origin` header shows it came from another origin |
| **Clean-up** | Expired sign-ups, sessions and activity entries are deleted by occasional requests, since there's no scheduler to set up from the repository |

## Layout

```
public/          the web root: index.php (the front controller), .htaccess, assets/
src/             the code, one file per concern; handlers/ has one file per module
templates/       the pages
migrations/      storage changes, applied in name order by bin/migrate.php
bin/             migrate.php, and serve-local.sh for tests
platform/        this file, and migrate-gate.php for the host
tests/contract/  the acceptance tests from design/modules/ (D-003), which every branch shares
tests/gate/      the accessibility and layout checks (D-018)
```

On the host, only `public/` goes in the web root. The code goes in a private folder outside it:

```
<web root>/                    the domain's document root: public/, plus code-path.php
  .deploy-target               marker: proves to the deploy it has the right folder
~/private/php2go/              APP_DIR: never served
  .app-target                  marker
  config.local.json            settings and secrets, written by every deploy
  migrate-gate.php             the only thing the migrate key can run
  releases/r12+abc1234/        each release's code; the newest five are kept
```

## How a deploy works

Each SSH key is limited on the host, in `~/.ssh/authorized_keys`, to one job:

| Key | Can only | How it's limited |
|---|---|---|
| **web** | rsync into the web root | `rrsync`, which confines rsync to one folder |
| **app** | rsync into `APP_DIR` | `rrsync` |
| **migrate** | run `migrate-gate.php`, which accepts one command: `apply <release>` | a forced `command=` |

A leaked key can't open a shell, read other folders or run anything else (D-015). Before syncing, the deploy checks that each key really is confined, and stops if one isn't. Then:

1. **The app key** uploads the new release's code to `APP_DIR/releases/<release>/`, along with `config.local.json` and `migrate-gate.php`. The live site doesn't use any of it yet.
2. **The migrate key** applies the new release's storage changes. These work with both the old and the new code (design/delivery.md).
3. **The web key** mirrors `public/` into the web root, including `code-path.php`, which points at the new release's code. That's the moment the site switches over.
4. **The smoke test** checks that `/health` reports the new release, and the commit is tagged with it.

**Tests and gate (D-026):** off by default, so a deploy takes seconds. To run them first, tick **Run the tests and the accessibility gate** under Actions → Deploy → **Run workflow**. To run them on every push, add a repository variable `RUN_TESTS` set to `true` (Settings → Secrets and variables → Actions → Variables). If they run and fail, nothing is deployed.

**Dry run:** if the file `.github/dry-run` is in the commit, the deploy checks the settings and keys and lists what each sync *would* change, but changes nothing on the host.

## Settings the site reads

The site reads these from environment variables when running locally or in CI. On the host, it reads them from `config.local.json`, which the deploy writes from the GitHub environment.

| Setting | What it is |
|---|---|
| `DB_HOST`, `DB_PORT`, `DB_NAME`, `DB_USER`, `DB_PASSWORD` | The MySQL database. `DB_PORT` defaults to 3306. `DB_HOST` may be a socket, written `localhost:/path/to/mysql.sock` |
| `ORGANIZER_PASSWORD` | The organizers' shared password (D-023) |
| `SESSION_SECRET` | A long random value, used to hash session codes and sign form tokens |
| `CLUB_NAME` | Shown in the header and page titles |
| `CLUB_TIMEZONE` | The club's time zone (D-006) |
| `RELEASE` | The release ID. The deploy writes it into each release's folder (D-016) |
| `FORM_MIN_SECONDS` | The shortest time to fill in a public form before it's treated as a bot (D-011). Defaults to 3; the tests set 0 |

## One-time setup

Do these steps in order. Every value in angle brackets is your own: write it into GitHub where a step says, and never into the repository (D-024). Steps marked 🔒 handle a private key or password: paste it nowhere except where the step says.

The commands for your own computer are for PowerShell on Windows. On macOS or Linux, use `~/.ssh/...` paths, and copy files with `pbcopy < file` or by opening them in an editor.

### A. Your GitHub account

1. **Turn on two-factor authentication:** GitHub → your avatar → **Settings** → **Password and authentication**. Anyone who controls your GitHub account can deploy to the site, so it's the real security boundary.
2. **Get an email when a deploy fails:** **Settings** → **Notifications** → **System** → **Actions** → turn on **Email**, and tick **Only notify for failed workflows**.

### B. The host

3. **Check the hosting:** SSH access, `rsync`, MySQL 8, and PHP 8.1 or later with the `pdo_mysql` and `mbstring` extensions. Set the PHP version in the control panel for both the web and the command line.
4. **Find the SSH server name, port and user.** Use the server's own host name, which the control panel usually shows, not your domain: it keeps working whatever happens to DNS. These become `SSH_HOST`, `SSH_PORT` (only if it isn't 22) and `SSH_USER`.
5. **Set up the domain or subdomain** for the site in the control panel, turn on HTTPS for it (AutoSSL or Let's Encrypt), and note its **document root**, such as `/home/<user>/public_html` or a subdomain's folder. That's `<web root>` below.
6. **Create the MySQL database and user** in the control panel:
   - **The database:** on MySQL 8. The panel may add a prefix to the name, such as `<user>_club`. Keep the default collation.
   - **The user:** give it a long random password from the panel's generator, and all privileges on this database only.
   - **Note four things:** the database name (with any prefix), the user, the password, and the database host *exactly* as the panel shows it for MySQL 8. That might be `localhost`, a host name, or a socket like `localhost:/path/to/mysql.sock`.
7. **Find two paths** in an SSH session (the control panel's web terminal works):
   ```sh
   which php        # the PHP to run migrations with; check it with: php -v
   which rrsync     # the rsync confinement script, often /usr/bin/rrsync
   ```
   If `php -v` isn't 8.1 or later, look for a versioned binary (the control panel's PHP settings usually name it). If there's no `rrsync`, ask the host for it: it's part of rsync.
8. **Create the app folder and the two marker files:**
   ```sh
   mkdir -p ~/private/php2go
   chmod 700 ~/private ~/private/php2go
   touch ~/private/php2go/.app-target
   touch <web root>/.deploy-target
   cd ~/private/php2go && pwd      # this is APP_DIR, such as /home/<user>/private/php2go
   ```
   The deploy refuses to sync into a folder without its marker. The first deploy removes any placeholder page the host left in the web root.

### C. The host's key → `SSH_KNOWN_HOSTS`

This lets GitHub Actions refuse to connect to an impostor. Read the host's key on the host, then cross-check it over the network from your own computer.

9. **On the host,** in an SSH session:
   ```sh
   cat /etc/ssh/ssh_host_ed25519_key.pub
   ssh-keygen -lf /etc/ssh/ssh_host_ed25519_key.pub
   ```
   The first command prints `ssh-ed25519 AAAA… <comment>`. The second prints its fingerprint, `SHA256:…`.
10. **From your computer,** connect without logging in:
    ```
    ssh -o HostKeyAlgorithms=ssh-ed25519 -p <port> <user>@<ssh-host>
    ```
    It shows `ED25519 key fingerprint is SHA256:…` and asks whether to continue. **Compare the fingerprints, then type `no`.** If they don't match, stop.
    If Windows' `ssh` fails with `unsupported KEX method`, add `-o KexAlgorithms=curve25519-sha256`. Avoid `ssh-keyscan` on Windows, which can fail the same way.
11. **Build the value:** the SSH host name, a space, then the first two fields from step 9, without the trailing comment. If the port isn't 22, write the host as `[<ssh-host>]:<port>`.
    ```
    <ssh-host> ssh-ed25519 AAAA…
    ```

### D. 🔒 The three deploy keys

12. **Make the keys on your computer.** When asked for a passphrase, press Enter twice: GitHub Actions can't type one.
    ```
    ssh-keygen -t ed25519 -C "php2go_web" -f $HOME\.ssh\php2go_web
    ssh-keygen -t ed25519 -C "php2go_app" -f $HOME\.ssh\php2go_app
    ssh-keygen -t ed25519 -C "php2go_migrate" -f $HOME\.ssh\php2go_migrate
    ```
    Each makes two files: the **private** key (no extension), which only GitHub will hold, and the **public** key (`.pub`), which goes on the host.
13. **Show each public key** and copy its single line:
    ```
    Get-Content $HOME\.ssh\php2go_web.pub
    ```
14. **Install the public keys on the host, each confined to its job.** Use an SSH session or the web terminal, not the control panel's key Import, which adds keys *without* the restriction. Replace each `ssh-ed25519 AAAA… php2go_…` with the line you copied, and put in your paths from steps 5, 7 and 8. Keep the single quotes, and use `>>` (append), never `>`, which would wipe the other keys.
    ```sh
    mkdir -p ~/.ssh && chmod 700 ~/.ssh
    echo 'command="<rrsync path> <web root>",restrict ssh-ed25519 AAAA… php2go_web' >> ~/.ssh/authorized_keys
    echo 'command="<rrsync path> <APP_DIR>",restrict ssh-ed25519 AAAA… php2go_app' >> ~/.ssh/authorized_keys
    echo 'command="<php path> <APP_DIR>/migrate-gate.php",restrict ssh-ed25519 AAAA… php2go_migrate' >> ~/.ssh/authorized_keys
    chmod 600 ~/.ssh/authorized_keys
    ```
    `restrict` turns off shells, port forwarding and the rest, and `command=` is the only thing the key can run. `migrate-gate.php` doesn't exist yet; the first deploy uploads it.
15. **Check the lines arrived whole.** Web terminals can drop characters from long pasted lines, and a missing `restrict ` is easy to miss:
    ```sh
    tail -3 ~/.ssh/authorized_keys | cut -c1-100
    ssh-keygen -lf ~/.ssh/authorized_keys
    ```
    Compare the fingerprints with `ssh-keygen -lf $HOME\.ssh\php2go_web.pub` and the other two on your computer. If the control panel ever rewrites `authorized_keys` and drops the `command=…,restrict` prefix, the deploy's safety check will stop with an error.
    SSH also ignores `authorized_keys` if other accounts can write to it, so check its permissions: `chmod 700 ~/.ssh && chmod 600 ~/.ssh/authorized_keys`.
16. **Test each key from your computer** before it goes into GitHub:
    ```
    ssh -i $HOME\.ssh\php2go_web <user>@<ssh-host>
    ssh -i $HOME\.ssh\php2go_app <user>@<ssh-host>
    ssh -i $HOME\.ssh\php2go_migrate <user>@<ssh-host>
    ```
    - **Any message from the host means the key is accepted and confined.** For example, `rrsync` refuses because no rsync command was sent, or PHP can't open `migrate-gate.php`, which the first deploy uploads.
    - **`Permission denied (publickey)` means the host doesn't accept that key.** Fix its line in `authorized_keys` before going on. Once the private keys are deleted in step 20, a bad key can only be replaced, not checked.

### E. 🔒 The GitHub environment

17. **Create the environment:** repo → **Settings** → **Environments** → **New environment** → name it `lamp`. Under **Deployment branches and tags**, choose **Selected branches** and add `main`, so only `main` can deploy.
18. **Add the environment variables.** These aren't credentials, so they stay visible and editable:

    | Variable | Value |
    |---|---|
    | `SSH_HOST` | the SSH server name from step 4 |
    | `SSH_PORT` | only if it isn't 22 |
    | `SSH_USER` | your hosting account's user name |
    | `SSH_KNOWN_HOSTS` | the line from step 11. It's a public key, not a credential |
    | `APP_DIR` | the path from step 8, such as `/home/<user>/private/php2go` |
    | `SITE_URL` | `https://<your domain>` |
    | `CLUB_NAME` | the club's name, such as `Riverside Community Club` |
    | `CLUB_TIMEZONE` | the club's time zone as a [tz database name](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones), such as `America/Chicago` |
    | `DB_HOST` | the database host from step 6, exactly as the panel shows it |
    | `DB_PORT` | only if it isn't 3306 and `DB_HOST` isn't a socket |
    | `DB_NAME` | the database name from step 6, with any prefix |

19. **Add the environment secrets.** These are the credentials:

    | Secret | Value |
    |---|---|
    | `SSH_WEB_KEY` | the **entire** private key file `php2go_web`, including the `-----BEGIN` and `-----END` lines |
    | `SSH_APP_KEY` | the entire private key file `php2go_app` |
    | `SSH_MIGRATE_KEY` | the entire private key file `php2go_migrate` |
    | `DB_USER` | the database user from step 6 |
    | `DB_PASSWORD` | the database password from step 6 |
    | `ORGANIZER_PASSWORD` | a password to share with the organizers: a few random words is easy to share and hard to guess |
    | `SESSION_SECRET` | a random value, generated below |

    To copy a private key without opening it:
    ```
    Get-Content $HOME\.ssh\php2go_web -Raw | Set-Clipboard
    ```
    To generate `SESSION_SECRET` straight to the clipboard:
    ```
    $b = New-Object byte[] 32; [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b); [Convert]::ToBase64String($b) | Set-Clipboard
    ```
    On macOS or Linux, use `openssl rand -base64 32`.
20. 🔒 **Delete the private keys from your computer.** GitHub now holds the only copies. If one is lost, make a new key rather than recover it. Keep the `.pub` files, which are harmless and help identify the keys later.
    ```
    Remove-Item $HOME\.ssh\php2go_web, $HOME\.ssh\php2go_app, $HOME\.ssh\php2go_migrate
    ```

### F. First deploy: a dry run, then the real one

21. **Dry run:** add an empty file `.github/dry-run`, then commit and push to `main`. Check the Deploy run in GitHub Actions:
    - every setting was found;
    - the web and app keys passed the safety check;
    - the lists show what *would* be uploaded and deleted.

    Make sure nothing you want to keep in the web root is on the delete list.
22. **The real deploy:** delete `.github/dry-run`, then commit and push. The run uploads the code, creates the tables, switches the web root over, and checks `/health`.
23. **Check it:** `https://<your domain>/health` shows `"status":"ok"` and the release ID from the run. Sign in from the **Organizers** link in the footer, and create the first event.

## If something goes wrong

- **`rsync error … (code 23)` about times or permissions:** the host owns the folder itself and won't let it be changed. The deploy already leaves folder times and permissions alone; if this still appears, check the folder belongs to your account.
- **`Couldn't connect with the … key` or `Permission denied (publickey)`:** check the key's secret holds the whole private key, that its `authorized_keys` line passed the test in step 16, and that `SSH_HOST`, `SSH_USER`, `SSH_PORT` and `SSH_KNOWN_HOSTS` match what you verified in steps 4 and 9–11.
- **`The marker … isn't in this key's folder`:** the key's `rrsync` folder in `authorized_keys` doesn't match where you created the marker in step 8.
- **The site shows the host's error page after the switch-over:** the control panel may have written PHP settings into the web root's `.htaccess`, which the deploy replaces. Set the PHP version in the panel's per-domain settings instead.
- **A slow first connection:** some hosts pause new SSH connections that arrive in quick succession. The deploy opens one connection per key and reuses it, so only the first one waits.

## Changing the organizers' password

Change the `ORGANIZER_PASSWORD` secret and run the Deploy workflow again. Every existing session ends (D-023).

## Running it locally

You need PHP 8.1+ with `pdo_mysql` and `mbstring`, a MySQL 8 database and Node 22. Set the settings above as environment variables, then:

```sh
sh bin/serve-local.sh 8080                                   # applies migrations, then serves on port 8080
node --test --test-concurrency=1 'tests/contract/*.test.mjs' # the acceptance tests
cd tests/gate && npm install && npx playwright install && npx playwright test
```

Use `FORM_MIN_SECONDS=0` for the tests. For the password-change test, start a second copy on the same database with another `ORGANIZER_PASSWORD` on port 8081, and set `ALT_BASE_URL=http://127.0.0.1:8081`; without it, that test is skipped.
