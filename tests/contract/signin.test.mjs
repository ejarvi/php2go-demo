// design/modules/signin.md, acceptance tests 1–6.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Client, ALT_BASE_URL, BASE_URL, signIn, formFields } from './lib.mjs';

test('1. the right password sets the session cookie and redirects to /admin', async () => {
  const client = new Client();
  const response = await signIn(client);
  assert.equal(response.status, 303);
  assert.equal(response.location, '/admin');
  assert.ok(client.cookies.has('session'));
});

test('2. a wrong password sets no cookie and shows the error', async () => {
  const client = new Client();
  const response = await signIn(client, 'not-the-password');
  assert.ok(!client.cookies.has('session'));
  assert.ok(!response.setCookies.some((c) => c.startsWith('session=')));
  assert.match(response.text, /That password isn(’|'|&#0*39;)t right\./);
});

test('3. with a session /admin is 200; after signing out it redirects to /signin', async () => {
  const client = new Client();
  await signIn(client);
  const admin = await client.get('/admin');
  assert.equal(admin.status, 200);

  const signout = await client.post('/signout', formFields(admin.text, '/signout'));
  assert.equal(signout.status, 303);
  const after = await client.get('/admin');
  assert.equal(after.status, 303);
  assert.match(after.location, /^\/signin/);
});

test('4. a session started before the password changed is refused', { skip: !ALT_BASE_URL && 'ALT_BASE_URL is not set' }, async () => {
  const client = new Client();
  await signIn(client);
  assert.equal((await client.get('/admin')).status, 200);

  // The same storage, served with a different password: as if the password had changed.
  const changed = new Client(ALT_BASE_URL);
  changed.cookies = new Map(client.cookies);
  const response = await changed.get('/admin');
  assert.equal(response.status, 303);
  assert.match(response.location, /^\/signin/);
});

test('5. the session cookie is Secure, HttpOnly and SameSite=Lax', async () => {
  const response = await signIn(new Client());
  const cookie = response.setCookies.find((c) => c.startsWith('session='));
  assert.ok(cookie, 'a session cookie is set');
  assert.match(cookie, /;\s*Secure/i);
  assert.match(cookie, /;\s*HttpOnly/i);
  assert.match(cookie, /;\s*SameSite=Lax/i);
});

test('6. sign-in works with a same-origin Referer, Origin and Sec-Fetch-Site (D-019, D-025)', async () => {
  const client = new Client();
  const page = await client.get('/signin');
  const response = await client.post(
    '/signin',
    { ...formFields(page.text, '/signin'), password: process.env.ORGANIZER_PASSWORD ?? 'contract-test-password' },
    { headers: { origin: new URL(BASE_URL).origin, referer: `${BASE_URL}/signin`, 'sec-fetch-site': 'same-origin' } },
  );
  assert.equal(response.status, 303);
  assert.equal(response.location, '/admin');
});
