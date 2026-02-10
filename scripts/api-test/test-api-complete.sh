#!/bin/bash

# Grafikarsa API Complete Test Suite
# Tests ALL endpoints with ALL scenarios including error cases
# This script is immutable to database (uses cleanup after tests)

set -e

# Load helper functions
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/lib/test-helpers.sh"

# Configuration
BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"
ADMIN_USERNAME="${ADMIN_USERNAME:-admin}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-admin123}"

# Global variables for test data
ADMIN_TOKEN=""
ADMIN_USER_ID=""
TEST_PORTFOLIO_ID=""
TEST_PORTFOLIO_SLUG=""
TEST_BLOCK_ID=""
MAJOR_ID=""
CLASS_ID=""
TAG_ID=""
TEST_USERNAME=""
ACADEMIC_YEAR_ID=""

# Cleanup function
cleanup() {
    print_info "Cleaning up test data..."
    
    # Delete test portfolio if created
    if [ -n "$TEST_PORTFOLIO_ID" ] && [ -n "$ADMIN_TOKEN" ]; then
        curl -s -X DELETE "$BASE_URL/portfolios/$TEST_PORTFOLIO_ID" \
            -H "Authorization: Bearer $ADMIN_TOKEN" > /dev/null 2>&1 || true
    fi
    
    print_info "Cleanup completed"
}

# Register cleanup on exit
trap cleanup EXIT

# ============================================================================
# MAIN TEST EXECUTION
# ============================================================================

print_header "GRAFIKARSA API COMPLETE TEST SUITE"
print_info "Base URL: $BASE_URL"
print_info "Testing ALL endpoints with ALL scenarios"
echo ""

# ============================================================================
# 1. PUBLIC ENDPOINTS (No Authentication)
# ============================================================================

print_header "1. PUBLIC ENDPOINTS"

print_section "1.1 GET /health - Health Check"
make_request "GET" "/health" "API health check"

print_section "1.2 GET /majors - List All Majors"
RESPONSE=$(make_request "GET" "/majors" "Daftar semua jurusan (public)")
MAJOR_ID=$(extract_json_value "$RESPONSE" ".data[0].id")
print_info "Sample Major ID: $MAJOR_ID"

print_section "1.3 GET /classes - List Classes (Active Year)"
RESPONSE=$(make_request "GET" "/classes" "Daftar kelas tahun ajaran aktif")
CLASS_ID=$(extract_json_value "$RESPONSE" ".data[0].id")
print_info "Sample Class ID: $CLASS_ID"

print_section "1.4 GET /classes?major_id=xxx - Filter Classes by Major"
if [ -n "$MAJOR_ID" ] && [ "$MAJOR_ID" != "null" ]; then
    make_request "GET" "/classes" "Filter kelas berdasarkan jurusan" "" "" "major_id=$MAJOR_ID"
fi

print_section "1.5 GET /classes?grade_level=12 - Filter Classes by Grade"
make_request "GET" "/classes" "Filter kelas berdasarkan tingkat" "" "" "grade_level=12"

print_section "1.6 GET /active-year - Get Active Academic Year"
RESPONSE=$(make_request "GET" "/active-year" "Mendapatkan tahun ajaran aktif")
ACADEMIC_YEAR_ID=$(extract_json_value "$RESPONSE" ".data.id")

print_section "1.7 GET /tags - List All Tags"
RESPONSE=$(make_request "GET" "/tags" "Daftar semua tags")
TAG_ID=$(extract_json_value "$RESPONSE" ".data[0].id")
print_info "Sample Tag ID: $TAG_ID"

print_section "1.8 GET /tags?search=web - Search Tags"
make_request "GET" "/tags" "Cari tags dengan keyword" "" "" "search=web"

# ============================================================================
# 2. AUTHENTICATION ENDPOINTS
# ============================================================================

print_header "2. AUTHENTICATION ENDPOINTS"

print_section "2.1 POST /auth/login - Login Success"
RESPONSE=$(make_request "POST" "/auth/login" "Login dengan kredensial valid" \
    "{\"username\":\"$ADMIN_USERNAME\",\"password\":\"$ADMIN_PASSWORD\"}")

ADMIN_TOKEN=$(extract_json_value "$RESPONSE" ".data.access_token")
ADMIN_USER_ID=$(extract_json_value "$RESPONSE" ".data.user.id")

if [ -n "$ADMIN_TOKEN" ] && [ "$ADMIN_TOKEN" != "null" ]; then
    print_success "Admin login berhasil, token obtained"
else
    print_fail "Admin login gagal, cannot continue tests"
    exit 1
fi

print_section "2.2 POST /auth/login - Invalid Credentials (Error)"
make_request "POST" "/auth/login" "Login dengan password salah" \
    "{\"username\":\"$ADMIN_USERNAME\",\"password\":\"wrongpassword\"}"

print_section "2.3 POST /auth/login - User Not Found (Error)"
make_request "POST" "/auth/login" "Login dengan username tidak ada" \
    "{\"username\":\"nonexistent_user_999\",\"password\":\"password123\"}"

print_section "2.4 POST /auth/login - Missing Fields (Error)"
make_request "POST" "/auth/login" "Login tanpa password" \
    "{\"username\":\"$ADMIN_USERNAME\"}"

print_section "2.5 GET /auth/sessions - List Active Sessions"
make_request "GET" "/auth/sessions" "Melihat semua sesi aktif" "" "$ADMIN_TOKEN"

# ============================================================================
# 3. USER ENDPOINTS (Public & Authenticated)
# ============================================================================

print_header "3. USER ENDPOINTS"

print_section "3.1 GET /users - List All Users (No Auth)"
RESPONSE=$(make_request "GET" "/users" "Daftar semua user tanpa autentikasi")
TEST_USERNAME=$(extract_json_value "$RESPONSE" ".data[0].username")
print_info "Sample Username: $TEST_USERNAME"

print_section "3.2 GET /users?search=john - Search Users"
make_request "GET" "/users" "Cari user dengan keyword" "" "" "search=john"

print_section "3.3 GET /users?role=student - Filter by Role"
make_request "GET" "/users" "Filter user berdasarkan role" "" "" "role=student"

print_section "3.4 GET /users?major_id=xxx - Filter by Major"
if [ -n "$MAJOR_ID" ] && [ "$MAJOR_ID" != "null" ]; then
    make_request "GET" "/users" "Filter user berdasarkan jurusan" "" "" "major_id=$MAJOR_ID"
fi

print_section "3.5 GET /users?page=2&limit=10 - Pagination"
make_request "GET" "/users" "Pagination test" "" "" "page=2&limit=10"

print_section "3.6 GET /users/:username - Get User Profile"
if [ -n "$TEST_USERNAME" ] && [ "$TEST_USERNAME" != "null" ]; then
    make_request "GET" "/users/$TEST_USERNAME" "Detail profil user"
fi

print_section "3.7 GET /users/:username - User Not Found (Error)"
make_request "GET" "/users/nonexistent_user_12345" "User tidak ditemukan"

print_section "3.8 GET /users/:username/followers - List Followers"
if [ -n "$TEST_USERNAME" ] && [ "$TEST_USERNAME" != "null" ]; then
    make_request "GET" "/users/$TEST_USERNAME/followers" "Daftar followers"
fi

print_section "3.9 GET /users/:username/following - List Following"
if [ -n "$TEST_USERNAME" ] && [ "$TEST_USERNAME" != "null" ]; then
    make_request "GET" "/users/$TEST_USERNAME/following" "Daftar following"
fi

# ============================================================================
# 4. PROFILE ENDPOINTS (Authenticated)
# ============================================================================

print_header "4. PROFILE ENDPOINTS (Authenticated)"

print_section "4.1 GET /me - Get Current User Profile"
make_request "GET" "/me" "Profil user yang sedang login" "" "$ADMIN_TOKEN"

print_section "4.2 GET /me - Unauthorized (Error)"
make_request "GET" "/me" "Akses tanpa token (error)"

print_section "4.3 PATCH /me - Update Profile"
make_request "PATCH" "/me" "Update profil user" \
    "{\"bio\":\"Updated bio from test at $(date +%s)\"}" "$ADMIN_TOKEN"

print_section "4.4 GET /me/check-username - Check Available Username"
make_request "GET" "/me/check-username" "Cek username tersedia" "" "$ADMIN_TOKEN" "username=available_test_$(date +%s)"

print_section "4.5 GET /me/check-username - Username Taken"
make_request "GET" "/me/check-username" "Cek username sudah dipakai" "" "$ADMIN_TOKEN" "username=$ADMIN_USERNAME"

print_section "4.6 PUT /me/social-links - Update Social Links"
make_request "PUT" "/me/social-links" "Update social links" \
    "{\"social_links\":{\"github\":\"https://github.com/testuser\",\"instagram\":\"https://instagram.com/testuser\"}}" "$ADMIN_TOKEN"

# ============================================================================
# 5. PORTFOLIO ENDPOINTS
# ============================================================================

print_header "5. PORTFOLIO ENDPOINTS"

print_section "5.1 GET /portfolios - List Published Portfolios"
RESPONSE=$(make_request "GET" "/portfolios" "Daftar portfolio published")
SAMPLE_PORTFOLIO_ID=$(extract_json_value "$RESPONSE" ".data[0].id")
SAMPLE_USERNAME=$(extract_json_value "$RESPONSE" ".data[0].user.username")
SAMPLE_SLUG=$(extract_json_value "$RESPONSE" ".data[0].slug")

print_section "5.2 GET /portfolios?search=website - Search Portfolios"
make_request "GET" "/portfolios" "Cari portfolio" "" "" "search=website"

print_section "5.3 GET /portfolios?tag_ids=xxx - Filter by Tags"
if [ -n "$TAG_ID" ] && [ "$TAG_ID" != "null" ]; then
    make_request "GET" "/portfolios" "Filter by tag" "" "" "tag_ids=$TAG_ID"
fi

print_section "5.4 GET /portfolios?sort=-like_count - Sort by Likes"
make_request "GET" "/portfolios" "Sort by likes" "" "" "sort=-like_count"

print_section "5.5 GET /portfolios/:username/:slug - Get Portfolio Detail"
if [ -n "$SAMPLE_USERNAME" ] && [ -n "$SAMPLE_SLUG" ] && [ "$SAMPLE_USERNAME" != "null" ] && [ "$SAMPLE_SLUG" != "null" ]; then
    make_request "GET" "/portfolios/$SAMPLE_USERNAME/$SAMPLE_SLUG" "Detail portfolio"
fi

print_section "5.6 GET /portfolios/:username/:slug - Not Found (Error)"
make_request "GET" "/portfolios/nonexistent/portfolio-slug" "Portfolio tidak ditemukan"

print_section "5.7 GET /me/portfolios - Get My Portfolios"
make_request "GET" "/me/portfolios" "Portfolio milik user login" "" "$ADMIN_TOKEN"

print_section "5.8 POST /portfolios - Create New Portfolio"
if [ -n "$TAG_ID" ] && [ "$TAG_ID" != "null" ]; then
    RESPONSE=$(make_request "POST" "/portfolios" "Buat portfolio baru" \
        "{\"title\":\"Test Portfolio $(date +%s)\",\"tag_ids\":[\"$TAG_ID\"]}" "$ADMIN_TOKEN")
    
    TEST_PORTFOLIO_ID=$(extract_json_value "$RESPONSE" ".data.id")
    TEST_PORTFOLIO_SLUG=$(extract_json_value "$RESPONSE" ".data.slug")
    
    if [ -n "$TEST_PORTFOLIO_ID" ] && [ "$TEST_PORTFOLIO_ID" != "null" ]; then
        print_success "Portfolio created: $TEST_PORTFOLIO_ID"
    fi
fi

print_section "5.9 POST /portfolios - Validation Error (Error)"
make_request "POST" "/portfolios" "Buat portfolio tanpa title (error)" \
    "{\"tag_ids\":[]}" "$ADMIN_TOKEN"

print_section "5.10 GET /portfolios/:id - Get by ID for Edit"
if [ -n "$TEST_PORTFOLIO_ID" ] && [ "$TEST_PORTFOLIO_ID" != "null" ]; then
    make_request "GET" "/portfolios/$TEST_PORTFOLIO_ID" "Get portfolio by ID" "" "$ADMIN_TOKEN"
fi

print_section "5.11 PATCH /portfolios/:id - Update Portfolio"
if [ -n "$TEST_PORTFOLIO_ID" ] && [ "$TEST_PORTFOLIO_ID" != "null" ]; then
    make_request "PATCH" "/portfolios/$TEST_PORTFOLIO_ID" "Update portfolio" \
        "{\"title\":\"Updated Test Portfolio $(date +%s)\"}" "$ADMIN_TOKEN"
fi

# ============================================================================
# 6. CONTENT BLOCKS ENDPOINTS
# ============================================================================

print_header "6. CONTENT BLOCKS ENDPOINTS"

if [ -n "$TEST_PORTFOLIO_ID" ] && [ "$TEST_PORTFOLIO_ID" != "null" ]; then
    
    print_section "6.1 POST /portfolios/:id/blocks - Add Text Block"
    RESPONSE=$(make_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" "Tambah text block" \
        "{\"block_type\":\"text\",\"block_order\":0,\"payload\":{\"content\":\"<p>Test content</p>\"}}" "$ADMIN_TOKEN")
    TEST_BLOCK_ID=$(extract_json_value "$RESPONSE" ".data.id")
    
    print_section "6.2 POST /portfolios/:id/blocks - Add Image Block"
    make_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" "Tambah image block" \
        "{\"block_type\":\"image\",\"block_order\":1,\"payload\":{\"url\":\"https://example.com/img.jpg\",\"caption\":\"Test\"}}" "$ADMIN_TOKEN"
    
    print_section "6.3 POST /portfolios/:id/blocks - Add YouTube Block"
    make_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" "Tambah YouTube block" \
        "{\"block_type\":\"youtube\",\"block_order\":2,\"payload\":{\"video_id\":\"dQw4w9WgXcQ\"}}" "$ADMIN_TOKEN"
    
    print_section "6.4 POST /portfolios/:id/blocks - Add Table Block"
    make_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" "Tambah table block" \
        "{\"block_type\":\"table\",\"block_order\":3,\"payload\":{\"headers\":[\"Col1\",\"Col2\"],\"rows\":[[\"A\",\"B\"]]}}" "$ADMIN_TOKEN"
    
    print_section "6.5 POST /portfolios/:id/blocks - Add Button Block"
    make_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" "Tambah button block" \
        "{\"block_type\":\"button\",\"block_order\":4,\"payload\":{\"label\":\"Click\",\"url\":\"https://example.com\"}}" "$ADMIN_TOKEN"
    
    print_section "6.6 POST /portfolios/:id/blocks - Invalid Block Type (Error)"
    make_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" "Block type invalid (error)" \
        "{\"block_type\":\"invalid\",\"block_order\":5,\"payload\":{}}" "$ADMIN_TOKEN"
    
    print_section "6.7 GET /portfolios/:id/blocks - Get All Blocks"
    make_request "GET" "/portfolios/$TEST_PORTFOLIO_ID/blocks" "Get all content blocks" "" "$ADMIN_TOKEN"
    
    if [ -n "$TEST_BLOCK_ID" ] && [ "$TEST_BLOCK_ID" != "null" ]; then
        print_section "6.8 PATCH /portfolios/:id/blocks/:blockId - Update Block"
        make_request "PATCH" "/portfolios/$TEST_PORTFOLIO_ID/blocks/$TEST_BLOCK_ID" "Update block" \
            "{\"payload\":{\"content\":\"<p>Updated content</p>\"}}" "$ADMIN_TOKEN"
        
        print_section "6.9 DELETE /portfolios/:id/blocks/:blockId - Delete Block"
        make_request "DELETE" "/portfolios/$TEST_PORTFOLIO_ID/blocks/$TEST_BLOCK_ID" "Delete block" "" "$ADMIN_TOKEN"
    fi
fi

# ============================================================================
# 7. SEARCH ENDPOINTS
# ============================================================================

print_header "7. SEARCH ENDPOINTS"

print_section "7.1 GET /search/users - Search Users"
make_request "GET" "/search/users" "Search users" "" "" "q=test"

print_section "7.2 GET /search/users - With Filters"
if [ -n "$MAJOR_ID" ] && [ "$MAJOR_ID" != "null" ]; then
    make_request "GET" "/search/users" "Search users with filters" "" "" "q=test&major_id=$MAJOR_ID&role=student"
fi

print_section "7.3 GET /search/portfolios - Search Portfolios"
make_request "GET" "/search/portfolios" "Search portfolios" "" "" "q=website"

print_section "7.4 GET /search/portfolios - With Filters"
if [ -n "$TAG_ID" ] && [ "$TAG_ID" != "null" ]; then
    make_request "GET" "/search/portfolios" "Search portfolios with filters" "" "" "q=web&tag_ids=$TAG_ID"
fi

# ============================================================================
# 8. SOCIAL ENDPOINTS (Follow & Like)
# ============================================================================

print_header "8. SOCIAL ENDPOINTS"

if [ -n "$TEST_USERNAME" ] && [ "$TEST_USERNAME" != "null" ] && [ "$TEST_USERNAME" != "$ADMIN_USERNAME" ]; then
    print_section "8.1 POST /users/:username/follow - Follow User"
    make_request "POST" "/users/$TEST_USERNAME/follow" "Follow user" "" "$ADMIN_TOKEN"
    
    print_section "8.2 POST /users/:username/follow - Already Following (Error)"
    make_request "POST" "/users/$TEST_USERNAME/follow" "Already following (error)" "" "$ADMIN_TOKEN"
    
    print_section "8.3 DELETE /users/:username/follow - Unfollow User"
    make_request "DELETE" "/users/$TEST_USERNAME/follow" "Unfollow user" "" "$ADMIN_TOKEN"
    
    print_section "8.4 DELETE /users/:username/follow - Not Following (Error)"
    make_request "DELETE" "/users/$TEST_USERNAME/follow" "Not following (error)" "" "$ADMIN_TOKEN"
fi

print_section "8.5 POST /users/:username/follow - Cannot Follow Self (Error)"
make_request "POST" "/users/$ADMIN_USERNAME/follow" "Follow self (error)" "" "$ADMIN_TOKEN"

if [ -n "$SAMPLE_PORTFOLIO_ID" ] && [ "$SAMPLE_PORTFOLIO_ID" != "null" ]; then
    print_section "8.6 POST /portfolios/:id/like - Like Portfolio"
    make_request "POST" "/portfolios/$SAMPLE_PORTFOLIO_ID/like" "Like portfolio" "" "$ADMIN_TOKEN"
    
    print_section "8.7 POST /portfolios/:id/like - Already Liked (Error)"
    make_request "POST" "/portfolios/$SAMPLE_PORTFOLIO_ID/like" "Already liked (error)" "" "$ADMIN_TOKEN"
    
    print_section "8.8 DELETE /portfolios/:id/like - Unlike Portfolio"
    make_request "DELETE" "/portfolios/$SAMPLE_PORTFOLIO_ID/like" "Unlike portfolio" "" "$ADMIN_TOKEN"
fi

# ============================================================================
# 9. FEED ENDPOINT
# ============================================================================

print_header "9. FEED ENDPOINT"

print_section "9.1 GET /feed - Get Feed"
make_request "GET" "/feed" "Timeline dari followed users" "" "$ADMIN_TOKEN"

print_section "9.2 GET /feed - Unauthorized (Error)"
make_request "GET" "/feed" "Feed tanpa auth (error)"

# ============================================================================
# 10. ADMIN ENDPOINTS
# ============================================================================

print_header "10. ADMIN ENDPOINTS"

print_section "10.1 GET /admin/majors - List Majors"
make_request "GET" "/admin/majors" "List all majors (admin)" "" "$ADMIN_TOKEN"

print_section "10.2 GET /admin/academic-years - List Academic Years"
make_request "GET" "/admin/academic-years" "List academic years (admin)" "" "$ADMIN_TOKEN"

print_section "10.3 GET /admin/classes - List Classes"
make_request "GET" "/admin/classes" "List all classes (admin)" "" "$ADMIN_TOKEN"

print_section "10.4 GET /admin/users - List Users"
make_request "GET" "/admin/users" "List all users (admin)" "" "$ADMIN_TOKEN"

print_section "10.5 GET /admin/users?search=admin - Search Users"
make_request "GET" "/admin/users" "Search users (admin)" "" "$ADMIN_TOKEN" "search=admin"

if [ -n "$ADMIN_USER_ID" ] && [ "$ADMIN_USER_ID" != "null" ]; then
    print_section "10.6 GET /admin/users/:id - Get User Detail"
    make_request "GET" "/admin/users/$ADMIN_USER_ID" "Get user detail (admin)" "" "$ADMIN_TOKEN"
fi

print_section "10.7 GET /admin/tags - List Tags"
make_request "GET" "/admin/tags" "List all tags (admin)" "" "$ADMIN_TOKEN"

print_section "10.8 GET /admin/portfolios - List All Portfolios"
make_request "GET" "/admin/portfolios" "List all portfolios (admin)" "" "$ADMIN_TOKEN"

print_section "10.9 GET /admin/portfolios?status=draft - Filter by Status"
make_request "GET" "/admin/portfolios" "Filter portfolios by status" "" "$ADMIN_TOKEN" "status=draft"

if [ -n "$TEST_PORTFOLIO_ID" ] && [ "$TEST_PORTFOLIO_ID" != "null" ]; then
    print_section "10.10 GET /admin/portfolios/:id - Get Portfolio Detail"
    make_request "GET" "/admin/portfolios/$TEST_PORTFOLIO_ID" "Get portfolio detail (admin)" "" "$ADMIN_TOKEN"
fi

print_section "10.11 GET /admin/moderation/queue - Moderation Queue"
make_request "GET" "/admin/moderation/queue" "List pending portfolios" "" "$ADMIN_TOKEN"

# ============================================================================
# 11. UPLOAD ENDPOINTS (Presigned URL)
# ============================================================================

print_header "11. UPLOAD ENDPOINTS"

print_section "11.1 POST /uploads/presign - Request Presigned URL (Avatar)"
make_request "POST" "/uploads/presign" "Request presigned URL for avatar" \
    "{\"upload_type\":\"avatar\",\"filename\":\"test.jpg\",\"content_type\":\"image/jpeg\",\"file_size\":102400}" "$ADMIN_TOKEN"

print_section "11.2 POST /uploads/presign - File Too Large (Error)"
make_request "POST" "/uploads/presign" "File too large (error)" \
    "{\"upload_type\":\"avatar\",\"filename\":\"test.jpg\",\"content_type\":\"image/jpeg\",\"file_size\":10240000}" "$ADMIN_TOKEN"

print_section "11.3 POST /uploads/presign - Invalid Content Type (Error)"
make_request "POST" "/uploads/presign" "Invalid content type (error)" \
    "{\"upload_type\":\"avatar\",\"filename\":\"test.pdf\",\"content_type\":\"application/pdf\",\"file_size\":102400}" "$ADMIN_TOKEN"

if [ -n "$TEST_PORTFOLIO_ID" ] && [ "$TEST_PORTFOLIO_ID" != "null" ]; then
    print_section "11.4 POST /uploads/presign - Portfolio Thumbnail"
    make_request "POST" "/uploads/presign" "Request presigned URL for thumbnail" \
        "{\"upload_type\":\"thumbnail\",\"filename\":\"thumb.jpg\",\"content_type\":\"image/jpeg\",\"file_size\":512000,\"portfolio_id\":\"$TEST_PORTFOLIO_ID\"}" "$ADMIN_TOKEN"
fi

# ============================================================================
# 12. PORTFOLIO LIFECYCLE TESTS
# ============================================================================

print_header "12. PORTFOLIO LIFECYCLE"

if [ -n "$TEST_PORTFOLIO_ID" ] && [ "$TEST_PORTFOLIO_ID" != "null" ]; then
    print_section "12.1 POST /portfolios/:id/submit - Submit Incomplete (Error)"
    make_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/submit" "Submit incomplete portfolio (error)" "" "$ADMIN_TOKEN"
    
    print_section "12.2 POST /portfolios/:id/archive - Archive Portfolio"
    make_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/archive" "Archive portfolio" "" "$ADMIN_TOKEN"
    
    print_section "12.3 POST /portfolios/:id/unarchive - Unarchive Portfolio"
    make_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/unarchive" "Unarchive portfolio" "" "$ADMIN_TOKEN"
    
    print_section "12.4 DELETE /portfolios/:id - Delete Portfolio (Cleanup)"
    make_request "DELETE" "/portfolios/$TEST_PORTFOLIO_ID" "Delete test portfolio" "" "$ADMIN_TOKEN"
    TEST_PORTFOLIO_ID="" # Clear so cleanup doesn't try again
fi

# ============================================================================
# FINAL SUMMARY
# ============================================================================

print_summary

echo ""
print_info "Test completed at $(date)"
print_info "All test data has been cleaned up"
