# Grafikarsa API Test Scripts

Comprehensive test suite untuk semua endpoint API Grafikarsa.

## Overview

Test script ini mencakup **SEMUA** endpoint API Grafikarsa dengan:
- ✅ Semua jenis user (admin, student, alumni)
- ✅ Semua success scenarios
- ✅ Semua error scenarios dan error handling
- ✅ Request/response lengkap dengan format JSON yang indah
- ✅ Immutable ke database (cleanup otomatis)
- ✅ Real operations (bukan mock)

## Quick Start

### Prerequisites

1. API server harus running di `http://localhost:8080` (atau set `API_BASE_URL`)
2. Database sudah di-seed dengan admin user
3. `jq` installed untuk JSON formatting
4. `curl` installed untuk HTTP requests

### Install jq (jika belum ada)

```bash
# Ubuntu/Debian
sudo apt-get install jq

# macOS
brew install jq

# Arch Linux
sudo pacman -S jq
```

### Run Tests

```bash
# Menggunakan make command (recommended)
make test-api

# Atau langsung run script
./scripts/test-api-complete.sh

# Dengan custom API URL
API_BASE_URL=http://api.grafikarsa.com/api/v1 ./scripts/test-api-complete.sh

# Dengan custom admin credentials
ADMIN_USERNAME=myadmin ADMIN_PASSWORD=mypass ./scripts/test-api-complete.sh
```

## File Structure

```
scripts/
├── README.md                    # Dokumentasi ini
├── lib/
│   └── test-helpers.sh         # Helper functions untuk testing
├── test-api-complete.sh        # Main comprehensive test script
├── test-api-auth.sh            # Authentication tests (modular)
├── test-api-public.sh          # Public endpoints tests (modular)
├── test-api-users.sh           # User endpoints tests (modular)
├── test-api-portfolios.sh      # Portfolio endpoints tests (modular)
├── test-api-content-blocks.sh  # Content blocks tests (modular)
└── test-api-social.sh          # Social & likes tests (modular)
```

## Test Coverage

### 1. Public Endpoints (8 tests)
- Health check
- List majors
- List classes (with filters)
- Get active academic year
- List tags (with search)

### 2. Authentication (5 tests)
- Login success
- Login with invalid credentials (error)
- Login with non-existent user (error)
- Login with missing fields (error)
- List active sessions

### 3. User Endpoints (9 tests)
- List all users (public)
- Search users
- Filter by role, major, class
- Pagination
- Get user profile
- User not found (error)
- List followers/following

### 4. Profile Endpoints (6 tests)
- Get current user profile
- Unauthorized access (error)
- Update profile
- Check username availability
- Update social links

### 5. Portfolio Endpoints (11 tests)
- List published portfolios
- Search and filter portfolios
- Sort portfolios
- Get portfolio detail
- Portfolio not found (error)
- Get my portfolios
- Create portfolio
- Validation errors (error)
- Update portfolio

### 6. Content Blocks (9 tests)
- Add text block
- Add image block
- Add YouTube block
- Add table block
- Add button block
- Invalid block type (error)
- Get all blocks
- Update block
- Delete block

### 7. Search Endpoints (4 tests)
- Search users
- Search users with filters
- Search portfolios
- Search portfolios with filters

### 8. Social Endpoints (8 tests)
- Follow user
- Already following (error)
- Unfollow user
- Not following (error)
- Cannot follow self (error)
- Like portfolio
- Already liked (error)
- Unlike portfolio

### 9. Feed Endpoint (2 tests)
- Get feed (authenticated)
- Unauthorized access (error)

### 10. Admin Endpoints (11 tests)
- List majors
- List academic years
- List classes
- List users
- Search users
- Get user detail
- List tags
- List all portfolios
- Filter portfolios by status
- Get portfolio detail
- Moderation queue

### 11. Upload Endpoints (4 tests)
- Request presigned URL (avatar)
- File too large (error)
- Invalid content type (error)
- Request presigned URL (thumbnail)

### 12. Portfolio Lifecycle (4 tests)
- Submit incomplete portfolio (error)
- Archive portfolio
- Unarchive portfolio
- Delete portfolio

**Total: 81+ test scenarios**

## Output Format

Test output menggunakan colored terminal dengan format yang jelas:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  1. PUBLIC ENDPOINTS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

▶ 1.1 GET /health - Health Check

┌─ REQUEST
│ Method: GET
│ Path: /health
│ Description: API health check
└─

┌─ RESPONSE
│ Status: 200
└─

┌─ RESPONSE BODY
{
  "status": "ok",
  "timestamp": "2026-02-10T10:00:00Z"
}
└─
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `API_BASE_URL` | `http://localhost:8080/api/v1` | Base URL API |
| `ADMIN_USERNAME` | `admin` | Username admin untuk testing |
| `ADMIN_PASSWORD` | `admin123` | Password admin untuk testing |

### Example

```bash
# Test against production API
API_BASE_URL=https://api.grafikarsa.com/api/v1 \
ADMIN_USERNAME=testadmin \
ADMIN_PASSWORD=testpass123 \
./scripts/test-api-complete.sh
```

## Cleanup

Script ini **immutable** ke database:
- Semua data test yang dibuat akan dihapus otomatis
- Menggunakan `trap` untuk cleanup on exit
- Portfolio, content blocks, dan data lainnya dibersihkan

## Test Summary

Di akhir test, akan ditampilkan summary:

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TEST SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Tests: 81
Passed: 81
Failed: 0

✓ ALL TESTS PASSED!
```

## Troubleshooting

### API tidak running
```bash
# Start API dengan docker compose
make dev
```

### jq not found
```bash
# Install jq
sudo apt-get install jq  # Ubuntu/Debian
brew install jq          # macOS
```

### Permission denied
```bash
# Make scripts executable
chmod +x scripts/test-api-complete.sh
chmod +x scripts/lib/test-helpers.sh
```

### Admin login failed
```bash
# Seed admin user
make seed-admin

# Atau dengan custom credentials
make seed-admin USERNAME=admin PASSWORD=admin123
```

## Notes

1. **Immutability**: Script ini tidak akan meninggalkan data sampah di database
2. **Real Operations**: Semua operasi adalah real API calls, bukan mock
3. **Error Scenarios**: Semua error scenarios di-test untuk memastikan error handling bekerja
4. **Comprehensive**: Mencakup SEMUA endpoint yang ada di `docs/api.md`

## Contributing

Jika ada endpoint baru yang ditambahkan:
1. Tambahkan test di `test-api-complete.sh`
2. Pastikan cleanup berfungsi dengan baik
3. Update dokumentasi ini

## License

Part of Grafikarsa project.
