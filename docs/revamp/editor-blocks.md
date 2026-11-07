# Grafikarsa Revamp — Spesifikasi Editor & Blok

> Format `body` di `content_drafts` / `content_versions`. Semua komponen UI editor memakai **shadcn/ui**
> dan font **Inter** (PRD §14.3).

---

## 1. Prinsip

1. **JSON terstruktur, bukan HTML.** Tidak pernah menyimpan HTML dari user. Halaman publik dirender dari
   JSON lewat allowlist node/blok. Node/blok tak dikenal = ditolak saat simpan.
2. **Divalidasi dua kali**: zod di web (feedback cepat) dan Go di api (sumber kebenaran). Skema keduanya
   di-generate dari satu definisi JSON Schema di `packages/contracts/blocks/` supaya tidak drift.
3. **Embed dibangun server.** User menempel URL, server mengenali provider dari allowlist dan menyimpan
   `provider` + `id`/URL kanonik. `src` iframe disusun saat render, bukan disimpan dari input user.
4. **Setiap blok/node level-atas punya `id`** (UUID v7). Dipakai untuk catatan moderasi per blok, anchor
   daftar isi, diff revisi, dan reorder.
5. **`body_schema`** (saat ini `1`) disimpan per draft/versi. Perubahan format = naikkan versi + migrator.

---

## 2. Field umum (di luar `body`)

| Field | Portfolio | Blog |
|---|---|---|
| `title` | Wajib, 3–150 karakter | Wajib, 3–150 karakter |
| `excerpt` | Opsional, ≤ 300 (otomatis dari blok teks pertama bila kosong) | **Wajib**, 50–300 |
| `cover_media_id` | **Wajib**, gambar, sisi terpendek ≥ 630 px, rasio disarankan 16:9 | Opsional (gambar OG otomatis bila kosong) |
| `category_id` | Wajib (kategori portfolio) | Wajib (kategori blog) |
| `tag_ids` | ≤ 10 | ≤ 10 |
| `tool_ids` | ≤ 20 | – |
| `skill_ids` | ≤ 20 | – |
| `meta.context` | Wajib | – |
| `meta.period` | Opsional (`YYYY-MM`) | – |
| `meta.links` | ≤ 5 tautan `https` | – |
| `meta.series_id` | – | Opsional |

Slug dibuat otomatis dari judul (transliterasi, huruf kecil, `-`, maks 120 karakter).

---

## 3. Portfolio: editor blok

### 3.1 Bentuk

```jsonc
{
  "blocks": [
    { "id": "0193...", "type": "image", "data": { "media_id": "0193...", "alt": "Poster HUT 80", "caption": "Final", "fit": "contain" } },
    { "id": "0193...", "type": "text",  "data": { "doc": { /* ProseMirror doc terbatas */ } } }
  ]
}
```

### 3.2 Daftar blok

| `type` | `data` | Batas & aturan |
|---|---|---|
| `image` | `media_id`, `alt` (≤ 200), `caption` (≤ 300), `fit`: `contain`\|`wide` | Gambar JPG/PNG/WebP/AVIF/GIF ≤ 10 MB. `alt` sangat disarankan (peringatan, bukan error) |
| `gallery` | `items[]` {`media_id`, `alt`, `caption`}, `layout`: `grid`\|`carousel` | 2–20 gambar |
| `video` | `provider`: `youtube`\|`vimeo`, `video_id`, `start` (detik, opsional), `title` | URL ditempel, server mengekstrak ID. YouTube dirender lewat `youtube-nocookie.com` |
| `embed` | `provider`, `url` kanonik, `aspect`: `16:9`\|`4:3`\|`1:1`\|`9:16` | Allowlist §3.3 |
| `file` | `media_id`, `title` (≤ 120) | PDF ≤ 25 MB, dipratinjau inline (render halaman ke gambar oleh worker + unduh). Dokumen Office hanya dari migrasi v1 (unduh saja) |
| `model3d` | `media_id` (`.glb`), `poster_media_id`, `auto_rotate` | ≤ 30 MB, dirender `<model-viewer>` |
| `text` | `doc`: ProseMirror terbatas (paragraf, H2–H3, list, bold, italic, link, kode inline) | ≤ 5.000 karakter teks per blok |
| `code` | `language` (allowlist ±30 bahasa), `code`, `filename` (opsional) | ≤ 10.000 karakter, syntax highlight saat render server (Shiki) |
| `table` | `headers[]`, `rows[][]` | ≤ 10 kolom, ≤ 50 baris, sel ≤ 200 karakter |
| `link` | `label` (≤ 60), `url` (`https`), `style`: `button`\|`card` | `card` menampilkan domain + ikon. Tidak ada fetch OG (menghindari SSRF) |
| `divider` | – | – |

### 3.3 Allowlist embed

| `provider` | URL yang diterima | Catatan |
|---|---|---|
| `figma` | `figma.com/file|design|proto|board/...` | Embed resmi Figma |
| `canva` | `canva.com/design/.../view` | Wajib link "view" publik |
| `google_slides` | `docs.google.com/presentation/d/.../` | Dikonversi ke `/embed` |
| `google_docs` | `docs.google.com/document/d/.../` | Dikonversi ke `/preview` |
| `google_drive` | `drive.google.com/file/d/.../` | Dikonversi ke `/preview` (PDF/video di Drive) |
| `sketchfab` | `sketchfab.com/3d-models/...` | Model 3D eksternal |
| `codepen` | `codepen.io/<user>/pen/<id>` | |
| `codesandbox` | `codesandbox.io/s/...`, `/p/...` | |
| `spotify` | `open.spotify.com/(track|episode|show)/...` | Karya audio / podcast |

Semua iframe dirender dengan `sandbox` minimal yang dibutuhkan provider, `loading="lazy"`,
`referrerpolicy="strict-origin-when-cross-origin"`, dan dimuat hanya setelah terlihat (facade/klik
untuk YouTube supaya halaman tetap cepat). Daftar ini juga menjadi `frame-src` di CSP.

### 3.4 Aturan portfolio

- 1–60 blok. Minimal **1 blok media** (`image`, `gallery`, `video`, `embed`, `file`, `model3d`).
- Total media unggahan per portfolio ≤ 50 file.
- Urutan blok bebas (drag & drop dnd-kit, juga tombol naik/turun untuk keyboard).

---

## 4. Blog: editor dokumen (TipTap / ProseMirror)

### 4.1 Bentuk

```jsonc
{
  "type": "doc",
  "content": [
    { "type": "heading", "attrs": { "id": "0193...", "level": 2 }, "content": [{ "type": "text", "text": "Persiapan" }] },
    { "type": "paragraph", "attrs": { "id": "0193..." }, "content": [ ... ] }
  ]
}
```

### 4.2 Node

| Node | Attrs | Catatan |
|---|---|---|
| `paragraph` | `id` | |
| `heading` | `id`, `level` 2–4 | H1 dipakai judul, tidak boleh di body. Dipakai untuk daftar isi |
| `bulletList`, `orderedList`, `listItem` | `id` (list) | Nesting ≤ 3 |
| `blockquote` | `id` | |
| `codeBlock` | `id`, `language` | ≤ 10.000 karakter per blok |
| `callout` | `id`, `variant`: `info`\|`tip`\|`warning` | Komponen shadcn `Alert` |
| `image` | `id`, `media_id`, `alt`, `caption`, `fit` | Sama dengan blok `image` portfolio |
| `embed` | `id`, `provider`, `url`/`video_id`, `aspect` | Allowlist §3.3 + `youtube`/`vimeo` |
| `portfolioCard` | `id`, `public_id` | Kartu ke portfolio tayang. Kalau portfolio tidak lagi tayang, kartu tidak dirender |
| `table`, `tableRow`, `tableCell`, `tableHeader` | `id` (table) | ≤ 10 kolom, ≤ 50 baris |
| `horizontalRule` | `id` | |
| `hardBreak`, `text` | – | |

### 4.3 Mark

`bold`, `italic`, `strike`, `code`, `link` (`href` wajib `https://` atau path internal `/…`).
Link eksternal dirender `rel="nofollow ugc noopener noreferrer"` + `target="_blank"`.

### 4.4 Aturan blog

- 150–20.000 kata. Gambar ≤ 30, embed ≤ 15.
- `reading_minutes` = `ceil(jumlah kata / 200)`, minimal 1.
- Daftar isi otomatis dari H2–H3 (anchor = `id` node).
- Fitur editor: slash command (`/`), toolbar mengambang untuk mark, tempel gambar = langsung unggah,
  markdown shortcut (`##`, `-`, `>`, ```` ``` ````).

---

## 5. Perilaku editor (umum)

| Aspek | Perilaku |
|---|---|
| Autosave | Debounce 1,5 detik setelah berhenti mengetik + saat pindah fokus / menutup tab. Indikator "Tersimpan · 10.21" |
| Konflik antar-editor | Optimistic locking `lock_version`. Kalau versi server lebih baru → dialog: muat ulang / timpa (hanya pemilik) / salin teks saya |
| Presence | "Dinda sedang mengedit" (heartbeat 20 detik di Valkey). Bukan edit realtime bersama |
| Unggah | Progress per file, bisa batal. Blok menampilkan placeholder blurhash sampai varian siap |
| Validasi | Inline saat mengetik (batas karakter), ringkasan error saat "Kirim ke Moderasi" dengan tautan ke blok bermasalah |
| Catatan moderasi | Saat `changes_requested`, blok/node yang diberi catatan ditandai dan catatannya tampil di samping |
| Pratinjau | Memakai renderer yang **sama persis** dengan halaman publik |
| Terkunci | Saat `in_review`, editor read-only + banner + tombol "Tarik Kembali" (jika belum di-claim) |
| Aksesibilitas | Semua aksi bisa via keyboard, label ARIA, fokus terlihat, kontras AA |

---

## 6. Diff revisi untuk moderator

- Field umum: dibandingkan per field (judul, excerpt, cover, kategori, tag, tools, skill, meta).
- Portfolio: dicocokkan per `id` blok → status `ditambah` / `dihapus` / `diubah` / `dipindah`. Blok `text`
  dan `table` diubah ditampilkan diff kata.
- Blog: dicocokkan per `id` node level-atas, lalu diff kata di dalam node yang berubah.
- Media baru ditandai jelas (moderator wajib melihat gambar baru).

---

## 7. Pemetaan blok v1 → revamp (untuk migrasi)

| Blok v1 | Jadi | Aturan |
|---|---|---|
| `text` (`content`) | `text` | HTML v1 disanitasi lalu dikonversi ke ProseMirror terbatas. Lebih dari 5.000 karakter → dipecah jadi beberapa blok `text` di batas paragraf |
| `image` (`url`, `caption`) | `image` | `url` → `media_id` lewat peta media. Gagal → karantina |
| `table` (`headers`, `rows`) | `table` | Melebihi batas → dipecah per 50 baris. Lebih dari 10 kolom → karantina |
| `youtube` (`video_id`) | `video` (youtube) | |
| `button` (`text`, `url`) | `link` (button) | URL non-https → dicoba https, gagal → karantina |
| `embed` (`html`) | `embed` / `link` | `src` iframe diambil. Provider di allowlist → `embed`. Selain itu → `link` ke URL tersebut + catatan karantina. HTML mentah **tidak pernah** dibawa |
| `figma` | `embed` (figma) | |
| `canva` | `embed` (canva) | |
| `ppt` (`google_slides`) | `embed` (google_slides) | |
| `ppt` (`upload`) | `file` | Dokumen Office, unduh saja |
| `pdf` | `file` | |
| `doc` | `file` / `embed` (google_docs) | URL Google Docs → embed, file unggahan → `file` |
