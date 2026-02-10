# Grafikarsa API Test Guide

## Overview

Comprehensive test suite untuk **SEMUA** endpoint API Grafikarsa telah dibuat dengan fitur:

✅ **81+ test scenarios** mencakup semua endpoint  
✅ **Semua jenis user** (admin, student, alumni)  
✅ **Success & error scenarios** lengkap  
✅ **Beautiful JSON output** dengan colored terminal  
✅ **Immutable to database** (auto cleanup)  
✅ **Real operations** (bukan mock)  

## Quick Start

### 1. Prerequisites

Pastikan sudah terinstall:
```bash
# Check jq (JSON processor)
jq --version

# Install jika belum ada
sudo apt-get install jq  # Ubuntu/Debian
brew install jq          # macOS
```

### 2. Start API Server

```bash
# Start semua services
make dev

# Tunggu sampai API ready (check logs)
docker compose logs -f api
```

### 3. Run Tests

```bash
# Cara termudah - menggunakan make command
make test-api

# Atau langsung run script
./scripts/test-api-complete.sh

# Dengan custom configuration
API_BASE_URL=http://localhost:8080/api/v1 \
ADMIN_USERNAME=admin \
ADMIN_PASSWORD=admin123 \
./scripts/test-api-complete.sh
```

## Test Coverage

### Endpoint Categories Tested

| Category | Endpoints | Scenarios |
|----------|-----------|-----------|
| **Public** | 8 | Health, majors, classes, tags |
| **Authentication** | 5 | Login, sessions, errors |
| **Users** | 9 | List, search, profile, followers |
| **Profiles** | 6 | Get, update, check username |
| **Portfolios** | 11 | CRUD, search, filter, sort |
| **Content Blocks** | 9 | All block types, CRUD |
| **Search** | 4 | Users & portfolios search |
| **Social** | 8 | Follow/unfollow, like/unlike |
| **Feed** | 2 | Timeline, auth errors |
| **Admin** | 11 | All admin endpoints |
| **Upload** | 4 | Presigned URLs, validation |
| **Lifecycle** | 4 | Submit, archive, delete |

**Total: 81+ test scenarios**

## 🔍 Error Tracking & Detailed Reporting

Script sekarang menampilkan **detail lengkap** untuk setiap test yang gagal di akhir test summary!

### Test Summary - All Pass

```bash
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TEST SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Tests: 81
Passed: 81
Failed: 0

✓ ALL TESTS PASSED!
```

### Test Summary - With Failures (Detailed Error Info)

```bash
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TEST SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Tests: 81
Passed: 78
Failed: 3

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  FAILED TESTS DETAILS
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

1. Endpoint: POST /auth/login
   Test: Expected status 200, got 401
   Reason: Error: INVALID_CREDENTIALS - Username atau password salah

2. Endpoint: GET /portfolios/nonexistent/portfolio-slug
   Test: Expected status 404, got 500
   Reason: Error: INTERNAL_ERROR - Terjadi kesalahan pada server

3. Endpoint: POST /portfolios
   Test: Request failed
   Reason: Curl error (exit code: 7)

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

✗ SOME TESTS FAILED
Please check the details above for more information.
```

### Error Information Captured

Untuk setiap test yang gagal, script menampilkan:

1. **Endpoint**: HTTP method dan path yang di-test
   - Contoh: `POST /auth/login`, `GET /users/:username`

2. **Test**: Deskripsi test yang gagal
   - Contoh: `Expected status 200, got 401`
   - Contoh: `Request failed`

3. **Reason**: Alasan spesifik kenapa test gagal
   - HTTP error dengan error code dan message dari API
   - Curl error dengan exit code
   - Connection error
   - Server error (5xx)

### Types of Errors Tracked

- ❌ **HTTP Status Mismatch**: Expected vs actual status code dengan error message dari API
- ❌ **Connection Errors**: Cannot connect to server
- ❌ **Curl Errors**: Network atau request errors dengan exit code
- ❌ **Server Errors**: 5xx responses
- ❌ **Validation Errors**: 422 dengan validation details
- ❌ **Authentication Errors**: 401/403 dengan error codes

## Test Output Example

```bash
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  GRAFIKARSA API COMPLETE TEST SUITE
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

ℹ INFO: Base URL: http://localhost:8080/api/v1
ℹ INFO: Testing ALL endpoints with ALL scenarios

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
  "status": "ok"
}
└─

...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TEST SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Tests: 81
Passed: 81
Failed: 0

✓ ALL TESTS PASSED!
```

## Features

### 1. Detailed Request/Response Display

Setiap test menampilkan:
- **HTTP Method & Path**
- **Description** singkat
- **Query Parameters** (jika ada)
- **Request Body** (formatted JSON)
- **Response Status Code**
- **Response Body** (formatted JSON)

### 2. Error Scenario Testing

Semua error scenarios di-test:
- Invalid credentials
- Missing required fields
- Unauthorized access
- Not found errors
- Validation errors
- Conflict errors (duplicate, already exists)
- Permission errors

### 3. Immutable Database

Script menggunakan cleanup strategy:
```bash
# Automatic cleanup on exit
trap cleanup EXIT

cleanup() {
    # Delete test portfolio
    # Delete test data
    # Restore original state
}
```

### 4. Colored Output

- 🔵 **Blue** - Test info
- 🟢 **Green** - Success responses
- 🟡 **Yellow** - Requests & warnings
- 🔴 **Red** - Errors & failures
- 🟣 **Magenta** - Section headers
- 🔷 **Cyan** - Main headers

## File Structure

```
scripts/
├── README.md                    # Dokumentasi lengkap
├── lib/
│   └── test-helpers.sh         # Helper functions
├── test-api-complete.sh        # Main comprehensive test
├── test-api-auth.sh            # Modular: Authentication
├── test-api-public.sh          # Modular: Public endpoints
├── test-api-users.sh           # Modular: User endpoints
├── test-api-portfolios.sh      # Modular: Portfolio endpoints
├── test-api-content-blocks.sh  # Modular: Content blocks
└── test-api-social.sh          # Modular: Social features
```

## Configuration

### Environment Variables

```bash
# API Base URL (default: http://localhost:8080/api/v1)
export API_BASE_URL="http://localhost:8080/api/v1"

# Admin credentials (default: admin/admin123)
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD="admin123"
```

### Custom Configuration Example

```bash
# Test against staging
API_BASE_URL=https://staging-api.grafikarsa.com/api/v1 \
ADMIN_USERNAME=staging_admin \
ADMIN_PASSWORD=staging_pass \
make test-api

# Test against production (read-only tests)
API_BASE_URL=https://api.grafikarsa.com/api/v1 \
ADMIN_USERNAME=readonly_admin \
ADMIN_PASSWORD=readonly_pass \
make test-api
```

## Test Details

### Authentication Flow

1. Login dengan admin credentials
2. Simpan access token
3. Gunakan token untuk authenticated requests
4. Test error scenarios (invalid credentials, etc.)

### Portfolio Lifecycle

1. Create portfolio (draft)
2. Add content blocks (text, image, youtube, table, button)
3. Update portfolio
4. Test submit (incomplete - error)
5. Archive portfolio
6. Unarchive portfolio
7. Delete portfolio (cleanup)

### Social Features

1. Follow user
2. Test already following (error)
3. Unfollow user
4. Test not following (error)
5. Like portfolio
6. Test already liked (error)
7. Unlike portfolio

## Running Specific Test Modules

Jika hanya ingin test kategori tertentu:

```bash
# Test authentication only
./scripts/test-api-auth.sh

# Test public endpoints only
./scripts/test-api-public.sh

# Test user endpoints only
./scripts/test-api-users.sh

# Test portfolios only
./scripts/test-api-portfolios.sh

# Test content blocks only
./scripts/test-api-content-blocks.sh

# Test social features only
./scripts/test-api-social.sh
```

## Troubleshooting

### Problem: API not responding

```bash
# Check if API is running
docker compose ps

# Check API logs
docker compose logs api

# Restart API
docker compose restart api
```

### Problem: jq command not found

```bash
# Install jq
sudo apt-get install jq  # Ubuntu/Debian
brew install jq          # macOS
sudo pacman -S jq        # Arch Linux
```

### Problem: Permission denied

```bash
# Make scripts executable
chmod +x scripts/test-api-complete.sh
chmod +x scripts/lib/test-helpers.sh
```

### Problem: Admin login failed

```bash
# Seed admin user
make seed-admin

# Or with custom credentials
make seed-admin USERNAME=admin PASSWORD=admin123 EMAIL=admin@test.com
```

### Problem: Database not initialized

```bash
# Clean and restart
make clean
make dev
```

## Adding New Tests

Jika ada endpoint baru:

1. **Tambahkan ke test-api-complete.sh**:
```bash
print_section "X.X GET /new-endpoint - Description"
make_request "GET" "/new-endpoint" "Test description" "" "$ADMIN_TOKEN"
```

2. **Test error scenarios**:
```bash
print_section "X.X GET /new-endpoint - Error Case"
make_request "GET" "/new-endpoint/invalid" "Error scenario" "" "$ADMIN_TOKEN"
```

3. **Update dokumentasi**:
- Update `scripts/README.md`
- Update `API_TEST_GUIDE.md`
- Update test count

## Best Practices

1. **Always run full test suite** sebelum commit
2. **Check test output** untuk memastikan semua pass
3. **Update tests** ketika API berubah
4. **Document new endpoints** di test scripts
5. **Keep tests immutable** - jangan tinggalkan data sampah

## Related Documentation

- [API Documentation](docs/api.md) - Complete API reference
- [API Routes Complete](API_ROUTES_COMPLETE.md) - All implemented routes
- [Implementation Summary](IMPLEMENTATION_SUMMARY.md) - Implementation details
- [Test Scripts README](scripts/README.md) - Detailed test documentation

## Checklist

Sebelum deploy ke production:

- [ ] Semua tests pass (`make test-api`)
- [ ] No data left in database after tests
- [ ] All error scenarios tested
- [ ] Documentation updated
- [ ] New endpoints added to tests

## Contributing

Ketika menambah endpoint baru:

1. Implement endpoint di backend
2. Add to `docs/api.md`
3. Add tests to `test-api-complete.sh`
4. Run full test suite
5. Update documentation
6. Create PR

## License

Part of Grafikarsa project - SMKN 4 Malang Portfolio Platform.

---

**Last Updated**: 2026-02-10  
**Test Coverage**: 81+ scenarios  
**Status**: ✅ Production Ready
