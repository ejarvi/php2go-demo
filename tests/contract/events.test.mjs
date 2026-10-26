// design/modules/events.md, acceptance tests 1–6.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Client, organizer, createEvent, signUp, clubDate, uniqueTitle } from './lib.mjs';

test('1. / lists published future events soonest first, and not drafts', async () => {
  const admin = await organizer();
  const later = uniqueTitle('Later potluck');
  const sooner = uniqueTitle('Sooner clean-up');
  const draft = uniqueTitle('Draft workshop');
  await createEvent(admin, { title: later, date: clubDate(20) });
  await createEvent(admin, { title: sooner, date: clubDate(10) });
  await createEvent(admin, { title: draft, date: clubDate(15), published: false });

  const page = await new Client().get('/');
  assert.equal(page.status, 200);
  assert.ok(page.text.includes(sooner) && page.text.includes(later), 'both published events are listed');
  assert.ok(page.text.indexOf(sooner) < page.text.indexOf(later), 'the sooner event comes first');
  assert.ok(!page.text.includes(draft), 'the draft is not listed');
});

test('2. / does not list a published event that started yesterday', async () => {
  const admin = await organizer();
  const title = uniqueTitle('Yesterday picnic');
  await createEvent(admin, { title, date: clubDate(-1) });

  const page = await new Client().get('/');
  assert.ok(!page.text.includes(title));
});

test('3. a draft is 404 without a session, and 200 for an organizer', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Draft'), published: false });

  assert.equal((await new Client().get(`/events/${id}`)).status, 404);
  const page = await admin.get(`/events/${id}`);
  assert.equal(page.status, 200);
  assert.match(page.text, /Draft: not visible to the public/);
});

test('4. an unknown event id is 404', async () => {
  assert.equal((await new Client().get('/events/999999999')).status, 404);
});

test('5. a full event shows Full and no sign-up form', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Small workshop'), capacity: '2' });
  assert.equal((await signUp(new Client(), id, { people: '2' })).status, 303);

  const page = await new Client().get(`/events/${id}`);
  assert.match(page.text, /\bFull\b/);
  assert.ok(!page.text.includes(`action="/events/${id}/signup"`), 'no sign-up form');
});

test('6. a description containing <script> is shown as text, not run', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Script'), description: 'Hello <script>alert(1)</script>' });

  const page = await new Client().get(`/events/${id}`);
  assert.ok(!page.text.includes('<script>alert(1)</script>'));
  assert.ok(page.text.includes('&lt;script&gt;alert(1)&lt;/script&gt;'));
});
