// Helpers for the acceptance tests (D-003). They speak plain HTTP to a running site,
// so the same tests check every implementation. No dependencies beyond Node 22.

export const BASE_URL = (process.env.BASE_URL ?? 'http://127.0.0.1:8080').replace(/\/$/, '');
// A second copy of the site on the same storage, started with a different organizer
// password. Only the password-change test needs it.
export const ALT_BASE_URL = process.env.ALT_BASE_URL?.replace(/\/$/, '');
export const PASSWORD = process.env.ORGANIZER_PASSWORD ?? 'contract-test-password';
export const TIME_ZONE = process.env.CLUB_TIMEZONE ?? 'America/Chicago';

// A browser-like client with its own cookie jar. It doesn't follow redirects, so tests
// can check them. Like a browser on localhost, it sends Secure cookies over plain HTTP.
export class Client {
  constructor(baseUrl = BASE_URL) {
    this.baseUrl = baseUrl;
    this.cookies = new Map();
  }

  async request(method, path, { form, headers = {} } = {}) {
    const sent = { ...headers };
    if (this.cookies.size > 0) {
      sent.cookie = [...this.cookies].map(([name, value]) => `${name}=${value}`).join('; ');
    }
    let body;
    if (form) {
      body = new URLSearchParams(form).toString();
      sent['content-type'] = 'application/x-www-form-urlencoded';
      // What a browser sends with a form on this site's own page (D-025).
      sent.origin ??= new URL(this.baseUrl).origin;
      sent['sec-fetch-site'] ??= 'same-origin';
    }
    const response = await fetch(this.baseUrl + path, { method, headers: sent, body, redirect: 'manual' });
    for (const cookie of response.headers.getSetCookie()) {
      this.#store(cookie);
    }
    return {
      status: response.status,
      headers: response.headers,
      location: response.headers.get('location'),
      setCookies: response.headers.getSetCookie(),
      text: await response.text(),
    };
  }

  get(path, options) {
    return this.request('GET', path, options);
  }

  post(path, form, options = {}) {
    return this.request('POST', path, { ...options, form });
  }

  #store(setCookie) {
    const [pair, ...attributes] = setCookie.split(';').map((part) => part.trim());
    const at = pair.indexOf('=');
    const name = pair.slice(0, at);
    const value = pair.slice(at + 1);
    const expires = attributes.find((a) => /^expires=/i.test(a));
    const maxAge = attributes.find((a) => /^max-age=/i.test(a));
    const expired = (maxAge && Number(maxAge.split('=')[1]) <= 0)
      || (expires && new Date(expires.slice(8)) < new Date());
    if (expired || value === '') {
      this.cookies.delete(name);
    } else {
      this.cookies.set(name, value);
    }
  }
}

// The fields of the form that posts to `action`, as a browser would submit them.
export function formFields(html, action) {
  const forms = [...html.matchAll(/<form\b([^>]*)>([\s\S]*?)<\/form>/g)];
  const form = forms.find(([, attrs]) => attr(attrs, 'action') === action);
  if (!form) {
    throw new Error(`No form posts to ${action}`);
  }
  const fields = {};
  for (const [tag] of form[2].matchAll(/<input\b[^>]*>/g)) {
    const name = attr(tag, 'name');
    const type = attr(tag, 'type') ?? 'text';
    if (!name || ((type === 'checkbox' || type === 'radio') && !/\schecked\b/.test(tag))) {
      continue;
    }
    fields[name] = attr(tag, 'value') ?? (type === 'checkbox' ? 'on' : '');
  }
  for (const [, attrs, content] of form[2].matchAll(/<textarea\b([^>]*)>([\s\S]*?)<\/textarea>/g)) {
    fields[attr(attrs, 'name')] = decode(content);
  }
  return fields;
}

function attr(tag, name) {
  const match = tag.match(new RegExp(`\\s${name}="([^"]*)"`));
  return match ? decode(match[1]) : undefined;
}

function decode(text) {
  return text
    .replace(/&quot;/g, '"')
    .replace(/&#0*39;/g, "'")
    .replace(/&lt;/g, '<')
    .replace(/&gt;/g, '>')
    .replace(/&amp;/g, '&');
}

// The headers a browser sends with a form posted from another site, or from another
// subdomain of this one (D-025).
export function crossSiteHeaders() {
  return { origin: 'https://attacker.example', 'sec-fetch-site': 'cross-site' };
}

export function otherSubdomainHeaders(baseUrl = BASE_URL) {
  const url = new URL(baseUrl);
  return { origin: `${url.protocol}//other.${url.host}`, 'sec-fetch-site': 'same-site' };
}

// Today's date in the club's time zone, moved by a number of days, as YYYY-MM-DD.
export function clubDate(offsetDays = 0) {
  const today = new Intl.DateTimeFormat('en-CA', {
    timeZone: TIME_ZONE, year: 'numeric', month: '2-digit', day: '2-digit',
  }).format(new Date());
  const date = new Date(`${today}T12:00:00Z`);
  date.setUTCDate(date.getUTCDate() + offsetDays);
  return date.toISOString().slice(0, 10);
}

// A title no other test will use, since all tests share one copy of the site.
export function uniqueTitle(label) {
  return `${label} ${Math.random().toString(36).slice(2, 8)}`;
}

export async function signIn(client, password = PASSWORD) {
  const page = await client.get('/signin');
  return client.post('/signin', { ...formFields(page.text, '/signin'), password });
}

export async function organizer() {
  const client = new Client();
  const response = await signIn(client);
  if (response.status !== 303) {
    throw new Error(`Sign-in failed with ${response.status}`);
  }
  return client;
}

// Creates an event through the organizers' form, and returns its id.
export async function createEvent(client, event) {
  const page = await client.get('/admin/events/new');
  const fields = {
    ...formFields(page.text, '/admin/events'),
    title: event.title,
    date: event.date ?? clubDate(7),
    start_time: event.start ?? '18:00',
    end_time: event.end ?? '',
    location: event.location ?? 'Community hall',
    description: event.description ?? '',
    capacity: event.capacity ?? '',
  };
  if (event.published ?? true) {
    fields.published = '1';
  }
  const response = await client.post('/admin/events', fields);
  const id = response.location?.match(/^\/admin\/events\/(\d+)/)?.[1];
  if (response.status !== 303 || !id) {
    throw new Error(`Creating an event failed with ${response.status}`);
  }
  return Number(id);
}

// Signs a visitor up through the event page's form, and returns the response.
export async function signUp(client, eventId, { name = 'Sam Visitor', email = 'sam@example.com', people = '1', ...extra } = {}) {
  const page = await client.get(`/events/${eventId}`);
  const action = `/events/${eventId}/signup`;
  return client.post(action, { ...formFields(page.text, action), name, email, people, ...extra });
}
