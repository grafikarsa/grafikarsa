# AI Agent Instructions

This document provides comprehensive instructions for AI coding agents working on the Grafikarsa project.

---

## 📋 Project Context

**Project Name:** Grafikarsa  
**Description:** Platform Katalog Portofolio & Social Network untuk Warga SMKN 4 Malang

### Purpose
Grafikarsa adalah platform showcase portfolio dan social networking yang dirancang khusus untuk siswa dan alumni SMKN 4 Malang. Platform ini memungkinkan siswa untuk:
- Menampilkan karya kreatif mereka dalam bentuk portfolio
- Membangun jaringan dengan sesama siswa dan alumni
- Mendapatkan feedback dan apresiasi dari komunitas
- Mengeksplorasi karya inspiratif dari teman-teman mereka

### Tech Stack

**Frontend:**
- Next.js 16 (App Router)
- React 19.2
- TypeScript 5
- Tailwind CSS v4
- shadcn/ui components (Radix UI primitives)
- Framer Motion / Motion (animations)
- React Query v5 (data fetching)
- Zustand v5 (state management)
- Zod v4 (schema validation)
- React Hook Form v7 (form handling)
- Axios (HTTP client)
- Recharts (charts)
- @dnd-kit (drag-and-drop)
- @react-pdf/renderer + pdf-lib (PDF export)
- Geist font (Sans + Mono)

**Backend:**
- Go 1.24 (Fiber v2 framework)
- GORM (ORM)
- PostgreSQL 15 (database)
- MinIO (S3-compatible object storage)
- Redis 7 (caching, rate limiting)
- Google Generative AI (Gemini for AI features)
- Excelize (Excel import/export)

**Deployment:**
- Docker + Docker Compose
- VPS (Ubuntu 24) or LXC (Proxmox)
- Nginx reverse proxy
- Cloudflare CDN + SSL
- GitHub Actions CI/CD

### Design System

**Colors:**
- Primary: Blue (#3B82F6)
- Using CSS variables with OKLCH color space for theming (light/dark mode)
- Semantic color tokens: `bg-primary`, `text-foreground`, `bg-muted`, etc.

**Typography:**
- Font: Geist Sans
- Monospace: Geist Mono

**Icons:**
- Lucide React (NEVER use emojis as UI icons)

**UI Library:**
- shadcn/ui components
- Consistent border radius: `--radius: 0.625rem`
- Custom breakpoint: `--breakpoint-xs: 475px`

**Image Standards:**
- Avatar: 1:1 ratio (800x800px recommended)
- Banner: 3:1 ratio (1500x500px recommended)
- Portfolio Thumbnail: 4:3 ratio (1200x900px recommended)
- See `docs/ui/image-standards.md` for details

### Key Features

1. **Portfolio Management**
   - Create, edit, delete portfolios
   - Modular content blocks (text, image, video, links, tables, embed, figma, canva, ppt, pdf, doc)
   - Draft, review, and publish workflow (draft → pending_review → published/rejected → archived)
   - Series templates for structured portfolios
   - Drag-and-drop block reordering
   - PDF export with QR codes

2. **Social Features**
   - Follow/unfollow users
   - Like and comment on portfolios (threaded comments)
   - Activity feed (smart algorithm, recent, following)
   - User profiles with stats (follower/following count, portfolio count)
   - Notifications (7 types: new_follower, portfolio_liked/approved/rejected, new/reply comment, feedback_updated)

3. **Admin Dashboard**
   - User management (CRUD, activate/deactivate, password reset)
   - Portfolio moderation (approve/reject with notes)
   - Tag management (CRUD)
   - Series template management (with PDF export)
   - Special roles with capability-based access control
   - Academic year management
   - Major (jurusan) management
   - Class (kelas) management
   - Assessment system (configurable metrics, scoring)
   - Feedback management (bug reports, suggestions)
   - Changelog management
   - Student import (Excel upload with dry-run)
   - Dashboard with statistics

4. **Discovery**
   - Browse portfolios by tags, class, major
   - Search users and portfolios (trigram fuzzy search)
   - Explore feed with smart algorithm (interest-based + engagement normalization)

5. **AI Features**
   - AI project idea generator (Google Gemini)
   - Interest-based recommendations

### User Roles

- **Guest:** Can browse public portfolios and user profiles
- **Student/Alumni:** Can create portfolios, follow users, interact with content
- **Admin:** Full access to moderation and management features
- **Special Roles:** Capability-based access (e.g., "Moderator Konten" with specific admin permissions)

---

## 🛠️ Agent Skills Reference

This project uses specialized agent skills for different types of tasks. Each skill provides focused expertise and best practices for its domain.

### Skills Overview Table

| Skill | Location | Use When | Key Features |
|-------|----------|----------|--------------|
| **context7** | `.agents/skills/context7/` | Working with external libraries or need up-to-date documentation | Fetch current library docs, verify API signatures, check latest versions, library best practices |
| **golang-pro** | `.agents/skills/golang-pro/` | Writing or modifying Go backend code | Go best practices, Fiber framework patterns, database operations, API design |
| **git-commit** | `.agents/skills/git-commit/` | Creating git commits | Conventional commit analysis, intelligent staging, message generation |

---

## 📚 Detailed Skills Guide

### 1. Context7 Documentation Fetcher

**Location:** `.agents/skills/context7/SKILL.md`

**Use this skill when:**
- Working with ANY external library (React, Next.js, Tailwind, etc.)
- Installing new dependencies or frameworks
- User asks about library APIs, patterns, or best practices
- Implementing features that rely on third-party packages
- Debugging library-specific issues
- Need current documentation beyond training data cutoff
- **ESPECIALLY when:** Installing dependencies - ALWAYS check docs for latest versions

**CRITICAL:** Use this PROACTIVELY. Don't guess library APIs or rely on outdated knowledge.

**Workflow:**

1. **Search for library:**
```bash
py ~/.agents/skills/context7/scripts/context7.py search "<library-name>"
```

2. **Fetch documentation:**
```bash
py ~/.agents/skills/context7/scripts/context7.py context "<library-id>" "<query>"
```

**Quick Reference:**

| Task | Command |
|------|---------|
| Find React docs | `search "react"` |
| Get React hooks info | `context "/facebook/react" "useEffect cleanup"` |
| Find Next.js docs | `search "next.js"` |
| Get Next.js routing | `context "/vercel/next.js" "app router dynamic routes"` |
| Find Tailwind docs | `search "tailwind css"` |
| Get Tailwind config | `context "/tailwindlabs/tailwindcss" "configuration"` |
| Find Framer Motion | `search "framer motion"` |
| Get animation patterns | `context "/framer/motion" "animation variants"` |

**Options:**
- `--type txt|md` - Output format (default: txt)
- `--tokens N` - Limit response tokens

**Note:** API key is stored in `.agents/skills/context7/.env` (hidden file)

---

### 2. Golang Pro Skill

**Location:** `.agents/skills/golang-pro/SKILL.md`

**Use this skill when:**
- Writing or modifying Go backend code
- Implementing API endpoints
- Working with database operations
- Adding authentication/authorization logic
- Optimizing backend performance
- Handling file uploads (MinIO)
- Writing Go tests

**Key Guidelines:**
- Follow Go best practices and idioms
- Use Fiber framework patterns
- Implement proper error handling
- Write clean, testable code
- Use dependency injection
- Follow repository pattern
- Implement proper logging

**Project Structure:**
```
apps/backend/
├── cmd/
│   ├── api/          # Main application entry
│   └── dbcli/        # Database CLI tool
├── internal/
│   ├── auth/         # JWT authentication
│   ├── config/       # Configuration (env-based)
│   ├── database/     # PostgreSQL connection via GORM
│   ├── domain/       # Domain models (GORM structs)
│   ├── dto/          # Data transfer objects (16 files)
│   ├── handler/      # HTTP handlers (17 files)
│   ├── middleware/    # Auth + Capability middleware
│   ├── repository/   # Database repositories (14 files)
│   ├── service/      # Business logic (5 files)
│   └── storage/      # MinIO storage
```

**Common Patterns:**
```go
// Handler pattern
func (h *Handler) GetPortfolio(c *fiber.Ctx) error {
    id := c.Params("id")
    portfolio, err := h.portfolioRepo.GetByID(c.Context(), id)
    if err != nil {
        return c.Status(404).JSON(dto.ErrorResponse("NOT_FOUND", "Portfolio tidak ditemukan"))
    }
    return c.JSON(dto.SuccessResponse(portfolio))
}

// Capability middleware
adminRoutes.Get("/jurusan", capMiddleware.RequireCapability("majors"), adminHandler.ListJurusan)
```

---

### 3. Git Commit Skill

**Location:** `.agents/skills/git-commit/SKILL.md`

**Use this skill when:**
- Creating git commits
- User asks to commit changes

**Features:**
- Conventional commit analysis from diff
- Intelligent file staging
- Message generation following conventional commits spec

---

## 🎯 Decision Tree: Which Skill to Use?

```
Is the task about...

├─ Need library documentation?
│  └─ Use: context7 (FIRST!)
│     Examples: Installing packages, checking API signatures, latest versions
│
├─ Backend Go code?
│  └─ Use: golang-pro
│     Examples: API endpoints, database queries, business logic
│
├─ Creating a commit?
│  └─ Use: git-commit
│     Examples: Staging files, writing commit messages
│
└─ Frontend UI/UX?
   └─ Follow design system guidelines below (no dedicated skill)
      Examples: Building components, styling, layouts, animations
```

---

## 🎨 Frontend Design Guidelines

When building frontend UI, follow these guidelines:

### Key Rules
- Follow Grafikarsa design system (colors, typography, spacing)
- Use shadcn/ui components consistently
- Always use Lucide React icons (NEVER emojis)
- Ensure WCAG 2.1 AA accessibility compliance
- Test in both light and dark modes
- Implement mobile-first responsive design
- Use smooth transitions (150-300ms)
- Provide helpful loading and empty states

### Pre-Delivery Checklist
- [ ] No emojis used as icons
- [ ] Proper color contrast (4.5:1 minimum)
- [ ] Responsive at 375px, 768px, 1024px, 1440px
- [ ] Keyboard navigation works
- [ ] Light and dark modes tested
- [ ] All images have alt text
- [ ] Touch targets are 44x44px minimum

### Common Patterns
```tsx
// Page layout
<div className="container mx-auto max-w-5xl px-6 py-8 md:px-12 lg:px-16">
  <h1 className="mb-6 text-3xl font-bold">Page Title</h1>
  {/* Content */}
</div>

// Card grid
<div className="grid grid-cols-[repeat(auto-fill,minmax(320px,320px))] gap-6">
  {items.map(item => <Card key={item.id}>...</Card>)}
</div>

// Image with aspect ratio
<div className="relative aspect-[4/3] overflow-hidden rounded-lg">
  <Image src={url} alt="Description" fill className="object-cover object-center" />
</div>
```

---

## 📁 Code Organization

### Frontend (`apps/web/`)
```
app/                          # Next.js App Router pages
├── (auth)/                   # Auth route group (login)
├── (main)/                   # Main app shell (sidebar/header switching)
│   ├── page.tsx              # Home/Feed
│   ├── [username]/           # User profile + portfolio detail
│   ├── portfolios/           # Portfolio catalog
│   ├── users/                # User directory
│   ├── ai-ideas/             # AI project idea generator
│   ├── changelog/            # Changelog timeline
│   ├── notifications/        # Notification inbox
│   └── settings/             # User settings
├── admin/                    # Admin dashboard (16 sub-pages)
│   ├── [secret-path]/        # Dynamic admin login
│   └── (dashboard)/          # Admin shell
│       ├── page.tsx          # Dashboard stats
│       ├── users/            # User management
│       ├── portfolios/       # Portfolio management
│       ├── moderation/       # Portfolio moderation
│       ├── series/           # Series template management
│       ├── tags/             # Tag management
│       ├── majors/           # Major (jurusan) management
│       ├── classes/          # Class (kelas) management
│       ├── academic-years/   # Academic year management
│       ├── assessments/      # Portfolio assessment
│       ├── assessment-metrics/ # Assessment metric CRUD
│       ├── special-roles/    # Special role management
│       ├── feedback/         # Feedback management
│       ├── changelogs/       # Changelog management
│       └── import/           # Student import (Excel)
└── api/upload-proxy/         # CORS proxy for MinIO uploads

components/                   # Reusable UI components
├── admin/                    # Admin-specific components
├── ai/                       # AI wizard components
├── changelog/                # Changelog components
├── feed/                     # Feed/timeline components
├── landing/                  # Landing page sections
├── layout/                   # Layout components (sidebar, header, nav)
├── portfolio/                # Portfolio cards, editor, block editor
├── providers/                # React Query + Auth providers
├── ui/                       # shadcn/ui components (40 files)
└── user/                     # User profile, edit forms, social chips

lib/                          # Utilities and API layer
├── api/                      # API client functions (13 modules)
│   ├── admin/                # Admin API functions
│   ├── ai/                   # AI API functions
│   ├── client.ts             # Axios instance with interceptors
│   └── ...                   # Feature-specific API modules
├── constants/                # App constants (social platforms)
├── hooks/                    # Custom React hooks (4 hooks)
├── stores/                   # Zustand stores (auth, ui)
├── types/                    # TypeScript type definitions (9 files)
└── utils/                    # Utility functions (format, crop, embed, etc.)
```

### Backend (`apps/backend/`)
```
cmd/
├── api/main.go               # Main API entry point
└── dbcli/main.go             # Database CLI tool

internal/
├── auth/jwt.go               # JWT service (access + refresh tokens)
├── config/config.go          # Environment-based configuration
├── database/database.go      # PostgreSQL connection
├── domain/models.go          # GORM domain models
├── dto/                      # Data transfer objects (16 files)
├── handler/                  # HTTP handlers (17 files)
│   ├── admin_handler.go      # Admin CRUD + dashboard
│   ├── ai_handler.go         # AI project ideas (Gemini)
│   ├── assessment_handler.go # Portfolio assessment
│   ├── auth_handler.go       # Login/logout/refresh
│   ├── portfolio_handler.go  # Portfolio CRUD + social
│   ├── feed_handler.go       # Feed with smart algorithm
│   └── ...
├── middleware/
│   ├── auth.go               # JWT authentication
│   └── capability.go         # Special role capability check
├── repository/               # Database repositories (14 files)
├── service/                  # Business logic (5 files)
│   ├── feed_service.go       # Smart feed algorithm
│   ├── notification_service.go
│   ├── comment_service.go
│   └── captcha_service.go
└── storage/minio.go          # MinIO presigned URL uploads
```

---

## 📚 Documentation

| Document | Location | Description |
|----------|----------|-------------|
| PRD | `docs/prd.md` | Product requirements and features |
| API Docs | `docs/api/api.md` | REST API documentation |
| API Standards | `docs/api/rest-api-standards.md` | API conventions |
| Development Guide | `docs/dev/dev.md` | Setup, workflow, troubleshooting |
| Deployment (VPS) | `docs/deploy/deployment-ubuntu-vps.md` | Ubuntu 24 VPS deployment |
| Deployment (LXC) | `docs/deploy/deployment-ubuntu-lxc.md` | Proxmox LXC deployment |
| JWT Implementation | `docs/jwt/jwt-implementation.md` | JWT auth details |
| MinIO Implementation | `docs/minio/minio-implementation.md` | Object storage details |
| Image Standards | `docs/ui/image-standards.md` | Image dimension specs |
| Tone of Voice | `docs/ui/tone-of-voice.md` | UI copy guidelines |
| Social Chips | `docs/ui/social-media-chips.md` | Social media chip specs |
| Admin DataTable | `docs/ui/admin-datatable-style-guide.md` | Admin table styling |
| Web Routing | `docs/web/web-routing.md` | Frontend routing docs |
| Debug Mode | `docs/web/debug-mode.md` | Debug mode documentation |
| AI Features | `docs/ai/SUMMARY.md` | AI features overview |
| AI Architecture | `docs/ai/ARCHITECTURE.md` | AI architecture details |
| AI Ideas Wizard | `docs/ai/ai-ideas-wizard-guide.md` | AI wizard guide |
| AI Configuration | `docs/ai/ai-features-configuration.md` | AI config options |

---

## 🚀 Common Commands

### Development
```bash
make dev                  # Start backend services (db, minio, redis, backend)
make dev-web              # Start frontend (run in separate terminal)
make dev-down             # Stop backend services
make dev-logs             # View backend logs
```

### Database
```bash
make db-import            # Import schema from db/db.sql
make db-backup            # Backup database
make db-shell             # Open psql shell
make db-reset             # Reset database (WARNING: deletes data)
```

### Production
```bash
make prod                 # Build and run production simulation
make prod-down            # Stop production
make build                # Build Docker images
make push                 # Push images to Docker Hub
```

### Testing
```bash
make test-backend         # Run Go tests
make test-backend-cover   # Run tests with coverage
```

### Frontend (from apps/web/)
```bash
npm run dev               # Start dev server
npm run build             # Build for production
npm run lint              # Run ESLint
```

### Backend (from apps/backend/)
```bash
go run cmd/api/main.go    # Run API server
go test ./...             # Run all tests
```

---

## 🔐 Environment Variables

Key variables (see `.env.example` for full list):

| Category | Variables |
|----------|-----------|
| Database | `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` |
| MinIO | `MINIO_ENDPOINT`, `MINIO_ACCESS_KEY`, `MINIO_SECRET_KEY`, `MINIO_BUCKET` |
| JWT | `JWT_ACCESS_SECRET`, `JWT_REFRESH_SECRET`, `JWT_ACCESS_EXPIRY`, `JWT_REFRESH_EXPIRY` |
| Redis | `REDIS_HOST`, `REDIS_PORT`, `REDIS_PASSWORD` |
| AI | `GOOGLE_GEMINI_API_KEY`, `NEXT_PUBLIC_AI_FEATURES_ENABLED` |
| Frontend | `NEXT_PUBLIC_API_URL`, `NEXT_PUBLIC_APP_URL`, `NEXT_PUBLIC_STORAGE_URL` |
| Admin | `ADMIN_LOGIN_PATH` (secret URL slug for admin login) |

---

## ⚠️ Critical Rules

1. **ALWAYS use context7 when working with external libraries** - Don't rely on outdated knowledge
2. **Follow the design system** - Don't deviate from established patterns
3. **Use Lucide React icons** - NEVER use emojis as UI icons
4. **Test your changes** - Both manually and with automated tests
5. **Consider accessibility** - This is a school project, accessibility matters
6. **Use systematic debugging** - Don't guess when encountering bugs
7. **Write clean, maintainable code** - Others will work on this
8. **Follow capability-based auth** - Check middleware/capability.go for role access patterns
9. **Ask for clarification** - If requirements are unclear, ask the user

---

## 🤝 Contributing Guidelines

When working on this project:
- Follow the established code style and patterns
- Write meaningful commit messages (conventional commits)
- Test thoroughly before submitting
- Update documentation when adding features
- Consider backward compatibility
- Think about edge cases and error handling
- Optimize for performance and user experience

---

## 📞 Questions?

If you're unsure about:
- **Library APIs or versions:** Use `context7` to fetch current documentation
- **Backend patterns:** Check `golang-pro` skill and existing handlers
- **Design decisions:** Follow frontend design guidelines above and check existing components
- **Git commits:** Use `git-commit` skill for conventional commits

Remember: The skills are there to help you deliver high-quality, consistent code. Use them proactively!
