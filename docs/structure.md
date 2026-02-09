# Grafikarsa Monorepo Structure

Dokumen ini menjelaskan struktur folder untuk project Grafikarsa menggunakan pendekatan Monorepo. Struktur ini dirancang untuk memudahkan development full-stack (Go + Next.js), sharing assets, dan deployment terintegrasi dengan Docker Compose.

## 1. Root Directory

Root directory berfungsi sebagai entry point untuk seluruh project, tempat konfigurasi global (Docker, Make, Git), dan dokumentasi.

```
grafikarsa/
├── apps/                # Source code aplikasi utama
│   ├── web/             # Frontend: Next.js + Tailwind
│   │   ├── Dockerfile
│   │   ├── Dockerfile.dev
│   │   └── .dockerignore
│   └── api/             # Backend: Go Fiber
│       ├── Dockerfile
│       ├── Dockerfile.dev
│       └── .dockerignore
├── ops/                 # Operations & Infrastructure code
│   ├── k8s/             # Kubernetes manifests (jika nanti perlu scale up)
│   └── seeds/           # SQL / JSON seeds untuk inisial data DB
├── docs/                # Dokumentasi Project
│   ├── PRD.md
│   ├── API_SPEC.md
│   ├── DB_SCHEMA.md
│   └── WEB_ROUTING.md
├── compose.yml          # Docker Compose utama (dev environment)
├── Makefile             # Command runner (dev, build, test, seed)
├── Makefile             # Command runner (dev, build, test, seed)
├── .gitignore
├── .env.example         # Single source of truth for ALL env vars (Web, API, DB)
└── README.md
```

---

## 2. Backend Structure (`apps/api`)

Backend menggunakan **Golang** dengan **Fiber**. Struktur mengikuti "Standard Go Project Layout" yang disederhanakan untuk service API.

```
apps/api/
├── cmd/
│   └── server/
│       └── main.go       # Entry point aplikasi. Load config dari OS Env (injected via Docker)
├── internal/             # Code private aplikasi (business logic)
│   ├── config/           # Struct config mapping (viper/envconfig)
│   ├── db/               # Database connection & migrations
│   ├── domain/           # Entities (structs) & interfaces
│   ├── handler/          # HTTP Handlers (Fiber controllers)
│   ├── middleware/       # JWT Auth, Logger, CORS
│   ├── repository/       # Database interactions (SQL queries)
│   ├── service/          # Business logic implementation
│   └── utils/            # Helper functions (bcyrpt, slug, random)
├── pkg/                  # Library code yang bisa di-import project lain (opsional)
│   └── dto/              # Data Transfer Objects (Request/Response structs)
├── go.mod
├── go.sum
├── Dockerfile            # Production Dockerfile
├── Dockerfile.dev        # Development Dockerfile (Air hot-reload)
└── .dockerignore
```

### Key Components:
- **Handler**: Menerima request HTTP, validasi input body/params, panggil service, return JSON.
- **Service**: Berisi logika bisnis (misal: "User A follow User B", "Draft portfolio versioning").
- **Repository**: Query SQL murni ke PostgreSQL. Jangan taruh logic bisnis di sini.
- **Middleware**: Cek token JWT, Rate Limiting.
- **db**: Logic koneksi database, driver initialization, dan migration scripts.

---

## 3. Frontend Structure (`apps/web`)

Frontend menggunakan **Next.js 16+ (App Router)**. Struktur folder dioptimalkan untuk fitur Grafikarsa (shared layout, auth, dashboard).

```
apps/web/
├── app/                  # Next.js App Router
│   ├── (main)/           # Public & User routes (Sidebar/Navbar conditional layout)
│   │   ├── layout.tsx
│   │   ├── page.tsx      # Landing / Feed
│   │   ├── login/
│   │   ├── register/
│   │   ├── search/
│   │   ├── explore/
│   │   ├── settings/
│   │   ├── portfolios/   # List portfolios generic
│   │   └── [username]/   # User Profile
│   │       ├── page.tsx  # Profile Page
│   │       └── [slug]/   # Portfolio Detail
│   │           └── page.tsx
│   ├── admin/            # Admin routes (Layout terpisah)
│   │   ├── (auth)/       # Login admin
│   │   └── (dashboard)/  # Panel admin
│   │       ├── layout.tsx
│   │       ├── page.tsx  # Dashboard overview
│   │       ├── users/
│   │       ├── portfolios/
│   │       ├── moderation/
│   │       ├── tags/
│   │       ├── majors/
│   │       ├── classes/
│   │       └── academic-years/
│   ├── api/              # Route Handlers (jika butuh proxy API)
│   ├── globals.css
│   └── layout.tsx        # Root layout (Fonts, Providers)
├── components/           # Reusable UI Components
│   ├── ui/               # Base UI (Button, Input, Card - dari Shadcn)
│   ├── shared/           # Components dipakai di banyak tempat (Navbar, Footer)
│   ├── features/         # Components spesifik fitur
│   │   ├── auth/         # LoginForm
│   │   ├── portfolio/    # PortfolioCard, ContentBlocks
│   │   └── admin/        # AdminSidebar, DataTable
│   └── layouts/          # Layout components (Sidebar, Topbar)
├── lib/                  # Utility functions
│   ├── api.ts            # Fetch wrapper / Axios instance ke Backend Go
│   ├── auth.ts           # Auth helpers (session management)
│   └── utils.ts          # CN class merger, formatter
├── hooks/                # Custom React Hooks
├── public/               # Static assets (images, icons)
├── types/                # TypeScript Interfaces (mirror dari Backend DTO)
├── next.config.js
├── tailwind.config.ts
├── package.json
├── tsconfig.json
├── Dockerfile            # Production Dockerfile
├── Dockerfile.dev        # Development Dockerfile
└── .dockerignore
```

---

## 4. Infrastructure & Dev Experience

### Docker Compose (`compose.yml`)

File ini akan menjalankan seluruh stack secara lokal, termasuk dependency services.

```yaml
services:
  # Database Utama
  postgres:
    image: postgres:16-alpine
    ports: ["5432:5432"]
    environment:
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASS}
      POSTGRES_DB: grafikarsa_db
    volumes:
      - pg_data:/var/lib/postgresql/data

  # Object Storage (S3 Compatible)
  minio:
    image: minio/minio
    command: server /data --console-address ":9001"
    ports: ["9000:9000", "9001:9001"]
    environment:
      MINIO_ROOT_USER: ${MINIO_USER}
      MINIO_ROOT_PASSWORD: ${MINIO_PASS}
    volumes:
      - minio_data:/data

  # Backend Go (Hot Reload dengan Air)
  api:
    build: 
      context: ./apps/api
      dockerfile: Dockerfile.dev
    volumes:
      - ./apps/api:/app
    ports: ["8080:8080"]
    depends_on:
      - postgres
      - minio

  # Frontend Next.js (Dev Server)
  web:
    build:
      context: ./apps/web
      dockerfile: Dockerfile.dev
    volumes:
      - ./apps/web:/app
      - /app/node_modules
    ports: ["3000:3000"]
    depends_on:
      - api
    # Inject semua env var dari root .env ke container
    env_file:
      - .env

volumes:
  pg_data:
  minio_data:
```

### Single `.env` Strategy

Kita menggunakan **satu file `.env` di root** (`grafikarsa/.env`) sebagai "Single Source of Truth". File ini di-inject ke semua service (api & web) via Docker Compose saat runtime.

#### 1. Cara Kerja di `compose.yml`

```yaml
services:
  api:
    env_file:
      - .env  # Inject SEMUA variabel di .env ke container API
    environment:
      # Bisa override variabel spesifik jika perlu
      APP_ENV: development
  
  web:
    env_file:
      - .env  # Inject SEMUA variabel di .env ke container Web (Next.js server)
```

#### 2. Aturan & Best Practices

| Service | Aturan Variabel | Cara Akses |
| :--- | :--- | :--- |
| **Backend (Go)** | Bebas (Snake Case). | `os.Getenv("DB_HOST")` |
| **Frontend (Server)** | Bebas (Secret Key, DB Pass). | `process.env.DB_PASS` |
| **Frontend (Client)** | Wajib diawali **`NEXT_PUBLIC_`**. | `process.env.NEXT_PUBLIC_API_URL` |

> **PENTING:** Variabel tanpa prefix `NEXT_PUBLIC_` **TIDAK AKAN** terekspos ke browser (client-side JS). Ini fitur keamanan bawaan Next.js.

#### 3. Contoh Isi File `.env`

```ini
# --- Database Connection (Internal Docker Network) ---
POSTGRES_USER=postgres
POSTGRES_PASSWORD=secret
POSTGRES_DB=grafikarsa_db
DB_HOST=postgres
DB_PORT=5432

# --- MinIO Object Storage ---
MINIO_ROOT_USER=admin
MINIO_ROOT_PASSWORD=password123
MINIO_ENDPOINT=minio:9000
MINIO_PUBLIC_URL=http://localhost:9000

# --- Authentication ---
JWT_SECRET=super-secret-key-change-this-in-production
JWT_EXPIRATION=15m
REFRESH_TOKEN_EXPIRATION=7d

# --- Frontend Public Config (Akses Browser) ---
# URL API yang diakses browser (harus bisa di-resolve dari host machine)
NEXT_PUBLIC_API_URL=http://localhost:8080/api/v1
NEXT_PUBLIC_APP_URL=http://localhost:3000
```

### Makefile

Untuk menyederhanakan command panjang.

```makefile
dev:
	docker compose up

stop:
	docker compose down

seed:
	docker compose exec api go run cmd/seed/main.go

migrate:
	docker compose exec api go run cmd/migrate/main.go
```

---

## 5. Development Workflow

1.  **Clone Repo**: `git clone grafikarsa`
2.  **Setup Env**: Copy `.env.example` ke `.env`
3.  **Start Services**: `make dev` (jalan semua DB, API, Web, MinIO)
4.  **Database Migration**: Dilakukan via `make migrate` atau otomatis saat startup (tergantung strategi yang dipilih).
5.  **Seed Data**: `make seed` (Isi user admin, jurusan, dll).