import { defineConfig, devices } from '@playwright/test';

// Every page is checked in three browser engines at three sizes (D-018).
const engines = {
  chromium: devices['Desktop Chrome'],
  firefox: devices['Desktop Firefox'],
  webkit: devices['Desktop Safari'],
};
const sizes = {
  phone: { width: 390, height: 844 },
  laptop: { width: 1366, height: 768 },
  wide: { width: 1920, height: 1080 },
};

export default defineConfig({
  testDir: '.',
  timeout: 60_000,
  // A hang fails the gate in minutes, not when the job's own limit runs out.
  globalTimeout: 10 * 60_000,
  reporter: 'list',
  forbidOnly: true,
  use: {
    baseURL: (process.env.BASE_URL ?? 'http://127.0.0.1:8080').replace(/\/$/, ''),
  },
  projects: Object.entries(engines).flatMap(([engine, device]) =>
    Object.entries(sizes).map(([size, viewport]) => ({
      name: `${engine}-${size}`,
      use: { ...device, viewport },
    })),
  ),
});
