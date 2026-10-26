<?php

declare(strict_types=1);

function signup_form_defaults(): array
{
    return ['name' => '', 'email' => '', 'people' => '1'];
}

function signup_form_from_post(): array
{
    return [
        'name' => trim(post('name')),
        'email' => trim(post('email')),
        'people' => trim(post('people')),
    ];
}

function validate_signup_form(array $form): array
{
    $errors = [];
    if ($form['name'] === '') {
        $errors['name'] = 'Enter your name.';
    } elseif (mb_strlen($form['name']) > 100) {
        $errors['name'] = 'Keep your name to 100 characters or fewer.';
    }
    if ($form['email'] === '') {
        $errors['email'] = 'Enter your email address.';
    } elseif (mb_strlen($form['email']) > 254 || filter_var($form['email'], FILTER_VALIDATE_EMAIL) === false) {
        $errors['email'] = 'Enter an email address like name@example.com.';
    }
    if (!ctype_digit($form['people']) || (int) $form['people'] < 1 || (int) $form['people'] > 6) {
        $errors['people'] = 'Enter a number from 1 to 6.';
    }
    return $errors;
}

// Saves a sign-up, or replaces the head count of an earlier one with the same email.
// Returns null when saved, or the message to show when there aren't enough places (D-010).
function save_signup(int $eventId, array $form): ?string
{
    $pdo = db();
    $people = (int) $form['people'];
    $pdo->beginTransaction();
    try {
        // Locking the event row makes sign-ups to one event take turns, so two at once can't overfill it.
        $capacity = query('SELECT capacity FROM events WHERE id = ? FOR UPDATE', [$eventId])->fetchColumn();
        $earlier = query('SELECT id, people FROM signups WHERE event_id = ? AND email = ?', [$eventId, $form['email']])->fetch();

        if ($capacity !== null) {
            $taken = (int) query('SELECT COALESCE(SUM(people), 0) FROM signups WHERE event_id = ?', [$eventId])->fetchColumn();
            $left = max(0, (int) $capacity - $taken + ($earlier ? (int) $earlier['people'] : 0));
            if ($people > $left) {
                $pdo->rollBack();
                return $left === 0
                    ? 'Sorry, this event is now full.'
                    : sprintf('Sorry, only %s left.', $left === 1 ? '1 place is' : "$left places are");
            }
        }

        if ($earlier) {
            query('UPDATE signups SET name = ?, people = ? WHERE id = ?', [$form['name'], $people, $earlier['id']]);
        } else {
            query(
                'INSERT INTO signups (event_id, name, email, people, created_at) VALUES (?, ?, ?, ?, ?)',
                [$eventId, $form['name'], $form['email'], $people, sql_time(now())],
            );
        }
        $pdo->commit();
        return null;
    } catch (Throwable $e) {
        if ($pdo->inTransaction()) {
            $pdo->rollBack();
        }
        throw $e;
    }
}

// Earliest first, for the organizers' table and the CSV.
function event_signups(int $eventId): array
{
    $rows = query('SELECT * FROM signups WHERE event_id = ? ORDER BY created_at, id', [$eventId])->fetchAll();
    return array_map(fn (array $row) => [
        'id' => (int) $row['id'],
        'name' => $row['name'],
        'email' => $row['email'],
        'people' => (int) $row['people'],
        'created_at' => new DateTimeImmutable($row['created_at']),
    ], $rows);
}

function find_signup(int $id): ?array
{
    $row = query('SELECT id, event_id FROM signups WHERE id = ?', [$id])->fetch();
    return $row ? ['id' => (int) $row['id'], 'event_id' => (int) $row['event_id']] : null;
}

function delete_signup(int $id): void
{
    query('DELETE FROM signups WHERE id = ?', [$id]);
}
