import { describe, expect, it } from "vitest";
import { cleanPayload, isHttps, parseTable, validateDraft, ytId } from "@/components/portfolio/block-logic";

// Test ini menjaga aturan validasi klien tetap sinkron dengan validator Go
// (validateBlockPayload) — payload yang lolos di sini harus lolos di server.
describe("validasi block (klien)", () => {
  it("text wajib punya markdown", () => {
    expect(validateDraft("text", {})).toBeTruthy();
    expect(validateDraft("text", { markdown: "   " })).toBeTruthy();
    expect(validateDraft("text", { markdown: "halo" })).toBeNull();
  });

  it("image butuh media_id atau url https + alt 5+", () => {
    expect(validateDraft("image", { alt: "gambar" })).toBeTruthy();
    expect(validateDraft("image", { media_id: "uuid", alt: "abc" })).toBeTruthy();
    expect(validateDraft("image", { media_id: "uuid", alt: "gambar baik" })).toBeNull();
    expect(validateDraft("image", { url: "http://tidak-https", alt: "gambar baik" })).toBeTruthy();
    expect(validateDraft("image", { url: "https://x.test/a.png", alt: "gambar baik" })).toBeNull();
  });

  it("table header 1-10 kolom", () => {
    expect(validateDraft("table", { headers: [] })).toBeTruthy();
    expect(validateDraft("table", { headers: Array(11).fill("h") })).toBeTruthy();
    expect(validateDraft("table", { headers: ["a", "b"], rows: [] })).toBeNull();
  });

  it("youtube wajib 11 karakter", () => {
    expect(validateDraft("youtube", { video_id: "pendek" })).toBeTruthy();
    expect(validateDraft("youtube", { video_id: "dQw4w9WgXcQ" })).toBeNull();
  });

  it("button butuh label dan url https", () => {
    expect(validateDraft("button", { label: "", url: "https://x.test" })).toBeTruthy();
    expect(validateDraft("button", { label: "Buka", url: "https://x.test" })).toBeNull();
  });

  it("embed/figma/canva/doc/ppt butuh url https", () => {
    for (const t of ["embed", "figma", "canva", "ppt", "doc"]) {
      expect(validateDraft(t, { url: "http://x.test" })).toBeTruthy();
      expect(validateDraft(t, { url: "https://x.test" })).toBeNull();
    }
  });

  it("pdf menerima url https", () => {
    expect(validateDraft("pdf", { url: "https://x.test/a.pdf" })).toBeNull();
  });
});

describe("helper", () => {
  it("isHttps hanya menerima https", () => {
    expect(isHttps("https://a.test")).toBe(true);
    expect(isHttps("http://a.test")).toBe(false);
    expect(isHttps("ftp://a.test")).toBe(false);
  });

  it("ytId mengambil id dari berbagai bentuk URL", () => {
    expect(ytId("https://www.youtube.com/watch?v=dQw4w9WgXcQ")).toBe("dQw4w9WgXcQ");
    expect(ytId("https://youtu.be/dQw4w9WgXcQ")).toBe("dQw4w9WgXcQ");
    expect(ytId("https://x.test/abc")).toBeNull();
  });

  it("parseTable memisah header dan baris", () => {
    const t = parseTable("Alat, Fungsi", "Figma | Desain\nCanva | Poster");
    expect(t?.headers).toEqual(["Alat", "Fungsi"]);
    expect(t?.rows).toEqual([
      ["Figma", "Desain"],
      ["Canva", "Poster"],
    ]);
    expect(parseTable("", "a | b")).toBeNull();
  });

  it("cleanPayload membuang field kosong dan merapikan url", () => {
    const out = cleanPayload("button", { label: "Buka", url: "  https://a.test  ", variant: "" });
    expect(out.url).toBe("https://a.test");
    expect("variant" in out).toBe(false);
  });
});
