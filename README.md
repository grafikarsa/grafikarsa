# Grafikarsa

A portfolio catalog platform for SMKN 4 Malang students and alumni.

Grafikarsa enables students to create, manage, and showcase their work through a structured portfolio system with content blocks, social features, and an admin moderation workflow.

---

## Tech Stack

| Layer | Technology |
|-------|-----------|
| Frontend | Next.js 15 (App Router), React 19, TypeScript |
| Backend | Go (Fiber framework), GORM |
| Database | PostgreSQL 15 |
| Storage | MinIO (S3-compatible object storage) |
| Auth | JWT (access + refresh tokens) |
| CI/CD | GitHub Actions, Docker Hub |
| Infra | Docker Compose, Nginx, Cloudflare |

## Repository Structure

```
grafikarsa/
  apps/
    backend/          Go API server
    web/              Next.js frontend
  db/                 Database schema
  docs/
    dev/              Development guide
    deploy/           Deployment guides (VPS, LXC)
  scripts/            Build, deploy, and test scripts (.sh + .ps1)
  docker-compose.yml          Development (backend services)
  docker-compose.prod.yml     Local production simulation
  docker-compose.deploy.yml   Server deployment (Docker Hub images)
  Makefile                    All workflow commands
  .env.example                Environment variable template
```

## Quick Start

Prerequisites: Docker, Node.js 20+, Make (optional).

```bash
git clone https://github.com/grafikarsa/grafikarsa.git
cd grafikarsa
cp .env.example .env
make dev
```

Then in a separate terminal:

```bash
cd apps/web
npm install
npm run dev
```

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| MinIO Console | http://localhost:9001 |
| Database | localhost:5432 |

For detailed instructions, see [docs/dev/README.md](docs/dev/README.md).

## Available Commands

| Command | Description |
|---------|-------------|
| `make dev` | Start backend services with hot reload |
| `make dev-web` | Start frontend (npm run dev) |
| `make dev-down` | Stop backend services |
| `make prod` | Build and run production simulation locally |
| `make prod-down` | Stop production simulation |
| `make db-import` | Import database schema |
| `make db-reset` | Reset database |
| `make build` | Build production Docker images |
| `make push` | Push images to Docker Hub |
| `make status` | Show container status |
| `make help` | List all commands |

## Deployment

Two deployment guides are available:

- [Ubuntu 24 VPS](docs/deploy/deployment-ubuntu-vps.md) -- Cloudflare + Nginx
- [Ubuntu 22 LXC](docs/deploy/deployment-ubuntu-lxc.md) -- Cloudflare + Nginx + LXC Nesting

## Documentation

| Document | Description |
|----------|-------------|
| [Development Guide](docs/dev/README.md) | Setup, workflow, troubleshooting |
| [Deployment - VPS](docs/deploy/deployment-ubuntu-vps.md) | Ubuntu 24 VPS step-by-step |
| [Deployment - LXC](docs/deploy/deployment-ubuntu-lxc.md) | Ubuntu 22 LXC step-by-step |
| [Changelog](CHANGELOG.md) | Version history |
| [Contributing](CONTRIBUTING.md) | Contribution guidelines |
| [Code of Conduct](CODE_OF_CONDUCT.md) | Community standards |
| [Security](SECURITY.md) | Security policy |

## License

All rights reserved. Copyright (c) 2024-2026 M. Rafa Shaquille Pradana.

This software was developed for SMKN 4 Malang. See [LICENSE](LICENSE) for details.

---

# Grafikarsa

Platform katalog portofolio untuk siswa dan alumni SMKN 4 Malang.

Grafikarsa memungkinkan siswa membuat, mengelola, dan memamerkan karya mereka melalui sistem portofolio terstruktur dengan content blocks, fitur sosial, dan alur moderasi admin.

---

## Tech Stack

| Layer | Teknologi |
|-------|-----------|
| Frontend | Next.js 15 (App Router), React 19, TypeScript |
| Backend | Go (Fiber framework), GORM |
| Database | PostgreSQL 15 |
| Storage | MinIO (S3-compatible object storage) |
| Auth | JWT (access + refresh tokens) |
| CI/CD | GitHub Actions, Docker Hub |
| Infra | Docker Compose, Nginx, Cloudflare |

## Struktur Repositori

```
grafikarsa/
  apps/
    backend/          Server API Go
    web/              Frontend Next.js
  db/                 Skema database
  docs/
    dev/              Panduan development
    deploy/           Panduan deployment (VPS, LXC)
  scripts/            Script build, deploy, dan test (.sh + .ps1)
  docker-compose.yml          Development (backend services)
  docker-compose.prod.yml     Simulasi production lokal
  docker-compose.deploy.yml   Deployment server (image Docker Hub)
  Makefile                    Semua perintah workflow
  .env.example                Template environment variable
```

## Quick Start

Prasyarat: Docker, Node.js 20+, Make (opsional).

```bash
git clone https://github.com/grafikarsa/grafikarsa.git
cd grafikarsa
cp .env.example .env
make dev
```

Kemudian di terminal terpisah:

```bash
cd apps/web
npm install
npm run dev
```

| Service | URL |
|---------|-----|
| Frontend | http://localhost:3000 |
| Backend API | http://localhost:8080 |
| MinIO Console | http://localhost:9001 |
| Database | localhost:5432 |

Untuk instruksi lengkap, lihat [docs/dev/README.md](docs/dev/README.md).

## Perintah Tersedia

| Perintah | Deskripsi |
|----------|-----------|
| `make dev` | Jalankan backend services dengan hot reload |
| `make dev-web` | Jalankan frontend (npm run dev) |
| `make dev-down` | Hentikan backend services |
| `make prod` | Build dan jalankan simulasi production lokal |
| `make prod-down` | Hentikan simulasi production |
| `make db-import` | Import skema database |
| `make db-reset` | Reset database |
| `make build` | Build Docker images production |
| `make push` | Push images ke Docker Hub |
| `make status` | Lihat status container |
| `make help` | Daftar semua perintah |

## Deployment

Tersedia dua panduan deployment:

- [Ubuntu 24 VPS](docs/deploy/deployment-ubuntu-vps.md) -- Cloudflare + Nginx
- [Ubuntu 22 LXC](docs/deploy/deployment-ubuntu-lxc.md) -- Cloudflare + Nginx + LXC Nesting

## Dokumentasi

| Dokumen | Deskripsi |
|---------|-----------|
| [Panduan Development](docs/dev/README.md) | Setup, workflow, troubleshooting |
| [Deployment - VPS](docs/deploy/deployment-ubuntu-vps.md) | Ubuntu 24 VPS step-by-step |
| [Deployment - LXC](docs/deploy/deployment-ubuntu-lxc.md) | Ubuntu 22 LXC step-by-step |
| [Changelog](CHANGELOG.md) | Riwayat versi |
| [Contributing](CONTRIBUTING.md) | Panduan kontribusi |
| [Code of Conduct](CODE_OF_CONDUCT.md) | Standar komunitas |
| [Security](SECURITY.md) | Kebijakan keamanan |

## Lisensi

Hak cipta dilindungi undang-undang. Copyright (c) 2024-2026 M. Rafa Shaquille Pradana.

Software ini dikembangkan untuk SMKN 4 Malang. Lihat [LICENSE](LICENSE) untuk detail.
