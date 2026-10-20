import { describe, expect, it } from "vitest";
import { jsonLdScript } from "@/lib/json-ld";

// Test escaper JSON-LD.
//
// Nilai JSON-LD untuk karya berisi field yang dikendalikan user (judul,
// deskripsi). JSON.stringify tidak meng-escape "<", jadi payload
// "</script><script>..." bisa menutup elemen script lebih awal dan
// menyisipkan JavaScript ke halaman publik — stored XSS.
//
// Next.js mendokumentasikan hal yang sama untuk JSON-LD.
describe("jsonLdScript", () => {
  it("meng-escape '<' supaya tidak bisa menutup elemen script", () => {
    const keluar = jsonLdScript({ name: "</script><script>alert(1)</script>" });
    expect(keluar).not.toContain("</script>");
    expect(keluar).not.toContain("<script");
    expect(keluar).toContain("\\u003c");
  });

  it("menjaga JSON tetap valid setelah di-escape", () => {
    const keluar = jsonLdScript({ name: "Karya <b>tebal</b>", n: 1, ok: true });
    const parsed = JSON.parse(keluar);
    expect(parsed.name).toBe("Karya <b>tebal</b>");
    expect(parsed.n).toBe(1);
    expect(parsed.ok).toBe(true);
  });

  it("tidak mengubah nilai tanpa karakter berbahaya", () => {
    const keluar = jsonLdScript({ name: "Karya biasa", list: [1, 2, 3] });
    expect(JSON.parse(keluar)).toEqual({ name: "Karya biasa", list: [1, 2, 3] });
  });

  it("menangani payload di dalam string bersarang", () => {
    const keluar = jsonLdScript({
      author: { name: "<img src=x onerror=alert(1)>" },
    });
    expect(keluar).not.toContain("<img");
    expect(JSON.parse(keluar).author.name).toBe("<img src=x onerror=alert(1)>");
  });
});
