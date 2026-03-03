# 🚀 Panduan Development — Grafikarsa

Panduan lengkap untuk memulai development Grafikarsa di mesin lokal.

---

## 📋 Prerequisites

| Tool | Versi Minimum | Cek |
|------|--------------|-----|
| **Docker Desktop** | 4.x | `docker --version` |
| **Docker Compose** | v2 | `docker compose version` |
| **Node.js** | 20.x | `node --version` |
| **npm** | 10.x | `npm --version` |
| **Git** | 2.x | `git --version` |
| **Make** (opsional) | any | `make --version` |

> **Windows**: Install Docker Desktop. Make bisa melalui `choco install make` atau gunakan WSL2.
>
> **Mac**: `brew install docker node git make`

---

## ⚡ Quick Start (5 menit)

```bash
# 1. Clone repo
git clone https://github.com/grafikarsa/grafikarsa.git
cd grafikarsa

# 2. Copy environment file
cp .env.example .env

# 3. Start backend services (db + minio + backend with hot reload)
make dev

# 4. Di terminal BARU, start web frontend
cd apps/web
npm install
npm run dev
```

**Selesai!** Buka:
- 🌐 **Frontend**: http://localhost:3000
- 🔌 **Backend API**: http://localhost:8080
- 📦 **MinIO Console**: http://localhost:9001 (user: `minioadmin`, pass: `minioadmin123`)
- 🗄️ **Database**: `localhost:5432` (user: `grafikarsa`, pass: `grafikarsa123`)

---

## 🏗️ Arsitektur

```
┌─────────────────────────────────────────────────────────────┐
│  Mesin Lokal                                                │
│                                                             │
│  ┌─────────────────┐          Docker Network                │
│  │   Next.js Web   │    ┌──────────────────────────┐       │
│  │   (npm run dev) │    │                          │       │
│  │   localhost:3000 │───▶│  Go Backend (Air)       │       │
│  └─────────────────┘    │  localhost:8080          │       │
│                          │         │      │         │       │
│                          │         ▼      ▼         │       │
│                          │  ┌──────┐  ┌───────┐    │       │
│                          │  │  DB  │  │ MinIO │    │       │
│                          │  │:5432 │  │ :9000 │    │       │
│                          │  └──────┘  └───────┘    │       │
│                          └──────────────────────────┘       │
└─────────────────────────────────────────────────────────────┘
```

**Kenapa web di luar Docker?**
- Hot reload lebih cepat (tanpa volume mount overhead)
- Bisa pakai fitur IDE/debugger secara penuh
- `npm install` lebih cepat tanpa Docker layer

---

## 📁 Struktur Project

```
grafikarsa/
├── apps/
│   ├── backend/              # Go API (Fiber + GORM)
│   │   ├── cmd/api/          # Entry point
│   │   ├── internal/         # Business logic
│   │   ├── Dockerfile        # Multi-stage (dev/prod)
│   │   ├── .air.toml         # Hot reload config
│   │   ├── go.mod
│   │   └── go.sum
│   │
│   └── web/                  # Next.js frontend
│       ├── app/              # App router (pages)
│       ├── components/       # React components
│       ├── lib/              # API client & utilities
│       ├── Dockerfile        # Multi-stage (dev/prod)
│       ├── package.json
│       └── next.config.ts
│
├── db/
│   └── db.sql                # Database schema
│
├── docs/                     # Documentation
│   ├── dev/                  # Development guide (this file)
│   └── deploy/               # Deployment guides
│
├── scripts/                  # Helper scripts
│
├── docker-compose.yml        # Dev: backend services only
├── docker-compose.prod.yml   # Local prod simulation
├── docker-compose.deploy.yml # Server deployment (pulled images)
├── Makefile                  # All commands
├── .env.example              # Environment template
└── .env                      # Your local config (gitignored)
```

---

## 🔧 Environment Variables

Semua environment variables ada di **satu file `.env` di root**. File ini digunakan oleh:

| Consumer | Bagaimana |
|----------|-----------|
| Docker Compose | `env_file: .env` |
| Go Backend (di Docker) | Docker injects dari `env_file` + override `DB_HOST=db` |
| Go Backend (lokal) | `godotenv.Load()` baca `.env` otomatis |
| Next.js web | Baca `NEXT_PUBLIC_*` dari `.env` via Next.js |
| Makefile | `-include .env` + `export` |

**Penting**: Di `.env`, `DB_HOST=localhost` (supaya web bisa akses DB kalau perlu). Docker Compose override jadi `DB_HOST=db` di dalam container backend.

---

## 📝 Perintah Make

| Command | Fungsi |
|---------|--------|
| `make dev` | Start backend services (db + minio + backend hot reload) |
| `make dev-web` | Start web frontend (`npm run dev`) |
| `make dev-down` | Stop backend services |
| `make dev-logs` | Lihat logs backend services |
| `make prod` | Simulasi production lokal (semua di Docker) |
| `make prod-down` | Stop simulasi production |
| `make db-import` | Import schema dari `db/db.sql` |
| `make db-reset` | Reset database (hapus data, import ulang) |
| `make db-shell` | Buka psql shell |
| `make db-backup` | Backup database ke file `.sql` |
| `make test-backend` | Jalankan Go tests |
| `make build` | Build production Docker images |
| `make push` | Push images ke Docker Hub |
| `make clean` | Hapus semua Docker resources |
| `make status` | Lihat status containers |
| `make help` | Lihat semua commands |

---

## 🔄 Workflow Development

### Backend (Go)

Backend menggunakan **Air** untuk hot reload. Setiap kali kamu save file `.go`, Air akan otomatis rebuild dan restart server.

```bash
# Lihat logs backend
make logs-backend

# Masuk ke container backend
make shell-backend

# Jalankan test
make test-backend

# Restart backend saja
make restart-backend
```

**Menambahkan dependency:**
```bash
# Masuk container dulu
docker exec -it grafikarsa-backend-dev sh

# Di dalam container:
go get github.com/some/package
go mod tidy
exit

# Rebuild container
docker compose up -d --build backend
```

### Frontend (Next.js)

Frontend berjalan native di mesinmu (bukan Docker), jadi hot reload instan.

```bash
# Di terminal terpisah
cd apps/web
npm run dev

# Install package baru
npm install some-package

# Lint
npm run lint

# Build production (test)
npm run build
```

### Database

```bash
# Buka shell psql
make db-shell

# Di dalam psql:
\dt              -- list tables
\d users         -- describe table
SELECT * FROM users LIMIT 5;
\q               -- quit

# Import schema
make db-import

# Reset database (hati-hati, hapus data!)
make db-reset

# Backup
make db-backup
```

### MinIO (File Storage)

1. Buka http://localhost:9001
2. Login: `minioadmin` / `minioadmin123`
3. Buat bucket `grafikarsa`
4. Set policy ke **public** untuk read access

Atau via CLI:
```bash
docker exec -it grafikarsa-minio-dev sh
mc alias set local http://localhost:9000 minioadmin minioadmin123
mc mb local/grafikarsa
mc anonymous set download local/grafikarsa
```

---

## 🧪 Testing Production Locally

Untuk test apakah build production berhasil:

```bash
# Build dan jalankan semua service sebagai production
make prod

# Cek: http://localhost:3000 dan http://localhost:8080

# Stop
make prod-down
```

Ini akan build Docker images production dan jalankan semuanya persis seperti di server.

---

## 🐛 Troubleshooting

### Port sudah dipakai

```bash
# Windows:
netstat -ano | findstr :3000
taskkill /PID <PID> /F

# Linux/Mac:
lsof -i :3000
kill -9 <PID>
```

### Container tidak mau start

```bash
# Cek logs
docker compose logs

# Cek container specific
docker logs grafikarsa-backend-dev

# Rebuild dari awal
docker compose down
docker compose up -d --build
```

### Database connection error

```bash
# Pastikan DB container running
docker ps | grep postgres

# Test koneksi
docker exec -it grafikarsa-db-dev psql -U grafikarsa -d grafikarsa

# Cek env di backend
docker exec grafikarsa-backend-dev env | grep DB_
```

### Hot reload tidak jalan (backend)

```bash
# Cek Air running
docker logs grafikarsa-backend-dev | grep "watching"

# Restart
docker compose restart backend
```

### npm run dev error

```bash
# Hapus node_modules dan install ulang
cd apps/web
rm -rf node_modules .next
npm install
npm run dev
```

---

## 🤝 Git Workflow

```bash
# Buat feature branch
git checkout -b feature/nama-fitur

# Commit (gunakan conventional commits)
git commit -m "feat: tambah fitur X"
git commit -m "fix: perbaiki bug Y"
git commit -m "docs: update panduan"

# Push
git push origin feature/nama-fitur

# Buat Pull Request di GitHub
```

---

## 💡 Tips

- Gunakan `make dev-logs` untuk monitor semua service sekaligus
- Jangan commit file `.env` — sudah di-gitignore
- Test production build dengan `make prod` sebelum push ke main
- Backend hot reload otomatis, tapi kalau ada perubahan `go.mod` harus rebuild container
- Selalu `make db-backup` sebelum `make db-reset`
