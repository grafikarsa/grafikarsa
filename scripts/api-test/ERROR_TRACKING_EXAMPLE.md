# Error Tracking Example

## Overview

Test script sekarang menampilkan detail lengkap untuk setiap test yang gagal di akhir test summary.

## Example Output

### When Tests Pass

```bash
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TEST SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Tests: 81
Passed: 81
Failed: 0

✓ ALL TESTS PASSED!
```

### When Tests Fail

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

## Error Information Captured

Untuk setiap test yang gagal, script akan menampilkan:

1. **Endpoint**: HTTP method dan path yang di-test
   - Contoh: `POST /auth/login`, `GET /users/:username`

2. **Test**: Deskripsi test yang gagal
   - Contoh: `Expected status 200, got 401`
   - Contoh: `Request failed`

3. **Reason**: Alasan kenapa test gagal
   - HTTP error dengan error code dan message dari API
   - Curl error dengan exit code
   - Connection error
   - Server error (5xx)

## Types of Errors Tracked

### 1. HTTP Status Mismatch
```
Endpoint: POST /auth/login
Test: Expected status 200, got 401
Reason: Error: INVALID_CREDENTIALS - Username atau password salah
```

### 2. Connection Errors
```
Endpoint: GET /health
Test: Request failed
Reason: Could not connect to server
```

### 3. Curl Errors
```
Endpoint: POST /portfolios
Test: Request failed
Reason: Curl error (exit code: 7)
```

### 4. Server Errors (5xx)
```
Endpoint: GET /portfolios/:id
Test: Server error (5xx)
Reason: HTTP 500 - Server error
```

### 5. Validation Errors
```
Endpoint: POST /portfolios
Test: Expected status 201, got 422
Reason: Error: VALIDATION_ERROR - Validasi gagal
```

## Benefits

1. **Quick Identification**: Langsung tahu endpoint mana yang error
2. **Root Cause**: Tahu alasan spesifik kenapa error
3. **Easy Debugging**: Informasi lengkap untuk debugging
4. **Comprehensive**: Semua error di-track, tidak ada yang terlewat
5. **Actionable**: Bisa langsung fix berdasarkan error message

## Usage

Script akan otomatis menampilkan error details di akhir test:

```bash
# Run tests
make test-api

# Atau
./scripts/api-test/test-api-complete.sh
```

Tidak perlu konfigurasi tambahan, error tracking sudah built-in.

## Error Codes Reference

Common error codes yang mungkin muncul:

| Error Code | HTTP Status | Description |
|------------|-------------|-------------|
| `INVALID_CREDENTIALS` | 401 | Username atau password salah |
| `TOKEN_EXPIRED` | 401 | Token sudah expired |
| `UNAUTHORIZED` | 401 | Token tidak valid atau tidak ada |
| `FORBIDDEN` | 403 | Tidak punya akses ke resource |
| `NOT_FOUND` | 404 | Resource tidak ditemukan |
| `USER_NOT_FOUND` | 404 | User tidak ditemukan |
| `PORTFOLIO_NOT_FOUND` | 404 | Portfolio tidak ditemukan |
| `ALREADY_FOLLOWING` | 409 | Sudah follow user |
| `ALREADY_LIKED` | 409 | Sudah like portfolio |
| `USERNAME_TAKEN` | 409 | Username sudah digunakan |
| `VALIDATION_ERROR` | 422 | Validasi input gagal |
| `INCOMPLETE_PORTFOLIO` | 422 | Portfolio belum lengkap |
| `RATE_LIMIT_EXCEEDED` | 429 | Rate limit terlampaui |
| `INTERNAL_ERROR` | 500 | Server error |

## Troubleshooting

### If you see connection errors:
```bash
# Check if API is running
docker compose ps

# Check API logs
docker compose logs api

# Restart API
docker compose restart api
```

### If you see authentication errors:
```bash
# Seed admin user
make seed-admin

# Or with custom credentials
ADMIN_USERNAME=admin ADMIN_PASSWORD=admin123 make seed-admin
```

### If you see database errors:
```bash
# Reinitialize database
make clean
make dev
```

## Example: Debugging Failed Tests

Jika test summary menunjukkan:

```
1. Endpoint: POST /auth/login
   Test: Expected status 200, got 401
   Reason: Error: INVALID_CREDENTIALS - Username atau password salah
```

**Action**: Check admin credentials
```bash
# Verify admin user exists
docker compose exec postgres psql -U postgres -d grafikarsa -c "SELECT username, role FROM users WHERE role='admin';"

# Reseed admin
make seed-admin USERNAME=admin PASSWORD=admin123
```

---

Jika test summary menunjukkan:

```
2. Endpoint: GET /portfolios/:id
   Test: Server error (5xx)
   Reason: HTTP 500 - Server error
```

**Action**: Check API logs
```bash
# View recent logs
docker compose logs --tail=50 api

# Follow logs in real-time
docker compose logs -f api
```

---

Jika test summary menunjukkan:

```
3. Endpoint: POST /portfolios
   Test: Request failed
   Reason: Curl error (exit code: 7)
```

**Action**: Check network connectivity
```bash
# Test API connectivity
curl http://localhost:8080/api/v1/health

# Check if port is open
netstat -an | grep 8080

# Restart services
docker compose restart
```
