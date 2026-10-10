"use client";

import { useRef, useState } from "react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { SelectField } from "@/components/ui/select-field";
import { Textarea } from "@/components/ui/textarea";
import { apiSend } from "@/lib/api";
import { useAuthStore } from "@/lib/auth-store";

// Semua tipe block yang didukung backend (validateBlockPayload + JSON Schema di
// packages/contracts/blocks/*.json). Editor memvalidasi field wajib lebih dulu
// supaya pesan errornya jelas sebelum request dikirim.
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
export function parseTable(headersRaw: string, rowsRaw: string): { headers: string[]; rows: string[][] } | null {
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
      return typeof d.markdown === "string" && d.markdown.trim().length > 0 ? null : "Teks tidak boleh kosong.";
    case "image":
      if (!d.media_id && !(typeof d.url === "string" && isHttps(d.url))) return "Unggah gambar atau isi URL https.";
      if (typeof d.alt !== "string" || d.alt.trim().length < 5) return "Alt wajib 5+ karakter (aksesibilitas).";
      return null;
    case "table":
      return validateTableDraft(d);
    case "youtube":
      return typeof d.video_id === "string" && d.video_id.length === 11 ? null : "ID video YouTube 11 karakter.";
    case "button":
      if (typeof d.label !== "string" || d.label.trim().length === 0 || d.label.length > 50) return "Label 1-50 karakter.";
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

// useUpload: unggah media dengan progress (XHR supaya progress dapat dilacak).
export function useUpload() {
  const [progress, setProgress] = useState<number | null>(null);
  async function upload(purpose: string, file: File): Promise<{ media_id: string; width?: number; height?: number }> {
    const { accessToken, csrfToken } = useAuthStore.getState();
    const fd = new FormData();
    fd.append("purpose", purpose);
    fd.append("file", file);
    return new Promise((resolve, reject) => {
      const xhr = new XMLHttpRequest();
      xhr.open("POST", "/api/v1/media/upload");
      if (accessToken) xhr.setRequestHeader("Authorization", `Bearer ${accessToken}`);
      if (csrfToken) xhr.setRequestHeader("X-CSRF-Token", csrfToken);
      xhr.upload.onprogress = (e) => {
        if (e.lengthComputable) setProgress(Math.round((e.loaded / e.total) * 100));
      };
      xhr.onload = () => {
        setProgress(null);
        try {
          const json = JSON.parse(xhr.responseText);
          if (!json.success) throw new Error(json.error?.message ?? "Upload gagal.");
          resolve(json.data);
        } catch (err) {
          reject(err instanceof Error ? err : new Error("Upload gagal."));
        }
      };
      xhr.onerror = () => {
        setProgress(null);
        reject(new Error("Upload gagal."));
      };
      xhr.send(fd);
    });
  }
  return { upload, progress };
}

// BlockForm: satu form untuk membuat ATAU mengedit block (dipakai editor).
export function BlockForm({
  blockType,
  initial,
  busy,
  onSubmit,
  onCancel,
  submitLabel,
}: {
  blockType: BlockType;
  initial?: Draft;
  busy?: boolean;
  onSubmit: (draft: Draft) => Promise<void> | void;
  onCancel?: () => void;
  submitLabel?: string;
}) {
  const init = initial ?? {};
  const [draft, setDraft] = useState<Draft>(init);
  const [error, setError] = useState<string | null>(null);
  const { upload, progress } = useUpload();
  const fileRef = useRef<HTMLInputElement | null>(null);

  function set(k: string, v: unknown) {
    setDraft((d) => ({ ...d, [k]: v }));
  }

  async function handleUpload(file: File) {
    try {
      const res = await upload(blockType === "pdf" ? "document" : "block_image", file);
      set("media_id", res.media_id);
      if (blockType === "image" && !draft.alt) {
        set("alt", `Gambar ${file.name.replace(/\.[a-z0-9]+$/i, "")}`.slice(0, 200));
      }
      toast.success("File terunggah.");
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "Upload gagal.");
    }
  }

  async function submit() {
    const payload = cleanPayload(blockType, draft);
    const vErr = validateDraft(blockType, payload);
    if (vErr) {
      setError(vErr);
      return;
    }
    setError(null);
    await onSubmit(payload);
    if (!initial) setDraft({});
  }

  return (
    <div className="flex flex-col gap-2 rounded border p-3">
      {blockType === "text" && (
        <>
          <Label htmlFor="b-text">Markdown</Label>
          <Textarea
            id="b-text"
            rows={5}
            value={String(draft.markdown ?? "")}
            onChange={(e) => set("markdown", e.target.value)}
            placeholder="Tulis penjelasan karya (mendukung markdown)."
          />
        </>
      )}

      {blockType === "image" && (
        <>
          <div className="flex flex-wrap items-end gap-2">
            <input
              ref={fileRef}
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif"
              className="text-sm"
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) handleUpload(f).catch(() => null);
              }}
            />
            {progress !== null && <span className="text-xs">Mengunggah… {progress}%</span>}
          </div>
          <div className="grid gap-1">
            <Label htmlFor="b-alt">Alt (5-200 karakter, untuk pembaca layar)</Label>
            <Input id="b-alt" value={String(draft.alt ?? "")} onChange={(e) => set("alt", e.target.value)} />
          </div>
          <div className="grid gap-1">
            <Label htmlFor="b-caption">Caption (opsional)</Label>
            <Input id="b-caption" value={String(draft.caption ?? "")} onChange={(e) => set("caption", e.target.value)} />
          </div>
          {typeof draft.media_id === "string" && draft.media_id !== "" && (
            <p className="text-muted-foreground text-xs">Media siap: {String(draft.media_id).slice(0, 8)}…</p>
          )}
        </>
      )}

      {blockType === "table" && (
        <>
          <div className="grid gap-1">
            <Label htmlFor="b-headers">Header (pisahkan dengan koma)</Label>
            <Input
              id="b-headers"
              defaultValue={Array.isArray(draft.headers) ? (draft.headers as string[]).join(", ") : ""}
              onChange={(e) => set("headers", e.target.value.split(",").map((h) => h.trim()).filter(Boolean))}
              placeholder="Alat, Fungsi, Hasil"
            />
          </div>
          <div className="grid gap-1">
            <Label htmlFor="b-rows">Baris (satu baris per record, sel dipisah |)</Label>
            <Textarea
              id="b-rows"
              rows={4}
              defaultValue={Array.isArray(draft.rows) ? (draft.rows as string[][]).map((r) => r.join(" | ")).join("\n") : ""}
              onChange={(e) => {
                const parsed = parseTable(
                  Array.isArray(draft.headers) ? (draft.headers as string[]).join(",") : "",
                  e.target.value,
                );
                set("rows", parsed ? parsed.rows : []);
              }}
              placeholder="Figma | Desain UI | Selesai"
            />
          </div>
        </>
      )}

      {blockType === "youtube" && (
        <div className="grid gap-1">
          <Label htmlFor="b-yt">URL YouTube</Label>
          <Input
            id="b-yt"
            defaultValue={typeof draft.video_id === "string" ? `https://youtu.be/${draft.video_id}` : ""}
            onChange={(e) => set("video_id", ytId(e.target.value) ?? "")}
            placeholder="https://www.youtube.com/watch?v=..."
          />
        </div>
      )}

      {blockType === "button" && (
        <>
          <div className="grid gap-1">
            <Label htmlFor="b-label">Label</Label>
            <Input id="b-label" value={String(draft.label ?? "")} onChange={(e) => set("label", e.target.value)} />
          </div>
          <div className="grid gap-1">
            <Label htmlFor="b-url">URL (https)</Label>
            <Input id="b-url" value={String(draft.url ?? "")} onChange={(e) => set("url", e.target.value)} />
          </div>
          <div className="grid gap-1">
            <Label>Varian</Label>
            <SelectField
              value={String(draft.variant ?? "primary")}
              onValueChange={(v) => set("variant", v)}
              options={[
                { value: "primary", label: "Primary" },
                { value: "secondary", label: "Secondary" },
              ]}
            />
          </div>
        </>
      )}

      {blockType === "link" && (
        <>
          <div className="grid gap-1">
            <Label htmlFor="b-ltitle">Judul (opsional)</Label>
            <Input id="b-ltitle" value={String(draft.title ?? "")} onChange={(e) => set("title", e.target.value)} />
          </div>
          <div className="grid gap-1">
            <Label htmlFor="b-lurl">URL (https)</Label>
            <Input id="b-lurl" value={String(draft.url ?? "")} onChange={(e) => set("url", e.target.value)} />
          </div>
        </>
      )}

      {["embed", "figma", "canva", "ppt", "doc"].includes(blockType) && (
        <div className="grid gap-1">
          <Label htmlFor="b-eurl">URL embed (https, domain diizinkan)</Label>
          <Input
            id="b-eurl"
            value={String(draft.url ?? "")}
            onChange={(e) => set("url", e.target.value)}
            placeholder={
              blockType === "figma"
                ? "https://www.figma.com/file/..."
                : blockType === "canva"
                  ? "https://www.canva.com/design/..."
                  : "https://codepen.io/..."
            }
          />
        </div>
      )}

      {blockType === "pdf" && (
        <>
          <input
            type="file"
            accept="application/pdf"
            className="text-sm"
            onChange={(e) => {
              const f = e.target.files?.[0];
              if (f) handleUpload(f).catch(() => null);
            }}
          />
          {progress !== null && <span className="text-xs">Mengunggah… {progress}%</span>}
          <div className="grid gap-1">
            <Label htmlFor="b-purl">atau URL PDF (https)</Label>
            <Input id="b-purl" value={String(draft.url ?? "")} onChange={(e) => set("url", e.target.value)} />
          </div>
        </>
      )}

      {error && <p className="text-destructive text-xs">{error}</p>}

      <div className="flex gap-2">
        <Button size="sm" onClick={submit} disabled={busy}>
          {submitLabel ?? (initial ? "Simpan perubahan" : "Tambah block")}
        </Button>
        {onCancel && (
          <Button size="sm" variant="ghost" onClick={onCancel} disabled={busy}>
            Batal
          </Button>
        )}
      </div>
    </div>
  );
}

// cleanPayload membuang field kosong agar payload sesuai skema backend.
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

// applyBlockEdit mengirim perubahan payload satu block.
export async function updateBlock(blockID: string, payload: Draft): Promise<void> {
  await apiSend(`v1/blocks/${blockID}`, "PATCH", { payload });
}
