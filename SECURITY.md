# Security Policy

## Reporting a Vulnerability

If you discover a security vulnerability in Grafikarsa, please report it responsibly. Do not open a public issue.

**Contact:** rafapradana.com@gmail.com

When reporting, please include:

1. A description of the vulnerability.
2. Steps to reproduce the issue.
3. The potential impact of the vulnerability.
4. Any suggested fixes, if applicable.

We will acknowledge receipt of your report within 48 hours and will work to address the issue promptly.

## Supported Versions

| Version | Supported |
|---------|-----------|
| Latest (main branch) | Yes |
| Older versions | No |

Only the latest version deployed from the `main` branch is actively supported with security updates.

## Security Practices

### Authentication

- JWT-based authentication with separate access and refresh tokens.
- Access tokens are short-lived (default: 15 minutes).
- Refresh tokens are longer-lived (default: 7 days).
- Passwords are hashed using bcrypt before storage.

### Environment Variables

- All secrets (JWT keys, database passwords, MinIO credentials) are stored in environment variables.
- The `.env` file is excluded from version control via `.gitignore`.
- Production secrets are managed through GitHub Actions Secrets and server-side `.env` files.
- SSH private keys must never be committed to the repository.

### Infrastructure

- Cloudflare provides DDoS protection and SSL termination.
- Nginx acts as a reverse proxy; backend services are not directly exposed to the internet.
- Docker containers run with minimal privileges.
- Database and MinIO ports are not exposed to the public network in production.

### Rate Limiting

- API endpoints are protected by configurable rate limiting.
- Default: 100 requests per minute per IP.

### Data

- File uploads are stored in MinIO with bucket-level access policies.
- Portfolio content is sanitized before storage.
- Database access is restricted to the application service within the Docker network.

## Known Limitations

- The application does not currently implement CSRF token protection for API endpoints (the API is stateless and uses JWT Bearer tokens).
- File upload size limits are enforced at the Nginx level (50MB for API, 100MB for storage).

## Disclosure Policy

We follow a coordinated disclosure policy. If a vulnerability is reported:

1. We will confirm receipt within 48 hours.
2. We will investigate and determine the severity.
3. We will develop and test a fix.
4. We will deploy the fix to production.
5. We will notify the reporter that the issue has been resolved.

We request that reporters refrain from publicly disclosing the vulnerability until a fix has been deployed.

---

# Kebijakan Keamanan

## Melaporkan Kerentanan

Jika Anda menemukan kerentanan keamanan di Grafikarsa, harap laporkan secara bertanggung jawab. Jangan membuka issue publik.

**Kontak:** rafapradana.com@gmail.com

Saat melaporkan, harap sertakan:

1. Deskripsi kerentanan.
2. Langkah-langkah untuk mereproduksi masalah.
3. Dampak potensial dari kerentanan.
4. Saran perbaikan, jika ada.

Kami akan mengkonfirmasi penerimaan laporan Anda dalam 48 jam dan akan bekerja untuk mengatasi masalah tersebut dengan segera.

## Versi yang Didukung

| Versi | Didukung |
|-------|----------|
| Terbaru (branch main) | Ya |
| Versi lama | Tidak |

Hanya versi terbaru yang di-deploy dari branch `main` yang didukung secara aktif dengan pembaruan keamanan.

## Praktik Keamanan

### Autentikasi

- Autentikasi berbasis JWT dengan token akses dan refresh terpisah.
- Token akses berumur pendek (default: 15 menit).
- Token refresh berumur lebih panjang (default: 7 hari).
- Password di-hash menggunakan bcrypt sebelum disimpan.

### Environment Variables

- Semua secrets (kunci JWT, password database, kredensial MinIO) disimpan di environment variables.
- File `.env` dikecualikan dari version control via `.gitignore`.
- Secrets production dikelola melalui GitHub Actions Secrets dan file `.env` di server.
- SSH private key tidak boleh pernah di-commit ke repositori.

### Infrastruktur

- Cloudflare menyediakan perlindungan DDoS dan SSL termination.
- Nginx berfungsi sebagai reverse proxy; layanan backend tidak terekspos langsung ke internet.
- Container Docker berjalan dengan privilege minimal.
- Port database dan MinIO tidak terekspos ke jaringan publik di production.

### Rate Limiting

- Endpoint API dilindungi oleh rate limiting yang dapat dikonfigurasi.
- Default: 100 request per menit per IP.

### Data

- File upload disimpan di MinIO dengan kebijakan akses tingkat bucket.
- Konten portofolio disanitasi sebelum disimpan.
- Akses database dibatasi pada layanan aplikasi dalam jaringan Docker.

## Keterbatasan yang Diketahui

- Aplikasi saat ini tidak mengimplementasikan proteksi CSRF token untuk endpoint API (API bersifat stateless dan menggunakan JWT Bearer token).
- Batas ukuran upload file diterapkan di level Nginx (50MB untuk API, 100MB untuk storage).

## Kebijakan Pengungkapan

Kami mengikuti kebijakan pengungkapan terkoordinasi. Jika kerentanan dilaporkan:

1. Kami akan mengkonfirmasi penerimaan dalam 48 jam.
2. Kami akan menyelidiki dan menentukan tingkat keparahan.
3. Kami akan mengembangkan dan menguji perbaikan.
4. Kami akan men-deploy perbaikan ke production.
5. Kami akan memberitahu pelapor bahwa masalah telah diselesaikan.

Kami meminta pelapor untuk tidak mengungkapkan kerentanan secara publik sampai perbaikan telah di-deploy.
