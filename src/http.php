<?php

declare(strict_types=1);

function request_method(): string
{
    $method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
    return $method === 'HEAD' ? 'GET' : $method;
}

function request_path(): string
{
    return parse_url($_SERVER['REQUEST_URI'] ?? '/', PHP_URL_PATH) ?: '/';
}

// A form field as a string. Anything else (missing, or an array) counts as empty.
function post(string $key): string
{
    $value = $_POST[$key] ?? '';
    return is_string($value) ? $value : '';
}

function query_param(string $key): string
{
    $value = $_GET[$key] ?? '';
    return is_string($value) ? $value : '';
}

// The headers every response carries (D-019, design/modules/site.md).
function send_security_headers(): void
{
    header("Content-Security-Policy: default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self'; "
        . "form-action 'self'; frame-ancestors 'none'; base-uri 'none'; object-src 'none'");
    header('Referrer-Policy: same-origin');
    header('X-Content-Type-Options: nosniff');
    header('Strict-Transport-Security: max-age=31536000');
}

function e(?string $text): string
{
    return htmlspecialchars($text ?? '', ENT_QUOTES | ENT_SUBSTITUTE, 'UTF-8');
}

function redirect(string $to): never
{
    header('Location: ' . $to, true, 303);
    exit;
}

function render(string $template, array $vars = [], int $status = 200): never
{
    http_response_code($status);
    $content = render_partial($template, $vars);
    echo render_partial('layout', [
        'title' => $vars['title'] ?? config('CLUB_NAME'),
        'content' => $content,
    ]);
    exit;
}

function render_partial(string $template, array $vars = []): string
{
    extract($vars, EXTR_SKIP);
    ob_start();
    try {
        require dirname(__DIR__) . "/templates/$template.php";
    } catch (Throwable $e) {
        ob_end_clean();
        throw $e;
    }
    return (string) ob_get_clean();
}

function json_response(array $body, int $status = 200): never
{
    http_response_code($status);
    header('Content-Type: application/json');
    header('Cache-Control: no-store');
    echo json_encode($body, JSON_UNESCAPED_SLASHES);
    exit;
}

function error_page(int $status, string $title, string $message, string $linkHref, string $linkText): never
{
    render('error', [
        'title' => $title,
        'message' => $message,
        'linkHref' => $linkHref,
        'linkText' => $linkText,
    ], $status);
}

function not_found(): never
{
    error_page(404, 'Page not found', "We couldn't find that page.", '/', 'See upcoming events');
}

// Where to send someone back to: the page they came from, if it was on this site.
function back_path(): string
{
    $referer = $_SERVER['HTTP_REFERER'] ?? '';
    $parts = parse_url($referer);
    if (!is_array($parts) || !isset($parts['host'], $parts['path'])) {
        return '/';
    }
    $host = $parts['host'] . (isset($parts['port']) ? ':' . $parts['port'] : '');
    return strcasecmp($host, $_SERVER['HTTP_HOST'] ?? '') === 0 ? $parts['path'] : '/';
}
