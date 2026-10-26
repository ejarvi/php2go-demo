<?php

declare(strict_types=1);

// design/modules/signin.md

function signin_form(): never
{
    if (is_organizer()) {
        redirect('/admin');
    }
    render('signin', ['title' => 'Organizers', 'next' => safe_next(query_param('next')), 'error' => null]);
}

function signin_submit(): never
{
    $next = safe_next(post('next'));
    if (!password_is_right(post('password'))) {
        // Answer slowly, to slow down guessing.
        sleep(1);
        render('signin', ['title' => 'Organizers', 'next' => $next, 'error' => "That password isn't right."], 422);
    }
    start_session();
    redirect($next);
}

function signout(): never
{
    end_session();
    redirect('/');
}

// Only ever send an organizer on to a page in the organizers' area.
function safe_next(string $next): string
{
    return preg_match('#^/admin(/[A-Za-z0-9/._-]*)?$#', $next) ? $next : '/admin';
}
