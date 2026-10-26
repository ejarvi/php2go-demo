<?php

declare(strict_types=1);

// design/modules/events.md

function events_index(): never
{
    render('events_index', ['title' => 'Upcoming events', 'events' => upcoming_events()]);
}

function events_show(string $id): never
{
    $event = find_event((int) $id);
    if ($event === null || (!$event['published'] && !is_organizer())) {
        not_found();
    }
    render_event_page($event, signup_form_defaults());
}

// The event page, also shown again by a sign-up that needs fixing.
function render_event_page(array $event, array $form, array $errors = [], ?string $notice = null, int $status = 200): never
{
    render('event_show', [
        'title' => $event['title'],
        'event' => $event,
        'confirmed' => take_signup_confirmation($event['id']),
        'form' => $form,
        'errors' => $errors,
        'notice' => $notice,
    ], $status);
}
