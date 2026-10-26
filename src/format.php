<?php

declare(strict_types=1);

// "Sat, Nov 8", with the year added when it isn't this year.
function event_day(array $event): string
{
    $start = $event['starts_at'];
    return $start->format($start->format('Y') === now()->format('Y') ? 'D, M j' : 'D, M j, Y');
}

// "9:00 AM" or "9:00 AM – 12:00 PM".
function event_times(array $event): string
{
    $times = $event['starts_at']->format('g:i A');
    if ($event['ends_at'] !== null) {
        $times .= ' – ' . $event['ends_at']->format('g:i A');
    }
    return $times;
}

function short_time(DateTimeImmutable $time): string
{
    return $time->format('M j, g:i A');
}

function people_label(int $count): string
{
    return $count === 1 ? '1 person' : "$count people";
}

function places_label(int $count): string
{
    return $count === 1 ? '1 place left' : "$count places left";
}

// The description is plain text: line breaks are kept and web addresses become links.
function description_html(string $text): string
{
    $html = preg_replace_callback('~https?://[^\s<]+~', function (array $m): string {
        $url = rtrim($m[0], '.,;:!?)');
        return '<a href="' . $url . '" rel="nofollow noopener">' . $url . '</a>' . substr($m[0], strlen($url));
    }, e($text));
    return nl2br($html, false);
}

// The attributes that tie a form field to its error message, for screen readers.
function field_error_attrs(array $errors, string $field): string
{
    return isset($errors[$field]) ? ' aria-invalid="true" aria-describedby="' . $field . '-error"' : '';
}

function field_error(array $errors, string $field): string
{
    return isset($errors[$field])
        ? '<p class="field-error" id="' . $field . '-error">' . e($errors[$field]) . '</p>'
        : '';
}
