<?php

declare(strict_types=1);

// The activity log: what changed, to which event, and when. Never who, and never
// a visitor's name or email address (design/data.md).

function record_activity(string $action, ?int $eventId, ?string $eventTitle): void
{
    query(
        'INSERT INTO activity (action, event_id, event_title, created_at) VALUES (?, ?, ?, ?)',
        [$action, $eventId, $eventTitle, sql_time(now())],
    );
}

function recent_activity(int $limit = 20): array
{
    $rows = query('SELECT * FROM activity ORDER BY created_at DESC, id DESC LIMIT ' . $limit)->fetchAll();
    return array_map(fn (array $row) => [
        'action' => $row['action'],
        'event_title' => $row['event_title'],
        'created_at' => new DateTimeImmutable($row['created_at']),
    ], $rows);
}
