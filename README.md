# Grafikarsa

Jejaring sosial, katalog portfolio, dan blogging untuk warga SMKN 4 Malang.

**Status:** sedang dibangun ulang dari nol (revamp). Lihat [docs/revamp/README.md](docs/revamp/README.md) untuk rencana lengkap.

## Mulai cepat (dev)

```bash
make dev-init     # buat .env dengan secret acak
make dev-up       # postgres, valkey, rustfs
make migrate-up   # migrasi goose + River
make seed         # data awal (idempoten)
make api-run      # API di http://localhost:8091
make web-dev      # web di http://localhost:3020 (terminal lain)
```

Jalankan `make help` untuk melihat semua target.

| Layanan | Port dev |
|---|---|
| PostgreSQL | 5439 |
| Valkey | 6391 |
| RustFS S3 | 9020 |
| RustFS console | 9021 |
| API | 8091 |
| Web | 3020 |

Semua port dev di-bind ke `127.0.0.1` (tidak terekspos ke jaringan luar).
