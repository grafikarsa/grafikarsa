# Grafikarsa

Grafikarsa adalah platform katalog portfolio dan sosial media untuk siswa SMKN 4 Malang.

## Tech Stack

- **Frontend**: Next.js 16 (App Router), Tailwind CSS, Shadcn UI
- **Backend**: Golang (Fiber), GORM
- **Database**: PostgreSQL
- **Storage**: MinIO
- **Infrastructure**: Docker Compose

## Getting Started

1.  **Clone Repository**
    ```bash
    git clone https://github.com/grafikarsa/grafikarsa.git
    cd grafikarsa
    ```

2.  **Environment Setup**
    ```bash
    cp .env.example .env
    ```

3.  **Start Development**
    ```bash
    make dev
    ```
    - Web: http://localhost:3000
    - API: http://localhost:8080
    - MinIO Console: http://localhost:9001
    - DB: localhost:5432

## Development Workflow

- **Migration**: `make migrate`
- **Seed Data**: `make seed`
- **Stop Services**: `make stop`

## Documentation

- [PRD](docs/prd.md)
- [API Specification](docs/api.md)
- [Database Schema](docs/db/01_db.sql)
