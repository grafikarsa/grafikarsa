import { describe, expect, it } from "vitest";
import { snappedWidth, default as mediaLoader } from "@/lib/image-loader";

// Test loader gambar (PRD §7.3).
//
// Yang dijaga: lebar yang diminta dibulatkan ke varian yang benar-benar ada
// (320/800). Kalau tidak, Next membuat srcset dengan lebar yang tidak punya
// varian, API menyajikan file asli yang jauh lebih besar untuk setiap
// request — boros bandwidth tanpa untung.

describe("snappedWidth", () => {
  it("memilih varian terkecil yang masih >= lebar diminta", () => {
    expect(snappedWidth(96)).toBe(320);
    expect(snappedWidth(320)).toBe(320);
    expect(snappedWidth(321)).toBe(800);
    expect(snappedWidth(640)).toBe(800);
  });

  it("memakai varian terbesar kalau lebar diminta lebih besar dari semua varian", () => {
    expect(snappedWidth(1200)).toBe(800);
    expect(snappedWidth(4000)).toBe(800);
  });

  it("tidak crash untuk lebar tidak valid", () => {
    expect(snappedWidth(0)).toBe(320);
    expect(snappedWidth(-10)).toBe(320);
    expect(snappedWidth(Number.NaN)).toBe(320);
  });
});

describe("mediaLoader", () => {
  it("menambahkan w & q ke URL media", () => {
    const url = mediaLoader({
      src: "/api/v1/media/abc-123",
      width: 640,
      quality: 75,
    });
    expect(url).toBe("/api/v1/media/abc-123?w=800&q=75");
  });

  it("mempertahankan query string yang sudah ada", () => {
    const url = mediaLoader({
      src: "/api/v1/media/abc-123?preview=1",
      width: 320,
      quality: 75,
    });
    expect(url).toContain("preview=1");
    expect(url).toContain("w=320");
  });

  it("meneruskan URL non-media apa adanya", () => {
    // Gambar statis / URL luar tidak punya varian — jangan diubah.
    expect(mediaLoader({ src: "/logo.png", width: 640, quality: 75 })).toBe("/logo.png");
    expect(mediaLoader({ src: "https://cdn.example.com/a.png", width: 640 })).toBe(
      "https://cdn.example.com/a.png",
    );
  });

  it("memakai lebar paling kecil untuk gambar sangat kecil", () => {
    // Avatar 32px tidak butuh varian 800 — memintanya hanya menambah
    // bytes yang diunduh.
    const url = mediaLoader({ src: "/api/v1/media/abc", width: 32 });
    expect(url).toContain("w=320");
  });
});
