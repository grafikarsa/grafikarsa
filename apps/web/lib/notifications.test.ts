import { describe, expect, it } from "vitest";
import { deepLink, type Notif } from "@/lib/notifications";

// Test pemetaan notifikasi ke halaman tujuan (PRD §10).
//
// Notifikasi harus bisa diklik menuju konteksnya. Kalau deepLink mengembalikan
// null untuk notifikasi yang sebenarnya punya konteks, penguna klik notifikasi
// dan tidak terjadi apa-apa.

function notif(data: Record<string, unknown> | null): Notif {
  return {
    id: "n1",
    type: "info",
    title: "Judul",
    body: "Isi",
    created_at: "2026-01-01T00:00:00Z",
    is_read: false,
    data,
  };
}

describe("deepLink", () => {
  it("mengarah ke karya bila ada username & slug", () => {
    expect(deepLink(notif({ username: "budi", slug: "karya-1" }))).toBe("/budi/karya-1");
  });

  it("mengarah ke daftar karya dengan sorotan bila hanya ada portfolio_id", () => {
    expect(deepLink(notif({ portfolio_id: "p1" }))).toBe("/portfolios?highlight=p1");
  });

  it("mengarah ke thread feedback bila ada thread_id", () => {
    expect(deepLink(notif({ thread_id: "t1" }))).toBe("/feedback?thread=t1");
  });

  it("mengarah ke changelog bila ada version", () => {
    expect(deepLink(notif({ version: "2.3" }))).toBe("/changelog/2.3");
  });

  it("kembali ke null kalau tidak ada konteks yang bisa dipakai", () => {
    expect(deepLink(notif(null))).toBeNull();
    expect(deepLink(notif({}))).toBeNull();
    expect(deepLink(notif({ username: "budi" }))).toBeNull();
  });

  it("menganduh nilai yang bukan string", () => {
    // Data dari server bisa berisi angka/null — jangan sampai membuat URL
    // berisi "undefined" atau "null".
    expect(deepLink(notif({ portfolio_id: 123 }))).toBeNull();
    expect(deepLink(notif({ version: null }))).toBeNull();
  });
});
