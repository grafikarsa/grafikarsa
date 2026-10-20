import AxeBuilder from "@axe-core/playwright";
import { expect, test, type Page } from "@playwright/test";

// E2E alur utama (PRD §14.3): login → jelajah → admin + audit a11y.
//
// Cakupannya sengaja fokus pada alur yang paling sering rusak dan paling
// berisiko (sesi, RBAC, halaman publik). Pembuatan portfolio lengkap tidak
// diuji di sini karena butuh thumbnail + block + tag + series sekaligus;
// itu diuji di integration test API.

const ADMIN_USER = process.env.E2E_USER || "admin";
const ADMIN_PASS = process.env.E2E_PASS || "AdminContractTest123";

/**
 * Login sebagai admin dan tunggu sampai sesi benar-benar terbentuk.
 *
 * Penting: setelah menekan tombol, harus MENUNGGU navigasi selesai dulu.
 * Cookie sesi baru ada setelah respons login diterima; kalau langsung
 * `goto()` ke halaman privat, middleware masih melihat request tanpa cookie
 * dan mengarahkan ke /login — bukan karena aplikasinya salah, tapi karena
 * test-nya lebih dulu.
 */
async function login(page: Page) {
  await page.goto("/login");
  await page.getByLabel(/username/i).fill(ADMIN_USER);
  await page.getByLabel(/password/i).fill(ADMIN_PASS);
  await page.getByRole("button", { name: /masuk/i }).click();
  // Tunggu navigasi keluar dari /login (login sukses → router.push("/")).
  await page.waitForURL((u) => !u.pathname.startsWith("/login"), { timeout: 20_000 });
  // Dan pastikan cookie sesi sudah ada. Cookie sesi httpOnly, jadi tidak
  // bisa dibaca lewat document.cookie — harus lewat API context Playwright.
  const cookies = await page.context().cookies();
  const sesi = cookies.find((c) => c.name === "rt" || c.name === "__Host-rt");
  if (!sesi) {
    throw new Error(
      `cookie sesi tidak ditemukan setelah login (cookie yang ada: ${cookies.map((c) => c.name).join(", ") || "tidak ada"})`,
    );
  }
}

test.describe("alur autentikasi", () => {
  test("halaman login memuat form dan menampilkan error untuk kredensial salah", async ({
    page,
  }) => {
    await page.goto("/login");
    await expect(page.getByRole("heading", { name: /masuk/i })).toBeVisible();

    await page.getByLabel(/username/i).fill(ADMIN_USER);
    await page.getByLabel(/password/i).fill("PasswordSalahSekali123");
    await page.getByRole("button", { name: /masuk/i }).click();

    // Pesan error dari server harus tampil (bukan diam-diam).
    //
    // Selector memakai atribut data-sonner-toast milik komponen toast, bukan
    // teksnya: teks toast bisa berubah (kode error vs pesan) dan toast
    // hilang sendiri, jadi pencarian teks rawan flake.
    const toast = page.locator("[data-sonner-toast]").first();
    await expect(toast).toBeVisible({ timeout: 20_000 });
    await expect(toast).toContainText(/salah|invalid|gagal|credential/i);
  });

  test("login admin berhasil lalu /me bisa diakses", async ({ page }) => {
    await login(page);
    await page.goto("/me");
    // /me mengarahkan ke profil sendiri.
    await expect(page).not.toHaveURL(/\/login/);
    await expect(page).toHaveURL(/\/admin$/);
  });
});

test.describe("halaman publik", () => {
  test("landing memuat leaderboard dan daftar rilis", async ({ page }) => {
    await page.goto("/");
    await expect(page).toHaveTitle(/grafikarsa/i);
    await expect(page.getByRole("heading").first()).toBeVisible();
  });

  test("feed, leaderboard, katalog, dan pencarian bisa diakses tanpa login", async ({ page }) => {
    for (const path of ["/feed", "/leaderboards", "/portfolios", "/search?q=karya"]) {
      const res = await page.goto(path);
      expect(res?.status(), `halaman ${path} harus 200`).toBe(200);
      await expect(page.getByRole("heading").first()).toBeVisible();
    }
  });
});

test.describe("halaman admin", () => {
  test("dashboard admin bisa diakses setelah login admin", async ({ page }) => {
    await login(page);
    await page.goto("/admin/dashboard");
    await expect(page.getByRole("heading").first()).toBeVisible();
  });
});

// A11y (PRD §14.3: axe, WCAG AA).
//
// Yang diperiksa: halaman yang paling sering diakses (landing, login, feed).
// Aturan serious/critical membuat build gagal; moderate dilaporkan tapi
// tidak memblokir, supaya tim bisa memperbaikinya bertahap.
test.describe("a11y (axe, WCAG AA)", () => {
  for (const path of ["/", "/login", "/feed"]) {
    test(`${path} tidak punya pelanggaran axe serious/critical`, async ({ page }) => {
      await page.goto(path);
      // Beri waktu FE merender + fetch data.
      await page.waitForTimeout(1500);

      const results = await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21a", "wcag21aa"])
        .analyze();

      const serious = results.violations.filter(
        (v) => v.impact === "serious" || v.impact === "critical",
      );
      const report = serious
        .map((v) => `  [${v.impact}] ${v.id}: ${v.help} (${v.nodes.length} node)`)
        .join("\n");
      expect(serious.length, `pelanggaran a11y di ${path}:\n${report}`).toBe(0);
    });
  }
});
