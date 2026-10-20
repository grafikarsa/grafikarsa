import { defineConfig, devices } from "@playwright/test";

// E2E Playwright (PRD §14.3) + a11y axe.
//
// Server yang diuji adalah aplikasi Next.js yang sudah built (next start),
// bukan dev server — supaya yang diuji sama dengan yang dideploy.
//
// Butuh:
//   - API Go hidup di INTERNAL_API_URL (BFF mem-proxy /api/* ke sana)
//   - Database sudah dimigrasi + di-seed (akun admin)
//
// Di CI, job "e2e" menyalakan keduanya lalu menjalankan `pnpm exec playwright test`.
// Locally: `make dev` + `pnpm --dir apps/web exec playwright test`.
export default defineConfig({
  testDir: "./e2e",
  timeout: 30_000,
  expect: { timeout: 10_000 },
  fullyParallel: false,
  workers: 1, // serial: satu database dipakai bersama
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [["github"], ["list"]] : [["list"]],
  use: {
    baseURL: process.env.E2E_BASE_URL || "http://127.0.0.1:3000",
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    ignoreHTTPSErrors: true,
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
});
