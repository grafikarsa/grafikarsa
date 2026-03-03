# Contributing to Grafikarsa

This document provides guidelines for internal development team members contributing to the Grafikarsa project.

Grafikarsa is a proprietary project. Contributions are limited to authorized team members at SMKN 4 Malang.

---

## Table of Contents

- [Onboarding](#onboarding)
- [Development Setup](#development-setup)
- [Branch Strategy](#branch-strategy)
- [Commit Conventions](#commit-conventions)
- [Pull Request Process](#pull-request-process)
- [Code Standards](#code-standards)
- [Testing](#testing)
- [Project Architecture](#project-architecture)

---

## Onboarding

### Step 1: Obtain Access

1. Request repository access from the project maintainer.
2. Set up a Docker Hub account and share your username with the team.
3. Obtain the `.env` file with development credentials from a team member.

### Step 2: Clone and Configure

```bash
git clone https://github.com/grafikarsa/grafikarsa.git
cd grafikarsa
cp .env.example .env
```

Edit `.env` with the values provided by your team. The default development values in `.env.example` will work for local development out of the box.

### Step 3: Start Development Environment

```bash
make dev
```

In a separate terminal:

```bash
cd apps/web
npm install
npm run dev
```

Verify all services are running:

| Service | URL | What to Expect |
|---------|-----|----------------|
| Frontend | http://localhost:3000 | Landing page |
| Backend API | http://localhost:8080/api/v1/health | Health response |
| MinIO Console | http://localhost:9001 | Login page |
| Database | localhost:5432 | Accessible via psql |

### Step 4: Familiarize Yourself with the Codebase

Read the following documents in order:

1. [README.md](README.md) -- Project overview
2. [docs/dev/README.md](docs/dev/README.md) -- Development workflow
3. [docs/api/](docs/api/) -- API documentation (if available)
4. [CHANGELOG.md](CHANGELOG.md) -- Recent changes

Review the project structure:

- `apps/backend/` -- Go API. Entry point: `cmd/api/main.go`. Business logic: `internal/`.
- `apps/web/` -- Next.js frontend. Pages: `app/`. Components: `components/`. API client: `lib/api/`.
- `db/db.sql` -- Database schema.

---

## Development Setup

### Prerequisites

| Tool | Version | Installation |
|------|---------|-------------|
| Docker Desktop | 4+ | docker.com |
| Node.js | 20+ | nodejs.org |
| Git | 2+ | git-scm.com |
| Make | any | `choco install make` (Windows) or pre-installed (macOS/Linux) |

### How It Works

- **Backend**: runs inside Docker with Air hot reload. Edit Go files and the server restarts automatically.
- **Frontend**: runs natively via `npm run dev`. Edit React/Next.js files and changes reflect instantly.
- **Database and MinIO**: run inside Docker. Data persists across restarts via named volumes.

### Common Operations

| Task | Command |
|------|---------|
| Start backend services | `make dev` |
| Start frontend | `cd apps/web && npm run dev` |
| Stop everything | `make dev-down` |
| View logs | `make dev-logs` |
| Reset database | `make db-reset` |
| Open database shell | `make db-shell` |
| Run backend tests | `make test-backend` |
| Test production build | `make prod` |

---

## Branch Strategy

```
main              Production branch. Deploys automatically via CI/CD.
  |
  +-- dev         Integration branch. All feature branches merge here first.
      |
      +-- feature/description    New features
      +-- fix/description        Bug fixes
      +-- refactor/description   Refactoring
      +-- docs/description       Documentation changes
```

**Rules:**

1. Never push directly to `main`. Always go through `dev` first.
2. Create feature branches from `dev`.
3. Keep branches short-lived. Merge within a few days.
4. Delete branches after merging.

---

## Commit Conventions

Use [Conventional Commits](https://www.conventionalcommits.org/).

```
<type>: <description>

[optional body]
```

### Types

| Type | Use When |
|------|----------|
| `feat` | Adding a new feature |
| `fix` | Fixing a bug |
| `refactor` | Changing code without adding features or fixing bugs |
| `docs` | Documentation changes only |
| `style` | Code formatting, missing semicolons, etc. |
| `test` | Adding or updating tests |
| `chore` | Build process, CI, dependencies, tooling |
| `perf` | Performance improvements |

### Examples

```
feat: add portfolio image compression before upload
fix: resolve JWT refresh token expiration handling
refactor: extract portfolio validation into separate middleware
docs: update deployment guide with SSL configuration
chore: upgrade Go dependencies to latest versions
```

---

## Pull Request Process

1. **Create a branch** from `dev` with a descriptive name.
2. **Make your changes** following the code standards below.
3. **Test locally** -- ensure `make dev` and `make prod` both work.
4. **Commit** using conventional commit messages.
5. **Push** your branch and open a Pull Request targeting `dev`.
6. **Describe your changes** in the PR description. Include:
   - What was changed and why.
   - How to test the changes.
   - Screenshots for UI changes.
7. **Request review** from at least one team member.
8. **Address feedback** if any.
9. **Merge** after approval. Use "Squash and merge" for feature branches.

---

## Code Standards

### Backend (Go)

- Follow standard Go conventions and `gofmt` formatting.
- Use meaningful variable and function names.
- Keep functions focused -- one function, one responsibility.
- Handle errors explicitly. Do not ignore returned errors.
- Add comments for exported functions and non-obvious logic.
- Use the existing patterns in `internal/` as reference for new code.

### Frontend (Next.js / TypeScript)

- Use TypeScript for all new files (`.ts`, `.tsx`).
- Follow the existing component structure in `components/`.
- Use the API client in `lib/api/` for backend communication.
- Keep components small and focused.
- Use server components by default. Add `"use client"` only when necessary.
- Run `npm run lint` before committing.

### General

- Do not commit `.env` files, secrets, or credentials.
- Do not commit `node_modules/`, `bin/`, or build artifacts.
- Write descriptive commit messages.
- Keep related changes in a single commit.

---

## Testing

### Backend

```bash
# Run all Go tests
make test-backend

# With coverage
make test-backend-cover

# Run the API test suite against a running server
./scripts/api_test.sh          # Bash
.\scripts\api_test.ps1         # PowerShell
```

### Frontend

```bash
cd apps/web
npm run lint
npm run build    # Ensures no build errors
```

### Production Simulation

Before submitting a PR that changes Docker configuration or build process:

```bash
make prod
# Verify http://localhost:3000 and http://localhost:8080 work correctly
make prod-down
```

---

## Project Architecture

### Backend

```
apps/backend/
  cmd/api/              Application entry point
  internal/
    config/             Environment variable loading
    database/           Database connection and migrations
    handlers/           HTTP request handlers (controllers)
    middleware/         Auth, CORS, rate limiting middleware
    models/             GORM database models
    routes/             Route definitions
    services/           Business logic layer
    utils/              Shared utility functions
```

Request flow: `Route -> Middleware -> Handler -> Service -> Model -> Database`

### Frontend

```
apps/web/
  app/                  Next.js App Router pages
    (user)/             User-facing pages (portfolio, profile)
    admin/              Admin dashboard pages
  components/           Reusable React components
    ui/                 Base UI components
  lib/
    api/                API client (server-fetch, client)
    hooks/              Custom React hooks
    stores/             State management (Zustand)
    types/              TypeScript type definitions
    utils/              Utility functions
```

---

# Kontribusi untuk Grafikarsa

Dokumen ini menyediakan panduan untuk anggota tim pengembang internal yang berkontribusi pada proyek Grafikarsa.

Grafikarsa adalah proyek proprietary. Kontribusi terbatas pada anggota tim yang berwenang di SMKN 4 Malang.

---

## Daftar Isi

- [Onboarding](#onboarding-1)
- [Setup Development](#setup-development)
- [Strategi Branch](#strategi-branch)
- [Konvensi Commit](#konvensi-commit)
- [Proses Pull Request](#proses-pull-request)
- [Standar Kode](#standar-kode)
- [Testing](#testing-1)
- [Arsitektur Proyek](#arsitektur-proyek)

---

## Onboarding

### Langkah 1: Dapatkan Akses

1. Minta akses repositori dari project maintainer.
2. Buat akun Docker Hub dan bagikan username ke tim.
3. Dapatkan file `.env` dengan kredensial development dari anggota tim.

### Langkah 2: Clone dan Konfigurasi

```bash
git clone https://github.com/grafikarsa/grafikarsa.git
cd grafikarsa
cp .env.example .env
```

Edit `.env` dengan nilai yang diberikan tim. Nilai default di `.env.example` sudah bisa langsung dipakai untuk development lokal.

### Langkah 3: Jalankan Environment Development

```bash
make dev
```

Di terminal terpisah:

```bash
cd apps/web
npm install
npm run dev
```

Verifikasi semua service berjalan:

| Service | URL | Yang Diharapkan |
|---------|-----|-----------------|
| Frontend | http://localhost:3000 | Landing page |
| Backend API | http://localhost:8080/api/v1/health | Health response |
| MinIO Console | http://localhost:9001 | Halaman login |
| Database | localhost:5432 | Bisa diakses via psql |

### Langkah 4: Pahami Codebase

Baca dokumen berikut secara berurutan:

1. [README.md](README.md) -- Gambaran proyek
2. [docs/dev/README.md](docs/dev/README.md) -- Workflow development
3. [docs/api/](docs/api/) -- Dokumentasi API (jika tersedia)
4. [CHANGELOG.md](CHANGELOG.md) -- Perubahan terbaru

Pelajari struktur proyek:

- `apps/backend/` -- Go API. Entry point: `cmd/api/main.go`. Business logic: `internal/`.
- `apps/web/` -- Next.js frontend. Pages: `app/`. Components: `components/`. API client: `lib/api/`.
- `db/db.sql` -- Skema database.

---

## Setup Development

### Prasyarat

| Tool | Versi | Instalasi |
|------|-------|-----------|
| Docker Desktop | 4+ | docker.com |
| Node.js | 20+ | nodejs.org |
| Git | 2+ | git-scm.com |
| Make | any | `choco install make` (Windows) atau sudah terinstall (macOS/Linux) |

### Cara Kerja

- **Backend**: berjalan di dalam Docker dengan Air hot reload. Edit file Go dan server restart otomatis.
- **Frontend**: berjalan native via `npm run dev`. Edit file React/Next.js dan perubahan langsung terlihat.
- **Database dan MinIO**: berjalan di dalam Docker. Data tetap tersimpan antar restart via named volumes.

### Operasi Umum

| Tugas | Perintah |
|-------|----------|
| Start backend services | `make dev` |
| Start frontend | `cd apps/web && npm run dev` |
| Stop semuanya | `make dev-down` |
| Lihat logs | `make dev-logs` |
| Reset database | `make db-reset` |
| Buka database shell | `make db-shell` |
| Jalankan backend tests | `make test-backend` |
| Test production build | `make prod` |

---

## Strategi Branch

```
main              Branch production. Deploy otomatis via CI/CD.
  |
  +-- dev         Branch integrasi. Semua feature branch merge ke sini dulu.
      |
      +-- feature/deskripsi    Fitur baru
      +-- fix/deskripsi        Perbaikan bug
      +-- refactor/deskripsi   Refactoring
      +-- docs/deskripsi       Perubahan dokumentasi
```

**Aturan:**

1. Jangan pernah push langsung ke `main`. Selalu melalui `dev` dulu.
2. Buat feature branch dari `dev`.
3. Jaga branch tetap singkat. Merge dalam beberapa hari.
4. Hapus branch setelah merge.

---

## Konvensi Commit

Gunakan [Conventional Commits](https://www.conventionalcommits.org/).

```
<tipe>: <deskripsi>

[body opsional]
```

### Tipe

| Tipe | Digunakan Ketika |
|------|-----------------|
| `feat` | Menambahkan fitur baru |
| `fix` | Memperbaiki bug |
| `refactor` | Mengubah kode tanpa menambah fitur atau memperbaiki bug |
| `docs` | Perubahan dokumentasi saja |
| `style` | Formatting kode, semicolon yang hilang, dll. |
| `test` | Menambah atau memperbarui test |
| `chore` | Build process, CI, dependencies, tooling |
| `perf` | Peningkatan performa |

### Contoh

```
feat: tambah kompresi gambar portofolio sebelum upload
fix: perbaiki penanganan expirasi JWT refresh token
refactor: ekstrak validasi portofolio ke middleware terpisah
docs: perbarui panduan deployment dengan konfigurasi SSL
chore: upgrade Go dependencies ke versi terbaru
```

---

## Proses Pull Request

1. **Buat branch** dari `dev` dengan nama deskriptif.
2. **Buat perubahan** mengikuti standar kode di bawah.
3. **Test secara lokal** -- pastikan `make dev` dan `make prod` keduanya berjalan.
4. **Commit** menggunakan pesan commit konvensional.
5. **Push** branch dan buka Pull Request yang menargetkan `dev`.
6. **Jelaskan perubahan** di deskripsi PR. Sertakan:
   - Apa yang diubah dan mengapa.
   - Cara menguji perubahan.
   - Screenshot untuk perubahan UI.
7. **Minta review** dari minimal satu anggota tim.
8. **Tanggapi feedback** jika ada.
9. **Merge** setelah disetujui. Gunakan "Squash and merge" untuk feature branches.

---

## Standar Kode

### Backend (Go)

- Ikuti konvensi Go standar dan formatting `gofmt`.
- Gunakan nama variabel dan fungsi yang bermakna.
- Jaga fungsi tetap fokus -- satu fungsi, satu tanggung jawab.
- Tangani error secara eksplisit. Jangan abaikan error yang dikembalikan.
- Tambahkan komentar untuk fungsi yang diekspor dan logika yang tidak jelas.
- Gunakan pola yang sudah ada di `internal/` sebagai referensi untuk kode baru.

### Frontend (Next.js / TypeScript)

- Gunakan TypeScript untuk semua file baru (`.ts`, `.tsx`).
- Ikuti struktur komponen yang sudah ada di `components/`.
- Gunakan API client di `lib/api/` untuk komunikasi dengan backend.
- Jaga komponen tetap kecil dan fokus.
- Gunakan server components secara default. Tambahkan `"use client"` hanya jika perlu.
- Jalankan `npm run lint` sebelum commit.

### Umum

- Jangan commit file `.env`, secrets, atau kredensial.
- Jangan commit `node_modules/`, `bin/`, atau build artifacts.
- Tulis pesan commit yang deskriptif.
- Simpan perubahan yang berkaitan dalam satu commit.

---

## Testing

### Backend

```bash
# Jalankan semua Go tests
make test-backend

# Dengan coverage
make test-backend-cover

# Jalankan API test suite terhadap server yang berjalan
./scripts/api_test.sh          # Bash
.\scripts\api_test.ps1         # PowerShell
```

### Frontend

```bash
cd apps/web
npm run lint
npm run build    # Memastikan tidak ada build error
```

### Simulasi Production

Sebelum submit PR yang mengubah konfigurasi Docker atau build process:

```bash
make prod
# Verifikasi http://localhost:3000 dan http://localhost:8080 berjalan
make prod-down
```

---

## Arsitektur Proyek

### Backend

```
apps/backend/
  cmd/api/              Entry point aplikasi
  internal/
    config/             Loading environment variable
    database/           Koneksi database dan migrasi
    handlers/           HTTP request handlers (controllers)
    middleware/         Auth, CORS, rate limiting middleware
    models/             GORM database models
    routes/             Definisi route
    services/           Layer business logic
    utils/              Fungsi utilitas bersama
```

Alur request: `Route -> Middleware -> Handler -> Service -> Model -> Database`

### Frontend

```
apps/web/
  app/                  Next.js App Router pages
    (user)/             Halaman user (portofolio, profil)
    admin/              Halaman admin dashboard
  components/           Komponen React yang dapat digunakan kembali
    ui/                 Base UI components
  lib/
    api/                API client (server-fetch, client)
    hooks/              Custom React hooks
    stores/             State management (Zustand)
    types/              TypeScript type definitions
    utils/              Fungsi utilitas
```
