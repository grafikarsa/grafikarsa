# Seed Data Awal

Folder ini berisi data master awal Grafikarsa dalam bentuk file SQL. Sumber
kebeneranannya adalah `docs/revamp/seed.md`.

## Tujuan

Mengisi database yang sudah dimigrasi dengan data minimum agar aplikasi bisa
dipakai: jurusan dan grupnya, katalog izin dan role sistem, setting default,
kategori konten, skill dan tools, badge, alasan moderasi, serta username
terlarang.

## Urutan File

File dijalankan berurutan berdasarkan nama:

1. `0001_majors.sql` — jurusan dan grup jurusan.
2. `0002_permissions_roles.sql` — izin, role sistem, dan pemetaan izin role.
3. `0003_settings.sql` — setting default aplikasi.
4. `0004_content_categories.sql` — kategori portfolio/blog dan saran jurusan.
5. `0005_skills_tools.sql` — katalog skill dan tools.
6. `0006_badges.sql` — badge sistem dan badge sekolah.
7. `0007_moderation_reasons.sql` — alasan moderasi standar.
8. `0008_reserved_usernames.sql` — username route dan nama resmi yang dicadangkan.

Urutan ini penting karena beberapa file merujuk baris dari file sebelumnya
(misalnya grup jurusan membutuhkan `majors`, dan pemetaan izin role
membutuhkan `roles` serta `permissions`).

## Idempotensi

Semua file aman dijalankan berulang kali:

- Tabel master memakai `INSERT ... ON CONFLICT (<natural key>) DO UPDATE SET ...`
  sehingga data yang sudah ada diperbarui, bukan diduplikasi.
- Tabel penghubung murni (misalnya `role_permissions` dan
  `content_category_majors`) memakai `ON CONFLICT DO NOTHING`.
- `role_permissions` milik role sistem dibersihkan lebih dulu dari izin yang
  tidak lagi ada di daftar, sehingga hasil akhirnya konvergen.
- UUID tidak pernah ditulis manual; nilai default `uuidv7()` yang membuatnya,
  dan relasi dicari lewat natural key (slug/code/key) menggunakan sub-select.

Setiap file dibungkus `BEGIN; ... COMMIT;`, jadi bila satu pernyataan gagal
seluruh perubahan file tersebut dibatalkan.

## Menjalankan Manual

Dari root repo, dengan `DATABASE_URL` yang sudah di-export:

```sh
for f in apps/api/db/seeds/*.sql; do
  psql "$DATABASE_URL" -v ON_ERROR_STOP=1 -f "$f"
done
```

Bila Postgres berjalan di Docker Compose dev dan `psql` tidak tersedia di host:

```sh
for f in apps/api/db/seeds/*.sql; do
  docker compose --env-file .env -f infra/compose/dev.yml -p grafikarsa-dev \
    exec -T postgres psql -U grafikarsa -d grafikarsa -v ON_ERROR_STOP=1 -q < "$f"
done
```
