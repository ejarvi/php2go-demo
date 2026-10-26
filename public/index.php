<?php

declare(strict_types=1);

// The front controller: Apache sends every request that isn't a file here (.htaccess).

// Under PHP's built-in server (local runs and CI), serve files such as the stylesheet directly.
if (PHP_SAPI === 'cli-server' && is_file(__DIR__ . parse_url($_SERVER['REQUEST_URI'], PHP_URL_PATH))) {
    return false;
}

// On the host, the code lives outside the web root, and the deploy writes code-path.php
// to say where this release's copy is. Locally and in CI, it's next to public/.
$code = is_file(__DIR__ . '/code-path.php') ? require __DIR__ . '/code-path.php' : dirname(__DIR__);
require $code . '/src/bootstrap.php';

send_security_headers();

set_exception_handler(function (Throwable $e): void {
    // Details go to the log, never to the page (design/modules/site.md).
    error_log((string) $e);
    while (ob_get_level() > 0) {
        ob_end_clean();
    }
    if (!headers_sent()) {
        error_page(500, 'Something went wrong', 'Something went wrong. Please try again.', '/', 'Go to the home page');
    }
});

// Every page and form, from design/pages.md.
const ROUTES = [
    ['GET', '#^/$#', 'events_index'],
    ['GET', '#^/events/(\d+)$#', 'events_show'],
    ['POST', '#^/events/(\d+)/signup$#', 'signups_create'],
    ['GET', '#^/signin$#', 'signin_form'],
    ['POST', '#^/signin$#', 'signin_submit'],
    ['POST', '#^/signout$#', 'signout'],
    ['GET', '#^/admin$#', 'admin_index'],
    ['GET', '#^/admin/events/new$#', 'admin_event_new'],
    ['POST', '#^/admin/events$#', 'admin_event_create'],
    ['GET', '#^/admin/events/(\d+)$#', 'admin_event_edit'],
    ['POST', '#^/admin/events/(\d+)$#', 'admin_event_update'],
    ['POST', '#^/admin/events/(\d+)/delete$#', 'admin_event_delete'],
    ['GET', '#^/admin/events/(\d+)/signups\.csv$#', 'admin_signups_csv'],
    ['POST', '#^/admin/signups/(\d+)/delete$#', 'admin_signup_delete'],
    ['GET', '#^/health$#', 'site_health'],
];

$method = request_method();
$path = request_path();

// Any /admin path needs a session: pages redirect to sign in, and changes are refused.
if (($path === '/admin' || str_starts_with($path, '/admin/')) && !is_organizer()) {
    if ($method === 'GET') {
        redirect('/signin?next=' . rawurlencode($path));
    }
    error_page(403, 'Please sign in', 'Your session has ended. Sign in again, then repeat the change.', '/signin', 'Sign in');
}

if ($method === 'POST') {
    check_csrf();
}

// The health check stays clear of this, so it reports storage problems itself.
if ($path === '/admin' || ($path !== '/health' && random_int(1, 50) === 1)) {
    housekeeping();
}

foreach (ROUTES as [$routeMethod, $pattern, $handler]) {
    if ($routeMethod === $method && preg_match($pattern, $path, $matches)) {
        $handler(...array_slice($matches, 1));
    }
}

not_found();
