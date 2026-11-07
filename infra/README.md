# Infra produksi

Direktori ini berisi konfigurasi untuk menjalankan Grafikarsa di server produksi
dengan Docker Compose. Hanya Caddy yang membuka port ke publik; seluruh layanan
lain hanya dapat diakses dari jaringan internal Compose.

## Berkas

- `compose/prod.yml` — definisi stack produksi: `caddy`, `web`, `api`,
  `postgres`, `valkey`, dan `rustfs`. Berkas ini membaca variabel yang sama
  dengan `.env` dev (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB`,
  `VALKEY_PASSWORD`, `RUSTFS_ACCESS_KEY`, `RUSTFS_SECRET_KEY`,
  `GK_SECRET_KEY`) ditambah `GK_DOMAIN` (default `localhost`), `GK_ENV`
  (default `production`), `HTTP_PORT` (default `80`), dan `HTTPS_PORT`
  (default `443`).
- `caddy/Caddyfile` — reverse proxy dan terminasi TLS. Caddy meneruskan
  `/api/*` ke `api:8091`, `/media/*` (GET/HEAD) ke bucket publik RustFS,
  memblokir `/internal/*`, dan sisanya ke `web:3000`.
- `../apps/web/Dockerfile` — build web Next.js (output standalone) dengan
  build context root repo.
- `../.dockerignore` — mengecualikan `.git`, `node_modules`, `.next`, `.env`,
  log, dan `docs` dari build context.

## Port yang terbuka

Hanya layanan `caddy` yang mempublikasikan port: `HTTP_PORT` (80), `HTTPS_PORT`
(443/TCP), dan `HTTPS_PORT/udp` (443/UDP untuk HTTP/3). `web`, `api`,
`postgres`, `valkey`, dan `rustfs` tidak memiliki `ports` sehingga hanya bisa
diakses lewat jaringan internal Compose. RustFS console juga tidak diekspos.

## Simulasi produksi lokal

Jalankan dari root repo. Pastikan `.env` sudah ada (`make dev-init`). Gunakan
port 8080/8443 agar tidak bentrok dengan port 80/443. `GK_ENV=staging` membuat
API otomatis membuat bucket yang dibutuhkan.

```bash
GK_ENV=staging HTTP_PORT=8080 HTTPS_PORT=8443 \
  docker compose --env-file .env -f infra/compose/prod.yml -p gk-prodtest \
  up -d --build --wait
```

Cek layanan:

```bash
curl -sk -o /dev/null -w '%{http_code}\n' https://localhost:8443/
curl -sk https://localhost:8443/api/v1/ping
curl -sk -o /dev/null -w '%{http_code}\n' https://localhost:8443/internal/x
curl -sk -o /dev/null -w '%{http_code}\n' -X POST https://localhost:8443/media/x
curl -skI https://localhost:8443/ | grep -iE 'strict-transport|x-content-type'
docker compose --env-file .env -f infra/compose/prod.yml -p gk-prodtest \
  ps --format '{{.Name}} {{.Ports}}'
```

Hentikan dan hapus volume uji:

```bash
docker compose --env-file .env -f infra/compose/prod.yml -p gk-prodtest down -v
```

## Catatan TODO

- Bucket `gk-public` harus diberi policy anonymous-read sebelum `/media/*`
  berfungsi. Saat ini bucket dibuat oleh API pada `GK_ENV=staging`/development,
  tetapi policy baca publik belum dipasang.
- Migrasi database harus dijalankan sebelum API mulai melayani request.
  Command `migrate` akan ditambahkan; sementara ini migrasi dijalankan manual.
- Untuk produksi nyata, gunakan `GK_DOMAIN` yang mengarah ke server dan biarkan
  Caddy menerbitkan sertifikat ACME. Email ACME memakai default
  `admin@localhost`; ganti di `infra/caddy/Caddyfile` atau tambahkan env
  `GK_ACME_EMAIL` pada service `caddy` bila diperlukan. Jangan memakai `.env`
  dev berisi secret contoh untuk server produksi.
