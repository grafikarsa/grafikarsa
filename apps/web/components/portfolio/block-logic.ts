// Logika murni editor block (tanpa JSX) supaya bisa diuji unit test dan tetap
// sinkron dengan validator backend (validateBlockPayload).
export const BLOCK_TYPES = [
  { value: "text", label: "Teks (Markdown)" },
  { value: "image", label: "Gambar" },
  { value: "table", label: "Tabel" },
  { value: "youtube", label: "YouTube" },
  { value: "button", label: "Tombol" },
  { value: "link", label: "Tautan" },
  { value: "embed", label: "Embed (CodePen/CodeSandbox)" },
  { value: "figma", label: "Figma" },
  { value: "canva", label: "Canva" },
  { value: "ppt", label: "Slide (PPT/Google Slides)" },
  { value: "pdf", label: "PDF (URL)" },
  { value: "doc", label: "Dokumen (Google Docs)" },
] as const;

export type BlockType = (typeof BLOCK_TYPES)[number]["value"];

// EXTRA: field tambahan per tipe (selain payload utama) yang ditampilkan editor.
export type Draft = Record<string, unknown>;

export function isHttps(u: string): boolean {
  return /^https:\/\/\S+$/.test(u);
}

export function ytId(url: string): string | null {
  const m = url.match(/(?:v=|youtu\.be\/|embed\/)([A-Za-z0-9_-]{11})/);
  return m ? m[1] : null;
}

// parseTable: baris "a | b | c" -> rows; header dari input koma.
export function parseTable(
  headersRaw: string,
  rowsRaw: string,
): { headers: string[]; rows: string[][] } | null {
  const headers = headersRaw
    .split(",")
    .map((h) => h.trim())
    .filter(Boolean);
  if (headers.length === 0 || headers.length > 10) return null;
  const rows = rowsRaw
    .split("\n")
    .map((r) => r.trim())
    .filter(Boolean)
    .map((r) => r.split("|").map((c) => c.trim()))
    .filter((r) => r.length > 0);
  if (rows.length > 50) return null;
  return { headers, rows };
}

// validateDraft: kembalikan pesan error pertama, atau null bila valid.
// Aturan ini cerminan validateBlockPayload di backend.
export function validateDraft(blockType: string, d: Draft): string | null {
  switch (blockType) {
    case "text":
      return typeof d.markdown === "string" && d.markdown.trim().length > 0
        ? null
        : "Teks tidak boleh kosong.";
    case "image":
      if (!d.media_id && !(typeof d.url === "string" && isHttps(d.url)))
        return "Unggah gambar atau isi URL https.";
      if (typeof d.alt !== "string" || d.alt.trim().length < 5)
        return "Alt wajib 5+ karakter (aksesibilitas).";
      return null;
    case "table":
      return validateTableDraft(d);
    case "youtube":
      return typeof d.video_id === "string" && d.video_id.length === 11
        ? null
        : "ID video YouTube 11 karakter.";
    case "button":
      if (typeof d.label !== "string" || d.label.trim().length === 0 || d.label.length > 50)
        return "Label 1-50 karakter.";
      return typeof d.url === "string" && isHttps(d.url) ? null : "URL tombol harus https.";
    case "link":
      return typeof d.url === "string" && isHttps(d.url) ? null : "URL tautan harus https.";
    default:
      return typeof d.url === "string" && isHttps(d.url) ? null : "URL harus https.";
  }
}

function validateTableDraft(d: Draft): string | null {
  const h = d.headers;
  if (!Array.isArray(h) || h.length === 0 || h.length > 10) return "Header 1-10 kolom.";
  const rows = d.rows;
  if (Array.isArray(rows) && rows.length > 50) return "Maksimal 50 baris.";
  return null;
}

export function cleanPayload(blockType: string, d: Draft): Draft {
  const out: Draft = { ...d };
  for (const [k, v] of Object.entries(out)) {
    if (v === "" || v === null || v === undefined) delete out[k];
    if (Array.isArray(v) && v.length === 0 && k !== "rows") delete out[k];
  }
  if (blockType === "table" && !Array.isArray(out.rows)) out.rows = [];
  if (blockType === "text" && typeof out.markdown === "string") out.markdown = out.markdown.trim();
  if (typeof out.url === "string") out.url = out.url.trim();
  return out;
}
