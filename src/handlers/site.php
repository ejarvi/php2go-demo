<?php

declare(strict_types=1);

// design/modules/site.md

function site_health(): never
{
    try {
        query('SELECT id FROM events LIMIT 1')->fetchAll();
        json_response(['status' => 'ok', 'release' => config('RELEASE')]);
    } catch (Throwable $e) {
        error_log('Health check failed: ' . $e->getMessage());
        json_response(['status' => 'error', 'release' => config('RELEASE')], 503);
    }
}
