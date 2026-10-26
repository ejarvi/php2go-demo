<?php

declare(strict_types=1);

// Events, as plain arrays: id, title, description, starts_at, ends_at (DateTimeImmutable
// or null), location, capacity (int or null), published (bool), people (signed up).

const EVENT_SELECT = 'SELECT e.*, (SELECT COALESCE(SUM(s.people), 0) FROM signups s WHERE s.event_id = e.id) AS people FROM events e';

function event_from_row(array $row): array
{
    return [
        'id' => (int) $row['id'],
        'title' => $row['title'],
        'description' => $row['description'],
        'starts_at' => new DateTimeImmutable($row['starts_at']),
        'ends_at' => $row['ends_at'] === null ? null : new DateTimeImmutable($row['ends_at']),
        'location' => $row['location'],
        'capacity' => $row['capacity'] === null ? null : (int) $row['capacity'],
        'published' => (bool) $row['published'],
        'people' => (int) $row['people'],
    ];
}

function find_event(int $id): ?array
{
    $row = query(EVENT_SELECT . ' WHERE e.id = ?', [$id])->fetch();
    return $row ? event_from_row($row) : null;
}

// Published events starting today or later, soonest first (design/modules/events.md).
function upcoming_events(): array
{
    $rows = query(EVENT_SELECT . ' WHERE e.published = 1 AND e.starts_at >= ? ORDER BY e.starts_at, e.id', [sql_time(today())]);
    return array_map('event_from_row', $rows->fetchAll());
}

// For organizers: upcoming events soonest first, then past ones most recent first.
function admin_events(): array
{
    $upcoming = query(EVENT_SELECT . ' WHERE e.starts_at >= ? ORDER BY e.starts_at, e.id', [sql_time(today())]);
    $past = query(EVENT_SELECT . ' WHERE e.starts_at < ? ORDER BY e.starts_at DESC, e.id DESC', [sql_time(today())]);
    return [
        'upcoming' => array_map('event_from_row', $upcoming->fetchAll()),
        'past' => array_map('event_from_row', $past->fetchAll()),
    ];
}

function places_left(array $event): ?int
{
    return $event['capacity'] === null ? null : max(0, $event['capacity'] - $event['people']);
}

function event_is_full(array $event): bool
{
    return places_left($event) === 0;
}

function event_has_started(array $event): bool
{
    return $event['starts_at'] <= now();
}

function event_has_ended(array $event): bool
{
    return ($event['ends_at'] ?? $event['starts_at']) < now();
}

// --- The event form ---

function event_form_defaults(): array
{
    return [
        'title' => '',
        'date' => '',
        'start_time' => '',
        'end_time' => '',
        'location' => '',
        'description' => '',
        'capacity' => '',
        'published' => false,
    ];
}

function event_form_from_event(array $event): array
{
    return [
        'title' => $event['title'],
        'date' => $event['starts_at']->format('Y-m-d'),
        'start_time' => $event['starts_at']->format('H:i'),
        'end_time' => $event['ends_at']?->format('H:i') ?? '',
        'location' => $event['location'],
        'description' => $event['description'],
        'capacity' => $event['capacity'] === null ? '' : (string) $event['capacity'],
        'published' => $event['published'],
    ];
}

function event_form_from_post(): array
{
    return [
        'title' => trim(post('title')),
        'date' => trim(post('date')),
        'start_time' => trim(post('start_time')),
        'end_time' => trim(post('end_time')),
        'location' => trim(post('location')),
        'description' => trim(str_replace("\r\n", "\n", post('description'))),
        'capacity' => trim(post('capacity')),
        'published' => post('published') === '1',
    ];
}

// Checks the form. Returns the values to save and a message for each field that needs fixing.
function validate_event_form(array $form): array
{
    $errors = [];

    if ($form['title'] === '') {
        $errors['title'] = 'Enter a title.';
    } elseif (mb_strlen($form['title']) > 120) {
        $errors['title'] = 'Keep the title to 120 characters or fewer.';
    }

    $startsAt = parse_local_time($form['date'], $form['start_time']);
    if ($form['date'] === '' || parse_local_time($form['date'], '00:00') === null) {
        $errors['date'] = 'Enter a date.';
    } elseif ($startsAt === null && preg_match('/^\d{2}:\d{2}$/', $form['start_time'])) {
        $errors['start_time'] = "That time doesn't exist on this date, because the clocks change. Choose another time.";
    } elseif ($startsAt === null) {
        $errors['start_time'] = 'Enter a start time.';
    }

    $endsAt = null;
    if ($form['end_time'] !== '') {
        $endsAt = parse_local_time($form['date'], $form['end_time']);
        if ($endsAt === null) {
            $errors['end_time'] = 'Enter the end time as hours and minutes, or leave it empty.';
        } elseif ($startsAt !== null && $endsAt <= $startsAt) {
            $errors['end_time'] = 'The end time must be after the start time.';
        }
    }

    if ($form['location'] === '') {
        $errors['location'] = 'Enter where the event is.';
    } elseif (mb_strlen($form['location']) > 200) {
        $errors['location'] = 'Keep the location to 200 characters or fewer.';
    }

    if (mb_strlen($form['description']) > 5000) {
        $errors['description'] = 'Keep the description to 5,000 characters or fewer.';
    }

    $capacity = null;
    if ($form['capacity'] !== '') {
        if (!ctype_digit($form['capacity']) || (int) $form['capacity'] < 1 || (int) $form['capacity'] > 500) {
            $errors['capacity'] = 'Enter a number from 1 to 500, or leave it empty for no limit.';
        } else {
            $capacity = (int) $form['capacity'];
        }
    }

    $values = [
        'title' => $form['title'],
        'description' => $form['description'],
        'starts_at' => $startsAt,
        'ends_at' => $endsAt,
        'location' => $form['location'],
        'capacity' => $capacity,
        'published' => $form['published'],
    ];
    return [$values, $errors];
}

function parse_local_time(string $date, string $time): ?DateTimeImmutable
{
    $parsed = DateTimeImmutable::createFromFormat('!Y-m-d H:i', "$date $time");
    // Reject dates that PHP would quietly roll over, such as February 30.
    return $parsed !== false && $parsed->format('Y-m-d H:i') === "$date $time" ? $parsed : null;
}

function create_event(array $values): int
{
    query(
        'INSERT INTO events (title, description, starts_at, ends_at, location, capacity, published, updated_at)
         VALUES (?, ?, ?, ?, ?, ?, ?, ?)',
        event_params($values),
    );
    $id = (int) db()->lastInsertId();
    record_activity('Event created', $id, $values['title']);
    if ($values['published']) {
        record_activity('Event published', $id, $values['title']);
    }
    return $id;
}

function update_event(array $event, array $values): void
{
    query(
        'UPDATE events SET title = ?, description = ?, starts_at = ?, ends_at = ?, location = ?, capacity = ?,
         published = ?, updated_at = ? WHERE id = ?',
        [...event_params($values), $event['id']],
    );
    record_activity('Event updated', $event['id'], $values['title']);
    if ($values['published'] !== $event['published']) {
        record_activity($values['published'] ? 'Event published' : 'Event unpublished', $event['id'], $values['title']);
    }
}

function event_params(array $values): array
{
    return [
        $values['title'],
        $values['description'],
        sql_time($values['starts_at']),
        $values['ends_at'] === null ? null : sql_time($values['ends_at']),
        $values['location'],
        $values['capacity'],
        $values['published'] ? 1 : 0,
        sql_time(now()),
    ];
}

// Removing an event also removes its sign-ups (design/data.md), through the foreign key.
function delete_event(array $event): void
{
    query('DELETE FROM events WHERE id = ?', [$event['id']]);
    record_activity('Event deleted', $event['id'], $event['title']);
}
