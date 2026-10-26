import { test, expect, type Page } from '@playwright/test';
import AxeBuilder from '@axe-core/playwright';
// The acceptance tests' HTTP helpers set up the data the pages need.
import { organizer, createEvent, signUp, Client, clubDate } from '../contract/lib.mjs';

let eventId = 0;
let sessionCode = '';

test.beforeAll(async () => {
  const admin = await organizer();
  eventId = await createEvent(admin, {
    title: 'Park clean-up',
    date: clubDate(7),
    start: '09:00',
    end: '12:00',
    location: 'Riverside Park, north gate',
    description: 'Gloves and bags are provided.\nMore about the park: https://example.org/riverside-park',
    capacity: '20',
  });
  await signUp(new Client(), eventId, { name: 'Ana Example', email: 'ana@example.com', people: '2' });
  sessionCode = admin.cookies.get('session');
});

const pages: { name: string; path: () => string; organizer?: boolean }[] = [
  { name: 'events', path: () => '/' },
  { name: 'event', path: () => `/events/${eventId}` },
  { name: 'sign in', path: () => '/signin' },
  { name: 'not found', path: () => '/no-such-page' },
  { name: 'admin', path: () => '/admin', organizer: true },
  { name: 'new event', path: () => '/admin/events/new', organizer: true },
  { name: 'edit event', path: () => `/admin/events/${eventId}`, organizer: true },
];

for (const { name, path, organizer: needsSession } of pages) {
  test(`${name} page is accessible and fits the screen`, async ({ page, context, baseURL }) => {
    if (needsSession) {
      // Added directly, because some engines won't store a Secure cookie over plain HTTP.
      await context.addCookies([{ name: 'session', value: sessionCode, url: baseURL! }]);
    }
    const problems = watchConsole(page);

    await page.goto(path());

    const results = await new AxeBuilder({ page })
      .withTags(['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa'])
      .analyze();
    expect(results.violations.map((v) => `${v.id}: ${v.help} (${v.nodes.length})`)).toEqual([]);

    const overflow = await page.evaluate(() => document.documentElement.scrollWidth - document.documentElement.clientWidth);
    expect(overflow, 'no sideways scrolling').toBeLessThanOrEqual(0);

    expect(problems, 'no console errors, including Content Security Policy reports').toEqual([]);
  });
}

// Collects console errors and uncaught exceptions. A 404 status on the not-found page
// is expected, so the browser's "failed to load resource" message is ignored.
function watchConsole(page: Page): string[] {
  const problems: string[] = [];
  page.on('console', (message) => {
    if (message.type() === 'error' && !/Failed to load resource/i.test(message.text())) {
      problems.push(message.text());
    }
  });
  page.on('pageerror', (error) => problems.push(error.message));
  return problems;
}
