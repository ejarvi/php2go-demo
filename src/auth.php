<?php

declare(strict_types=1);

// Organizers share one password, set as a deploy secret (D-023). A session is a
// random code in a cookie; only its hash is stored (design/data.md).

const SESSION_HOURS = 8;

// Ties a session to the password it was started with, so a new password ends every session.
function password_fingerprint(): string
{
    return sign('password', config('ORGANIZER_PASSWORD'));
}

function password_is_right(string $typed): bool
{
    // Comparing hashes takes the same time whatever is typed, including its length.
    return hash_equals(hash('sha256', config('ORGANIZER_PASSWORD')), hash('sha256', $typed));
}

function start_session(): void
{
    $code = bin2hex(random_bytes(32));
    $expires = now()->modify('+' . SESSION_HOURS . ' hours');
    query(
        'INSERT INTO sessions (code_hash, expires_at, password_fingerprint) VALUES (?, ?, ?)',
        [sign('session', $code), sql_time($expires), password_fingerprint()],
    );
    set_cookie('session', $code, $expires->getTimestamp());
}

function is_organizer(): bool
{
    static $result = null;
    if ($result === null) {
        $result = false;
        $code = $_COOKIE['session'] ?? '';
        if (is_string($code) && preg_match('/^[0-9a-f]{64}$/', $code)) {
            $result = (bool) query(
                'SELECT 1 FROM sessions WHERE code_hash = ? AND expires_at > ? AND password_fingerprint = ?',
                [sign('session', $code), sql_time(now()), password_fingerprint()],
            )->fetchColumn();
        }
    }
    return $result;
}

function end_session(): void
{
    $code = $_COOKIE['session'] ?? '';
    if (is_string($code) && $code !== '') {
        query('DELETE FROM sessions WHERE code_hash = ?', [sign('session', $code)]);
    }
    clear_cookie('session');
}
