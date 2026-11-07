# Kontrak API

`openapi.yaml` adalah sumber kebenaran (source of truth) untuk API Grafikarsa.
Kode server Go dan klien TypeScript di-generate dari berkas ini; jangan
mengubah kode hasil generate secara manual.

- Nama field memakai camelCase.
- Error mengikuti RFC 9457 dengan media type `application/problem+json`.
- Setelah mengubah spesifikasi, jalankan `go generate ./internal/http/apigen/`
  di `apps/api`, lalu commit hasil generate beserta perubahannya.
