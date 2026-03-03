# Changelog

All notable changes to the Grafikarsa project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Monorepo structure consolidating backend and web into a single repository
- Three-tier Docker Compose configuration: development, production simulation, and server deployment
- Makefile with `dev`, `prod`, and utility targets for streamlined workflows
- Hot reload for backend development using Air
- Native frontend development via `npm run dev` (outside Docker for faster iteration)
- Single `.env` file at project root, injected into all services
- Development scripts with both Bash (.sh) and PowerShell (.ps1) versions
- Comprehensive API test suite (`api_test.sh` / `api_test.ps1`)
- API inspector tool for raw request/response debugging (`api_inspect.sh` / `api_inspect.ps1`)
- Development guide at `docs/dev/README.md`
- Deployment guide for Ubuntu 24 VPS at `docs/deploy/deployment-ubuntu-vps.md`
- Deployment guide for Ubuntu 22 LXC at `docs/deploy/deployment-ubuntu-lxc.md`
- GitHub Actions CI/CD workflow for automated build, push, and deploy
- Docker Hub image publishing scripts with semantic versioning
- README, LICENSE, CONTRIBUTING, CODE_OF_CONDUCT, and SECURITY documentation

### Changed
- Migrated from polyrepo (separate `grafikarsa/web` and `grafikarsa/backend`) to monorepo
- Rewrote both Dockerfiles with explicit `development`, `builder`, and `production` stages
- Restructured environment variable management to use a single source of truth
- Updated GitHub Actions to use `docker-compose.deploy.yml` for server deployments
- Replaced JWT_SECRET with JWT_ACCESS_SECRET and JWT_REFRESH_SECRET for proper token separation
- Fixed all legacy path references (`infra/db/` to `db/`)
- Updated documentation to reflect monorepo structure

### Fixed
- Docker Compose environment variable mismatches between config.go and compose files
- Missing environment variables: DB_SSLMODE, ADMIN_LOGIN_PATH, MINIO_PRESIGN_HOST, MINIO_PRESIGN_USE_SSL
- Duplicate and conflicting entries in `.env.example`
- Outdated polyrepo URLs in `package.json`

### Removed
- Separate backend and web repositories
- Redundant environment files and duplicate docker-compose configurations
- Exposed SSH deploy keys from repository root

### Security
- Removed committed SSH private/public key pair from repository
- Added SSH key patterns to `.gitignore`
- Enforced strong secret generation guidance in `.env.example`

## [1.0.0] - 2024-01-01

### Added
- Initial release as polyrepo
- Backend API built with Go (Fiber framework, GORM ORM)
- Frontend built with Next.js (App Router)
- PostgreSQL database with schema migrations
- MinIO object storage for file uploads
- JWT-based authentication with access and refresh tokens
- User management with role-based access control (admin, student, alumni)
- Portfolio management with content blocks (text, image, table, YouTube, button, embed)
- Portfolio submission and moderation workflow (draft, pending, approved, rejected)
- Social features: follow system, portfolio likes
- Search functionality for users and portfolios
- Admin dashboard with statistics and CRUD operations
- Rate limiting and CORS configuration
