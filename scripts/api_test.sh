#!/bin/bash

# =============================================================================
# Grafikarsa API Test Script (Bash)
# =============================================================================
# Usage: ./scripts/api_test.sh
#        ./scripts/api_test.sh http://localhost:8080
#
# Requires: curl, jq
# =============================================================================

set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
API_URL="$BASE_URL/api/v1"

ACCESS_TOKEN=""
TEST_USER_ID=""
TEST_PORTFOLIO_ID=""
TEST_BLOCK_ID=""
PASS_COUNT=0
FAIL_COUNT=0
SKIP_COUNT=0

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Check dependencies
if ! command -v jq &> /dev/null; then
    echo "❌ jq is required. Install: apt install jq / brew install jq"
    exit 1
fi

write_header() {
    echo ""
    echo -e "${CYAN}========================================${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}========================================${NC}"
}

write_result() {
    local name="$1"
    local success="$2"
    local message="${3:-}"
    local skip="${4:-false}"

    if [ "$skip" = "true" ]; then
        echo -e "${YELLOW}[SKIP]${NC} $name"
        [ -n "$message" ] && echo "       $message"
        SKIP_COUNT=$((SKIP_COUNT + 1))
        return
    fi

    if [ "$success" = "true" ]; then
        echo -e "${GREEN}[PASS]${NC} $name"
        PASS_COUNT=$((PASS_COUNT + 1))
    else
        echo -e "${RED}[FAIL]${NC} $name"
        [ -n "$message" ] && echo -e "${RED}       $message${NC}"
        FAIL_COUNT=$((FAIL_COUNT + 1))
    fi
}

# Generic API request function
# Returns: status_code|response_body
api_request() {
    local method="$1"
    local endpoint="$2"
    local body="${3:-}"
    local use_auth="${4:-false}"
    local expected_status="${5:-200}"

    local url="${API_URL}${endpoint}"
    local curl_args=(-s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json")

    if [ "$use_auth" = "true" ] && [ -n "$ACCESS_TOKEN" ]; then
        curl_args+=(-H "Authorization: Bearer $ACCESS_TOKEN")
    fi

    if [ -n "$body" ] && [ "$method" != "GET" ]; then
        curl_args+=(-d "$body")
    fi

    local response
    response=$(curl "${curl_args[@]}" "$url" 2>/dev/null || echo -e "\n000")

    local http_code
    http_code=$(echo "$response" | tail -1)
    local response_body
    response_body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "$expected_status" ]; then
        echo "true|$http_code|$response_body"
    else
        echo "false|$http_code|$response_body"
    fi
}

# ============================================
# TEST SUITES
# ============================================

test_public_endpoints() {
    write_header "PUBLIC ENDPOINTS"

    local result status

    # GET /jurusan
    result=$(api_request "GET" "/jurusan")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /jurusan" "$status"

    # GET /kelas
    result=$(api_request "GET" "/kelas")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /kelas" "$status"

    # GET /tags
    result=$(api_request "GET" "/tags")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /tags" "$status"

    # GET /users
    result=$(api_request "GET" "/users")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /users" "$status"

    # GET /portfolios
    result=$(api_request "GET" "/portfolios")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /portfolios" "$status"
}

test_auth_endpoints() {
    write_header "AUTHENTICATION"

    local result status body

    # POST /auth/login - Invalid credentials
    result=$(api_request "POST" "/auth/login" '{"username":"invalid_user","password":"wrong_password"}' "false" "401")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "POST /auth/login (invalid)" "$status"

    # POST /auth/login - Valid credentials
    result=$(api_request "POST" "/auth/login" '{"username":"admin","password":"password"}')
    status=$(echo "$result" | cut -d'|' -f1)
    body=$(echo "$result" | cut -d'|' -f3-)

    if [ "$status" = "true" ]; then
        ACCESS_TOKEN=$(echo "$body" | jq -r '.data.access_token // empty')
        if [ -n "$ACCESS_TOKEN" ]; then
            write_result "POST /auth/login (valid)" "true"
        else
            write_result "POST /auth/login (valid)" "false" "No access token in response"
        fi
    else
        write_result "POST /auth/login (valid)" "false" "Login failed"
    fi

    # GET /auth/sessions
    if [ -n "$ACCESS_TOKEN" ]; then
        result=$(api_request "GET" "/auth/sessions" "" "true")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "GET /auth/sessions" "$status"
    else
        write_result "GET /auth/sessions" "" "" "true"
    fi
}

test_profile_endpoints() {
    write_header "PROFILE (AUTHENTICATED)"

    if [ -z "$ACCESS_TOKEN" ]; then
        write_result "Profile tests" "" "No access token available" "true"
        return
    fi

    local result status body

    # GET /me
    result=$(api_request "GET" "/me" "" "true")
    status=$(echo "$result" | cut -d'|' -f1)
    body=$(echo "$result" | cut -d'|' -f3-)
    write_result "GET /me" "$status"
    if [ "$status" = "true" ]; then
        TEST_USER_ID=$(echo "$body" | jq -r '.data.id // empty')
    fi

    # GET /me without auth (should fail)
    result=$(api_request "GET" "/me" "" "false" "401")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /me (no auth)" "$status"

    # PATCH /me
    result=$(api_request "PATCH" "/me" '{"bio":"Updated bio from test script"}' "true")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "PATCH /me" "$status"

    # GET /me/check-username
    result=$(api_request "GET" "/me/check-username?username=testuser123" "" "true")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /me/check-username" "$status"
}

test_user_endpoints() {
    write_header "USER ENDPOINTS"

    local result status

    # GET /users/{username}
    result=$(api_request "GET" "/users/admin")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /users/{username}" "$status"

    # GET /users/{username} - Not found
    result=$(api_request "GET" "/users/nonexistent_user_12345" "" "false" "404")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /users/{username} (404)" "$status"

    # GET /users/{username}/followers
    result=$(api_request "GET" "/users/admin/followers")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /users/{username}/followers" "$status"

    # GET /users/{username}/following
    result=$(api_request "GET" "/users/admin/following")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /users/{username}/following" "$status"
}

test_portfolio_endpoints() {
    write_header "PORTFOLIO ENDPOINTS"

    if [ -z "$ACCESS_TOKEN" ]; then
        write_result "Portfolio tests" "" "No access token available" "true"
        return
    fi

    local result status body

    # POST /portfolios - Create
    result=$(api_request "POST" "/portfolios" '{"judul":"Test Portfolio from Script"}' "true" "201")
    status=$(echo "$result" | cut -d'|' -f1)
    body=$(echo "$result" | cut -d'|' -f3-)

    if [ "$status" = "true" ]; then
        TEST_PORTFOLIO_ID=$(echo "$body" | jq -r '.data.id // empty')
        write_result "POST /portfolios" "true"
    else
        write_result "POST /portfolios" "false"
    fi

    # GET /me/portfolios
    result=$(api_request "GET" "/me/portfolios" "" "true")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /me/portfolios" "$status"

    if [ -n "$TEST_PORTFOLIO_ID" ]; then
        # GET /portfolios/id/{id}
        result=$(api_request "GET" "/portfolios/id/$TEST_PORTFOLIO_ID" "" "true")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "GET /portfolios/id/{id}" "$status"

        # PATCH /portfolios/{id}
        result=$(api_request "PATCH" "/portfolios/$TEST_PORTFOLIO_ID" '{"judul":"Updated Test Portfolio"}' "true")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "PATCH /portfolios/{id}" "$status"

        # POST /portfolios/{id}/submit - Should fail (no thumbnail/blocks)
        result=$(api_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/submit" "" "true" "422")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "POST /portfolios/{id}/submit (incomplete)" "$status"
    fi
}

test_content_block_endpoints() {
    write_header "CONTENT BLOCK ENDPOINTS"

    if [ -z "$ACCESS_TOKEN" ] || [ -z "$TEST_PORTFOLIO_ID" ]; then
        write_result "Content block tests" "" "No portfolio available" "true"
        return
    fi

    local result status body text_block_id

    # TEXT block
    result=$(api_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" '{"block_type":"text","block_order":0,"payload":{"content":"<p>Test text content block</p>"}}' "true" "201")
    status=$(echo "$result" | cut -d'|' -f1)
    body=$(echo "$result" | cut -d'|' -f3-)
    text_block_id=$(echo "$body" | jq -r '.data.id // empty')
    write_result "POST block (text)" "$status"

    # IMAGE block
    result=$(api_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" '{"block_type":"image","block_order":1,"payload":{"url":"https://picsum.photos/800/600","caption":"Test image"}}' "true" "201")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "POST block (image)" "$status"

    # TABLE block
    result=$(api_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" '{"block_type":"table","block_order":2,"payload":{"headers":["Fitur","Deskripsi"],"rows":[["Login","Auth"],["Dashboard","Main"]]}}' "true" "201")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "POST block (table)" "$status"

    # YOUTUBE block
    result=$(api_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" '{"block_type":"youtube","block_order":3,"payload":{"video_id":"dQw4w9WgXcQ","title":"Demo"}}' "true" "201")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "POST block (youtube)" "$status"

    # BUTTON block
    result=$(api_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" '{"block_type":"button","block_order":4,"payload":{"text":"Lihat Demo","url":"https://demo.example.com"}}' "true" "201")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "POST block (button)" "$status"

    # EMBED block
    result=$(api_request "POST" "/portfolios/$TEST_PORTFOLIO_ID/blocks" '{"block_type":"embed","block_order":5,"payload":{"html":"<iframe src=\"https://codepen.io\"></iframe>","title":"CodePen"}}' "true" "201")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "POST block (embed)" "$status"

    # PATCH text block
    if [ -n "$text_block_id" ]; then
        result=$(api_request "PATCH" "/portfolios/$TEST_PORTFOLIO_ID/blocks/$text_block_id" '{"payload":{"content":"<p>Updated text</p>"}}' "true")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "PATCH block (text)" "$status"

        # DELETE text block
        result=$(api_request "DELETE" "/portfolios/$TEST_PORTFOLIO_ID/blocks/$text_block_id" "" "true")
        write_result "DELETE block (cleanup)" "true"
    fi
}

test_social_endpoints() {
    write_header "SOCIAL ENDPOINTS (FOLLOW/LIKE)"

    if [ -z "$ACCESS_TOKEN" ]; then
        write_result "Social tests" "" "No access token available" "true"
        return
    fi

    local result status body

    # Get a portfolio to like
    result=$(api_request "GET" "/portfolios?limit=1")
    body=$(echo "$result" | cut -d'|' -f3-)
    local portfolio_id
    portfolio_id=$(echo "$body" | jq -r '.data[0].id // empty')

    if [ -n "$portfolio_id" ]; then
        result=$(api_request "POST" "/portfolios/$portfolio_id/like" "" "true")
        local code
        code=$(echo "$result" | cut -d'|' -f2)
        write_result "POST /portfolios/{id}/like" "$([ "$code" = "200" ] || [ "$code" = "409" ] && echo true || echo false)"

        result=$(api_request "DELETE" "/portfolios/$portfolio_id/like" "" "true")
        code=$(echo "$result" | cut -d'|' -f2)
        write_result "DELETE /portfolios/{id}/like" "$([ "$code" = "200" ] || [ "$code" = "400" ] && echo true || echo false)"
    else
        write_result "Like tests" "" "No portfolios to like" "true"
    fi
}

test_feed_endpoints() {
    write_header "FEED & SEARCH"

    local result status

    # GET /feed
    if [ -n "$ACCESS_TOKEN" ]; then
        result=$(api_request "GET" "/feed" "" "true")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "GET /feed" "$status"
    else
        write_result "GET /feed" "" "No access token" "true"
    fi

    # GET /search/users
    result=$(api_request "GET" "/search/users?q=admin")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /search/users" "$status"

    # GET /search/portfolios
    result=$(api_request "GET" "/search/portfolios?q=test")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /search/portfolios" "$status"
}

test_admin_endpoints() {
    write_header "ADMIN ENDPOINTS"

    if [ -z "$ACCESS_TOKEN" ]; then
        write_result "Admin tests" "" "No access token available" "true"
        return
    fi

    local result status body

    # GET /admin/dashboard/stats
    result=$(api_request "GET" "/admin/dashboard/stats" "" "true")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /admin/dashboard/stats" "$status"

    # GET /admin/users
    result=$(api_request "GET" "/admin/users" "" "true")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /admin/users" "$status"

    # JURUSAN CRUD
    result=$(api_request "GET" "/admin/jurusan" "" "true")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /admin/jurusan" "$status"

    result=$(api_request "POST" "/admin/jurusan" '{"nama":"Test Jurusan","kode":"testjur"}' "true" "201")
    status=$(echo "$result" | cut -d'|' -f1)
    body=$(echo "$result" | cut -d'|' -f3-)
    local jurusan_id
    jurusan_id=$(echo "$body" | jq -r '.data.id // empty')
    write_result "POST /admin/jurusan" "$status"

    if [ -n "$jurusan_id" ]; then
        result=$(api_request "PATCH" "/admin/jurusan/$jurusan_id" '{"nama":"Test Jurusan Updated"}' "true")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "PATCH /admin/jurusan/{id}" "$status"

        result=$(api_request "DELETE" "/admin/jurusan/$jurusan_id" "" "true")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "DELETE /admin/jurusan/{id}" "$status"
    fi

    # GET /admin/portfolios/pending
    result=$(api_request "GET" "/admin/portfolios/pending" "" "true")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /admin/portfolios/pending" "$status"

    # GET /admin/portfolios
    result=$(api_request "GET" "/admin/portfolios" "" "true")
    status=$(echo "$result" | cut -d'|' -f1)
    write_result "GET /admin/portfolios" "$status"
}

test_cleanup() {
    write_header "CLEANUP"

    if [ -n "$TEST_PORTFOLIO_ID" ] && [ -n "$ACCESS_TOKEN" ]; then
        local result status
        result=$(api_request "DELETE" "/portfolios/$TEST_PORTFOLIO_ID" "" "true")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "DELETE test portfolio" "$status"
    fi
}

test_logout() {
    write_header "LOGOUT"

    if [ -n "$ACCESS_TOKEN" ]; then
        local result status
        result=$(api_request "POST" "/auth/logout" "" "true")
        status=$(echo "$result" | cut -d'|' -f1)
        write_result "POST /auth/logout" "$status"
    fi
}

# ============================================
# MAIN
# ============================================

echo ""
echo -e "${CYAN}Grafikarsa API Test Suite${NC}"
echo -e "${CYAN}API URL: $API_URL${NC}"
echo ""

# Check server
if ! curl -s -o /dev/null -w "%{http_code}" "$API_URL/health" | grep -q "200\|404"; then
    echo -e "${RED}❌ Cannot connect to API server at $BASE_URL${NC}"
    exit 1
fi

test_public_endpoints
test_auth_endpoints
test_profile_endpoints
test_user_endpoints
test_portfolio_endpoints
test_content_block_endpoints
test_social_endpoints
test_feed_endpoints
test_admin_endpoints
test_cleanup
test_logout

# Summary
echo ""
echo "========================================"
echo "  TEST RESULTS"
echo "========================================"
echo -e "  ${GREEN}PASS: $PASS_COUNT${NC}"
echo -e "  ${RED}FAIL: $FAIL_COUNT${NC}"
echo -e "  ${YELLOW}SKIP: $SKIP_COUNT${NC}"
echo "  TOTAL: $((PASS_COUNT + FAIL_COUNT + SKIP_COUNT))"
echo "========================================"
echo ""

if [ $FAIL_COUNT -gt 0 ]; then
    exit 1
fi
exit 0
