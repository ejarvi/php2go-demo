<?php

declare(strict_types=1);

// Loads everything a request or a command-line script needs.

require __DIR__ . '/config.php';
require __DIR__ . '/db.php';
require __DIR__ . '/http.php';
require __DIR__ . '/security.php';
require __DIR__ . '/auth.php';
require __DIR__ . '/format.php';
require __DIR__ . '/events.php';
require __DIR__ . '/signups.php';
require __DIR__ . '/activity.php';
require __DIR__ . '/housekeeping.php';
require __DIR__ . '/handlers/events.php';
require __DIR__ . '/handlers/signups.php';
require __DIR__ . '/handlers/signin.php';
require __DIR__ . '/handlers/admin.php';
require __DIR__ . '/handlers/site.php';

// Every date and time is in the club's time zone (D-006).
date_default_timezone_set(config('CLUB_TIMEZONE'));

// Turn warnings into exceptions, so a mistake fails loudly instead of rendering half a page.
// Deprecation notices are left to the log, so a host's PHP upgrade can't take the site down.
set_error_handler(function (int $severity, string $message, string $file, int $line): bool {
    if (!(error_reporting() & $severity) || ($severity & (E_DEPRECATED | E_USER_DEPRECATED))) {
        return false;
    }
    throw new ErrorException($message, 0, $severity, $file, $line);
});
