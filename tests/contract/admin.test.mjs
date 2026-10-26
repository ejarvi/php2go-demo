// design/modules/admin.md, acceptance tests 1–7.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Client, organizer, createEvent, signUp, formFields, signIn, clubDate, uniqueTitle } from './lib.mjs';

test('1. /admin pages redirect to /signin without a session, and /admin changes are refused', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Guarded') });
  const visitor = new Client();

  for (const path of ['/admin', '/admin/events/new', `/admin/events/${id}`, `/admin/events/${id}/signups.csv`, '/admin/anything']) {
    const response = await visitor.get(path);
    assert.equal(response.status, 303, path);
    assert.match(response.location, /^\/signin/, path);
  }

  // A real token from a public form, so the refusal is about the missing session.
  const { csrf } = formFields((await visitor.get('/signin')).text, '/signin');
  for (const path of ['/admin/events', `/admin/events/${id}`, `/admin/events/${id}/delete`, '/admin/signups/1/delete']) {
    const response = await visitor.post(path, { csrf, title: 'Changed', confirm: 'yes' });
    assert.equal(response.status, 403, path);
  }
  assert.equal((await new Client().get(`/events/${id}`)).status, 200, 'the event is still there');
});

test('2. a published event appears on /; unticking Published removes it', async () => {
  const admin = await organizer();
  const title = uniqueTitle('Toggle');
  const id = await createEvent(admin, { title });
  assert.ok((await new Client().get('/')).text.includes(title));

  const edit = await admin.get(`/admin/events/${id}`);
  const fields = formFields(edit.text, `/admin/events/${id}`);
  delete fields.published;
  assert.equal((await admin.post(`/admin/events/${id}`, fields)).status, 303);
  assert.ok(!(await new Client().get('/')).text.includes(title));
});

test('3. an end time before the start time is refused, and the form keeps what was typed', async () => {
  const admin = await organizer();
  const title = uniqueTitle('Backwards');
  const page = await admin.get('/admin/events/new');
  const response = await admin.post('/admin/events', {
    ...formFields(page.text, '/admin/events'),
    title, date: clubDate(5), start_time: '18:00', end_time: '17:00', location: 'Hall', published: '1',
  });
  assert.equal(response.status, 422);
  assert.ok(response.text.includes(`value="${title}"`), 'the title is kept');
  assert.ok(!(await new Client().get('/')).text.includes(title), 'nothing was created');
});

test('4. deleting an event removes it and its sign-ups; its page is then 404', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Doomed') });
  await signUp(new Client(), id, { email: 'doomed@example.com' });

  // The Delete button leads to a confirmation page, whose button deletes.
  const edit = await admin.get(`/admin/events/${id}`);
  const removeAction = edit.text.match(/action="(\/admin\/signups\/\d+\/delete)"/)[1];
  const confirm = await admin.post(`/admin/events/${id}/delete`, formFields(edit.text, `/admin/events/${id}/delete`));
  assert.equal(confirm.status, 200);
  const deleted = await admin.post(`/admin/events/${id}/delete`, formFields(confirm.text, `/admin/events/${id}/delete`));
  assert.equal(deleted.status, 303);

  assert.equal((await new Client().get(`/events/${id}`)).status, 404);
  assert.equal((await admin.get(`/admin/events/${id}`)).status, 404);
  // The sign-up itself is gone, not just its event: removing it now finds nothing.
  const fields = formFields((await admin.get('/admin')).text, '/signout');
  assert.equal((await admin.post(removeAction, { csrf: fields.csrf })).status, 404, 'its sign-ups are gone');
});

test('5. the CSV has a header row and one row per sign-up, served as an attachment', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Export') });
  await signUp(new Client(), id, { name: 'Ana', email: 'ana@example.com', people: '2' });
  await signUp(new Client(), id, { name: 'Ben', email: 'ben@example.com', people: '1' });

  const response = await admin.get(`/admin/events/${id}/signups.csv`);
  assert.equal(response.status, 200);
  assert.match(response.headers.get('content-type'), /^text\/csv/);
  assert.match(response.headers.get('content-disposition'), /^attachment/);
  const lines = response.text.trim().split(/\r?\n/);
  assert.equal(lines.length, 3);
  assert.match(lines[0], /name/i);
  assert.ok(lines.some((line) => line.includes('ana@example.com')));
  assert.ok(lines.some((line) => line.includes('ben@example.com')));
});

test('6. an organizer can sign in with the configured password and create an event', async () => {
  // Run against a fresh site in CI, where this is the first event; elsewhere it still checks the path.
  const client = new Client();
  assert.equal((await signIn(client)).status, 303);
  const title = uniqueTitle('First event');
  const id = await createEvent(client, { title });
  const page = await new Client().get(`/events/${id}`);
  assert.equal(page.status, 200);
  assert.ok(page.text.includes(title));
});

test('7. each change adds an activity entry, with no visitor names or emails', async () => {
  const admin = await organizer();
  const title = uniqueTitle('Logged');
  const id = await createEvent(admin, { title });
  await signUp(new Client(), id, { name: 'Zed Private', email: 'zed.private@example.com' });

  const edit = await admin.get(`/admin/events/${id}`);
  const removeAction = edit.text.match(/action="(\/admin\/signups\/\d+\/delete)"/)[1];
  assert.equal((await admin.post(removeAction, formFields(edit.text, removeAction))).status, 303);

  const page = await admin.get('/admin');
  assert.match(page.text, new RegExp(`Event created: ${title}`, 'i'));
  assert.match(page.text, new RegExp(`Sign-up removed: ${title}`, 'i'));
  assert.ok(!page.text.includes('Zed Private') && !page.text.includes('zed.private@example.com'));
});
