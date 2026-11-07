# Klien API

Folder `gen/` berisi kode klien TypeScript yang di-generate dari
`packages/contracts/openapi.yaml` menggunakan orval.

- Jangan pernah mengubah berkas di dalam `gen/` secara manual.
- Setelah kontrak OpenAPI berubah, jalankan `pnpm --filter web api:generate`
  dari root repo, lalu komit seluruh perubahan di `gen/`.
- `fetcher.ts` adalah pembungkus `fetch` kustom: base URL, credential, dan
  error RFC 9457 (`ApiError` + `Problem`).
