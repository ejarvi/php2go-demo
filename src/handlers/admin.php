<?php

declare(strict_types=1);

// design/modules/admin.md. The front controller has already checked for a session.

function admin_index(): never
{
    $events = admin_events();
    render('admin_index', [
        'title' => 'Admin',
        'upcoming' => $events['upcoming'],
        'past' => $events['past'],
        'activity' => recent_activity(),
    ]);
}

function admin_event_new(): never
{
    render_event_form(null, event_form_defaults());
}

function admin_event_create(): never
{
    $form = event_form_from_post();
    [$values, $errors] = validate_event_form($form);
    if ($errors) {
        render_event_form(null, $form, $errors, 422);
    }
    $id = create_event($values);
    redirect("/admin/events/$id?saved=1");
}

function admin_event_edit(string $id): never
{
    $event = find_event((int) $id) ?? not_found();
    render_event_form($event, event_form_from_event($event));
}

function admin_event_update(string $id): never
{
    $event = find_event((int) $id) ?? not_found();
    $form = event_form_from_post();
    [$values, $errors] = validate_event_form($form);
    if ($errors) {
        render_event_form($event, $form, $errors, 422);
    }
    update_event($event, $values);
    redirect("/admin/events/{$event['id']}?saved=1");
}

function render_event_form(?array $event, array $form, array $errors = [], int $status = 200): never
{
    render('admin_event_form', [
        'title' => $event === null ? 'New event' : 'Edit event',
        'event' => $event,
        'form' => $form,
        'errors' => $errors,
        'saved' => $event !== null && query_param('saved') === '1' && $errors === [],
        'signups' => $event === null ? [] : event_signups($event['id']),
    ], $status);
}

// The first POST shows a confirmation page; its button posts again with confirm=yes.
function admin_event_delete(string $id): never
{
    $event = find_event((int) $id) ?? not_found();
    if (post('confirm') !== 'yes') {
        render('admin_event_delete', ['title' => 'Delete event', 'event' => $event]);
    }
    delete_event($event);
    redirect('/admin');
}

function admin_signups_csv(string $id): never
{
    $event = find_event((int) $id) ?? not_found();
    header('Content-Type: text/csv; charset=utf-8');
    header('Content-Disposition: attachment; filename="signups-' . $event['id'] . '.csv"');
    header('Cache-Control: no-store');
    $out = fopen('php://output', 'w');
    fputcsv($out, ['Name', 'Email', 'People', 'Signed up'], ',', '"', '');
    foreach (event_signups($event['id']) as $signup) {
        fputcsv($out, [
            csv_safe($signup['name']),
            csv_safe($signup['email']),
            $signup['people'],
            $signup['created_at']->format('Y-m-d H:i'),
        ], ',', '"', '');
    }
    fclose($out);
    exit;
}

// Stops a spreadsheet from running a name like "=HYPERLINK(...)" as a formula.
function csv_safe(string $value): string
{
    return preg_match('/^[=+\-@\t\r]/', $value) ? "'" . $value : $value;
}

function admin_signup_delete(string $id): never
{
    $signup = find_signup((int) $id) ?? not_found();
    $event = find_event($signup['event_id']);
    delete_signup($signup['id']);
    record_activity('Sign-up removed', $signup['event_id'], $event['title'] ?? null);
    redirect("/admin/events/{$signup['event_id']}");
}
