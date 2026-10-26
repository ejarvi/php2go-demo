<?php

declare(strict_types=1);

// Every signed value uses the session secret, with a purpose so that one kind of
// value can never be passed off as another.
function sign(string $purpose, string $data): string
{
    return hash_hmac('sha256', $purpose . '|' . $data, config('SESSION_SECRET'));
}

function set_cookie(string $name, string $value, int $expires = 0): void
{
    setcookie($name, $value, [
        'expires' => $expires,
        'path' => '/',
        'secure' => true,
        'httponly' => true,
        'samesite' => 'Lax',
    ]);
}

function clear_cookie(string $name): void
{
    set_cookie($name, '', 1);
}

// CSRF protection (D-019). The browser holds a random key in a cookie, and each form
// carries a token signed from that key. Visitors have no session, so this needs none.
function csrf_token(): string
{
    static $token = null;
    if ($token === null) {
        $key = $_COOKIE['csrf'] ?? '';
        if (!is_string($key) || !preg_match('/^[0-9a-f]{32}$/', $key)) {
            $key = bin2hex(random_bytes(16));
            set_cookie('csrf', $key);
        }
        $token = sign('csrf', $key);
    }
    return $token;
}

// Forms accept only same-origin requests (D-025). Here that means a valid token, and
// no sign from the browser that the request came from another origin. A refusal gets
// 403 and a way back to the form.
function check_csrf(): void
{
    $key = $_COOKIE['csrf'] ?? '';
    $tokenOk = is_string($key) && $key !== '' && hash_equals(sign('csrf', $key), post('csrf'));
    if (!$tokenOk || !origin_is_this_site()) {
        error_page(
            403,
            'Please try again',
            'This form had expired, or it came from another site. Go back, reload the page and try again.',
            back_path(),
            'Go back to the form',
        );
    }
}

function origin_is_this_site(): bool
{
    // Browsers say where a request came from. "none" means typed in or bookmarked.
    $fetchSite = $_SERVER['HTTP_SEC_FETCH_SITE'] ?? null;
    if ($fetchSite !== null && $fetchSite !== 'same-origin' && $fetchSite !== 'none') {
        return false;
    }

    $origin = $_SERVER['HTTP_ORIGIN'] ?? null;
    if ($origin === null) {
        // Older browsers and other clients send no Origin; the token still applies.
        return true;
    }
    $parts = parse_url($origin);
    if (!is_array($parts) || !isset($parts['host'])) {
        // Includes "null", which browsers send when they hide the origin.
        return false;
    }
    $host = $parts['host'] . (isset($parts['port']) ? ':' . $parts['port'] : '');
    return strcasecmp($host, $_SERVER['HTTP_HOST'] ?? '') === 0;
}

// The form timer (D-011): when the form was shown, signed so it can't be faked.
function form_timer(): string
{
    $shown = (string) time();
    return $shown . '.' . sign('timer', $shown);
}

function form_filled_too_fast(string $timer): bool
{
    [$shown, $signature] = array_pad(explode('.', $timer, 2), 2, '');
    if (!ctype_digit($shown) || !hash_equals(sign('timer', $shown), $signature)) {
        return true;
    }
    return time() - (int) $shown < (int) config('FORM_MIN_SECONDS');
}

// After a sign-up, the event page shows a confirmation. A short-lived signed cookie
// carries which event and how many people, and nothing personal.
function set_signup_confirmation(int $eventId, int $people): void
{
    $value = $eventId . '.' . $people;
    set_cookie('signedup', $value . '.' . sign('signedup', $value), time() + 600);
}

function take_signup_confirmation(int $eventId): ?int
{
    $cookie = $_COOKIE['signedup'] ?? '';
    if (!is_string($cookie) || !preg_match('/^(\d+)\.(\d+)\.([0-9a-f]{64})$/', $cookie, $m)) {
        return null;
    }
    if (!hash_equals(sign('signedup', $m[1] . '.' . $m[2]), $m[3]) || (int) $m[1] !== $eventId) {
        return null;
    }
    clear_cookie('signedup');
    return (int) $m[2];
}
