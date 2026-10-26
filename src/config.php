<?php

declare(strict_types=1);

// Settings come from environment variables when testing locally and in CI. On the
// host they come from config.local.json, which the deploy writes from the GitHub
// environment's secrets and variables (D-015, D-024). It's never in the web root:
// locally it sits beside src/, and on the host it's in the app folder, above releases/.

const CONFIG_DEFAULTS = [
    'DB_HOST' => '127.0.0.1',
    'DB_PORT' => '3306',
    'DB_PASSWORD' => '',
    // The shortest time a person could fill in a public form (D-011).
    'FORM_MIN_SECONDS' => '3',
];

function config(string $key): string
{
    static $file = null;
    if ($file === null) {
        $file = [];
        foreach ([dirname(__DIR__), dirname(__DIR__, 3)] as $dir) {
            if (is_file("$dir/config.local.json")) {
                $file = json_decode((string) file_get_contents("$dir/config.local.json"), true, 512, JSON_THROW_ON_ERROR);
                break;
            }
        }
    }

    $value = getenv($key);
    if ($value !== false && $value !== '') {
        return $value;
    }
    if (isset($file[$key]) && $file[$key] !== '') {
        return (string) $file[$key];
    }
    if ($key === 'RELEASE') {
        // The deploy writes the release ID into each release's folder (D-016).
        $path = dirname(__DIR__) . '/RELEASE';
        return is_file($path) ? trim((string) file_get_contents($path)) : 'dev';
    }
    if (array_key_exists($key, CONFIG_DEFAULTS)) {
        return CONFIG_DEFAULTS[$key];
    }
    throw new RuntimeException("The setting $key is missing.");
}
