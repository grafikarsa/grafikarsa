# Changelog - API Test Scripts

## [1.1.0] - 2026-02-10

### Added - Detailed Error Tracking

#### New Features

1. **Comprehensive Error Details in Test Summary**
   - Setiap test yang gagal sekarang menampilkan detail lengkap
   - Informasi meliputi: endpoint, test description, dan reason

2. **Error Information Captured**
   - **Endpoint**: HTTP method dan path (e.g., `POST /auth/login`)
   - **Test**: Deskripsi test yang gagal (e.g., `Expected status 200, got 401`)
   - **Reason**: Alasan spesifik dengan error code dan message dari API

3. **Multiple Error Types Tracked**
   - HTTP status mismatch dengan API error message
   - Connection errors
   - Curl errors dengan exit code
   - Server errors (5xx)
   - Validation errors dengan details

#### Example Output

**Before (v1.0.0):**
```bash
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  TEST SUMMARY
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Total Tests: 81
Passed: 78
Failed: 3

✗ SOME TESTS FAILED
```

**After (v1.1.0):**
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

#### Technical Changes

**File: `scripts/api-test/lib/test-helpers.sh`**

1. Added error tracking array:
   ```bash
   declare -a FAILED_TEST_DETAILS=()
   ```

2. Enhanced `print_fail()` function:
   - Now accepts 3 parameters: message, endpoint, reason
   - Stores failure details in array for summary

3. Enhanced `make_request()` function:
   - Detects curl failures with exit codes
   - Detects server errors (5xx)
   - Detects connection failures
   - Automatically tracks errors

4. Enhanced `test_endpoint()` function:
   - Extracts error code and message from API response
   - Provides detailed failure information
   - Tracks all failure types

5. Enhanced `print_summary()` function:
   - Displays "FAILED TESTS DETAILS" section when failures exist
   - Shows numbered list of all failures
   - Includes endpoint, test description, and reason for each failure

**File: `scripts/api-test/test-api-complete.sh`**

1. Removed `set -e`:
   - Allows script to continue after errors
   - Collects all failures instead of stopping at first error

#### Benefits

1. **Quick Identification**: Langsung tahu endpoint mana yang error
2. **Root Cause Analysis**: Tahu alasan spesifik kenapa error
3. **Easy Debugging**: Informasi lengkap untuk debugging
4. **Comprehensive**: Semua error di-track, tidak ada yang terlewat
5. **Actionable**: Bisa langsung fix berdasarkan error message

#### Documentation

New documentation files:
- `scripts/api-test/ERROR_TRACKING_EXAMPLE.md` - Detailed examples and troubleshooting
- `scripts/api-test/CHANGELOG.md` - This file

Updated documentation:
- `scripts/api-test/API_TEST_GUIDE.md` - Added error tracking section
- `scripts/api-test/README.md` - Updated with error tracking info

---

## [1.0.0] - 2026-02-10

### Initial Release

#### Features

1. **Comprehensive Test Coverage**
   - 73 endpoints tested
   - 81+ test scenarios
   - Success & error cases

2. **Beautiful Output**
   - Colored terminal output
   - JSON formatting with jq
   - Clear request/response display

3. **Immutable to Database**
   - Auto cleanup on exit
   - No data left behind
   - Safe to run multiple times

4. **Real Operations**
   - Not mocked
   - Real API calls
   - Real database operations

5. **Easy to Use**
   - Single command: `make test-api`
   - Environment variable configuration
   - Modular structure

#### Files Created

- `scripts/api-test/test-api-complete.sh` - Main test script
- `scripts/api-test/lib/test-helpers.sh` - Helper functions
- `scripts/api-test/README.md` - Documentation
- `scripts/api-test/API_TEST_GUIDE.md` - Usage guide
- `Makefile` - Added `test-api` command

---

## Future Enhancements

### Planned Features

- [ ] JSON test report output
- [ ] HTML test report generation
- [ ] CI/CD integration examples
- [ ] Performance metrics tracking
- [ ] Test result history
- [ ] Parallel test execution
- [ ] Custom test filters
- [ ] Test retry mechanism

### Ideas

- Integration with test reporting tools
- Slack/Discord notifications for failures
- Test coverage visualization
- API response time tracking
- Load testing capabilities

---

**Maintained by**: Grafikarsa Development Team  
**Project**: SMKN 4 Malang Portfolio Platform
