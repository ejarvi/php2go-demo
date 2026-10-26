// design/modules/site.md, acceptance tests 1–5.
import { test } from 'node:test';
import assert from 'node:assert/strict';
import { Client, organizer, createEvent, uniqueTitle, formFields, PASSWORD, crossSiteHeaders, otherSubdomainHeaders } from './lib.mjs';

test('1. /health is 200 with status ok and the release ID', async () => {
  const response = await new Client().get('/health');
  assert.equal(response.status, 200);
  const body = JSON.parse(response.text);
  assert.equal(body.status, 'ok');
  assert.equal(typeof body.release, 'string');
  if (process.env.RELEASE) {
    assert.equal(body.release, process.env.RELEASE);
  }
});

test('2. every HTML response carries the security headers', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Headers') });
  for (const path of ['/', `/events/${id}`, '/signin', '/no-such-page']) {
    const { headers } = await new Client().get(path);
    const csp = headers.get('content-security-policy') ?? '';
    assert.match(csp, /script-src 'self'/, path);
    assert.match(csp, /style-src 'self'/, path);
    assert.match(csp, /frame-ancestors 'none'/, path);
    assert.doesNotMatch(csp, /unsafe-inline/, path);
    assert.equal(headers.get('referrer-policy'), 'same-origin', path);
    assert.equal(headers.get('x-content-type-options'), 'nosniff', path);
    assert.match(headers.get('strict-transport-security') ?? '', /max-age=\d+/, path);
  }
});

test('3. an unknown path is 404, in the site layout', async () => {
  const response = await new Client().get('/no-such-page');
  assert.equal(response.status, 404);
  assert.match(response.text, /<footer/);
  assert.match(response.text, /href="\/signin"/);
});

test('4. the footer of / shows the same release ID as /health', async () => {
  const client = new Client();
  const { release } = JSON.parse((await client.get('/health')).text);
  const home = await client.get('/');
  // Compared as the browser shows it: an implementation may write "+" as "&#43;".
  const footer = home.text.slice(home.text.indexOf('<footer'))
    .replace(/&#(\d+);/g, (_, n) => String.fromCharCode(Number(n)))
    .replace(/&#x([0-9a-f]+);/gi, (_, n) => String.fromCharCode(parseInt(n, 16)));
  assert.ok(footer.includes(release), `the footer shows ${release}`);
});

test('5. no page loads a script or stylesheet from another site', async () => {
  const admin = await organizer();
  const id = await createEvent(admin, { title: uniqueTitle('Local only') });
  for (const path of ['/', `/events/${id}`, '/signin', '/admin']) {
    const { text } = await admin.get(path);
    const sources = [
      ...[...text.matchAll(/<script\b[^>]*\ssrc="([^"]*)"/g)].map((m) => m[1]),
      ...[...text.matchAll(/<link\b[^>]*\shref="([^"]*)"/g)].map((m) => m[1]),
    ];
    for (const source of sources) {
      assert.ok(source.startsWith('/') && !source.startsWith('//'), `${path} loads ${source}`);
    }
  }
});

test('6. a POST from another site, or another subdomain, is refused with 403 (D-025)', async () => {
  for (const [from, headers] of [['another site', crossSiteHeaders()], ['another subdomain', otherSubdomainHeaders()]]) {
    const client = new Client();
    // A genuine form and the right password, so only where the request came from is wrong.
    const fields = { ...formFields((await client.get('/signin')).text, '/signin'), password: PASSWORD };
    const response = await client.post('/signin', fields, { headers });
    assert.equal(response.status, 403, from);
    assert.ok(!client.cookies.has('session'), `${from}: no session was started`);
  }
});
