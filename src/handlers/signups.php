<?php

declare(strict_types=1);

// design/modules/signups.md

function signups_create(string $id): never
{
    // 1. The event exists, is published, and hasn't started.
    $event = find_event((int) $id);
    if ($event === null || !$event['published']) {
        not_found();
    }
    $form = signup_form_from_post();
    if (event_has_started($event)) {
        render_event_page($event, $form, [], 'Sign-ups for this event have closed.', 409);
    }

    // 2. The fields are valid.
    $errors = validate_signup_form($form);
    if ($errors) {
        render_event_page($event, $form, $errors, null, 422);
    }

    // 3. The trap field is empty, and the form wasn't filled in too fast (D-011).
    // A bot gets the normal success page, but nothing is saved.
    if (post('website') !== '' || form_filled_too_fast(post('timer'))) {
        set_signup_confirmation($event['id'], (int) $form['people']);
        redirect('/events/' . $event['id']);
    }

    // 4. The head count fits in the places left (D-010).
    $problem = save_signup($event['id'], $form);
    if ($problem !== null) {
        render_event_page(find_event($event['id']) ?? $event, $form, [], $problem, 409);
    }

    set_signup_confirmation($event['id'], (int) $form['people']);
    redirect('/events/' . $event['id']);
}
