#!/bin/sh
# Starts the site on PHP's built-in web server, for the acceptance tests and the gate.
# Settings come from environment variables (see platform/README.md).
#
#   sh bin/serve-local.sh 8080
set -eu
cd "$(dirname "$0")/.."
php bin/migrate.php
exec php -S "127.0.0.1:${1:-8080}" -t public public/index.php
