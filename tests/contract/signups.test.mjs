// design/modules/signups.md, acceptance tests 1–6.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Client, organizer, createEvent, signUp, formFields, clubDate, uniqueTitle, crossSiteHeaders } from './lib.mjs';

// The organizers' page for an event lists one row per sign-up, with its email address.
async function signupRows(admin, id, email) {
  const page = await admin.get(`/admin/events/${id}`);
  return page.text.split(email).length - 1;
}

test('1. a valid sign-up redirects to the event page, which confirms it; places left drop', async () => {
  const admin = await organizer();
  const title = uniqueTitle('Potluck');
  const id = await createEvent(admin, { title, capacity: '10' });
  const visitor = new Client();

  const response = await signUp(visitor, id, { people: '3' });
  assert.equal(response.status, 303);
  assert.equal(response.location, `/events/${id}`);

  const page = await visitor.get(`/events/${id}`);
  assert.ok(page.text.includes(`signed up for ${title}, 3 people`), 'shows the confirmation');
  assert.match(page.text, /7 places left/);
});

test('2. a head count larger than the places left is refused, and nothing is saved', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Tiny tour'), capacity: '2' });

  const response = await signUp(new Client(), id, { people: '3', email: 'too-many@example.com' });
  assert.ok(response.status >= 400, `refused with ${response.status}`);
  assert.match(response.text, /only 2 places are left/);
  assert.equal(await signupRows(admin, id, 'too-many@example.com'), 0);
});

test('3. signing up twice with the same email leaves one sign-up, with the later head count', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Clean-up'), capacity: '20' });

  await signUp(new Client(), id, { email: 'twice@example.com', people: '2' });
  await signUp(new Client(), id, { email: 'twice@example.com', people: '5' });

  assert.equal(await signupRows(admin, id, 'twice@example.com'), 1);
  const page = await new Client().get(`/events/${id}`);
  assert.match(page.text, /15 places left/);
});

test('4. a sign-up with the trap field filled in looks successful but saves nothing', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Bake sale') });

  const response = await signUp(new Client(), id, { email: 'bot@example.com', website: 'http://spam.example' });
  assert.equal(response.status, 303);
  assert.equal(response.location, `/events/${id}`);
  assert.equal(await signupRows(admin, id, 'bot@example.com'), 0);
});

test('5. a sign-up to a draft, a past event or an unknown id is refused', async () => {
  const admin = await organizer();
  const visitor = new Client();
  const openId = await createEvent(admin, { title: uniqueTitle('Open') });
  // A real form from an open event, so only the target differs.
  const fields = {
    ...formFields((await visitor.get(`/events/${openId}`)).text, `/events/${openId}/signup`),
    name: 'Sam', email: 'refused@example.com', people: '1',
  };

  const draftId = await createEvent(admin, { title: uniqueTitle('Draft'), published: false });
  assert.equal((await visitor.post(`/events/${draftId}/signup`, fields)).status, 404);
  assert.equal((await visitor.post('/events/999999999/signup', fields)).status, 404);

  const pastId = await createEvent(admin, { title: uniqueTitle('Past'), date: clubDate(-2) });
  const past = await visitor.post(`/events/${pastId}/signup`, fields);
  assert.ok(past.status >= 400, `refused with ${past.status}`);
  assert.equal(await signupRows(admin, pastId, 'refused@example.com'), 0);
});

test('6. a sign-up sent from another site is refused with 403, and nothing is saved', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Workshop') });
  const visitor = new Client();
  // Everything else is a genuine form, so only where it came from is wrong.
  const action = `/events/${id}/signup`;
  const fields = { ...formFields((await visitor.get(`/events/${id}`)).text, action), name: 'Mal', email: 'cross-site@example.com', people: '1' };

  const response = await visitor.post(action, fields, { headers: crossSiteHeaders() });
  assert.equal(response.status, 403);
  assert.equal(await signupRows(admin, id, 'cross-site@example.com'), 0);
});
