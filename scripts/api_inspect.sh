#!/bin/bash

# =============================================================================
# Grafikarsa API Inspector (Bash) - Shows raw JSON request/response
# =============================================================================
# Usage: ./scripts/api_inspect.sh
#        ./scripts/api_inspect.sh http://localhost:8080
#        ./scripts/api_inspect.sh http://localhost:8080 /users
#
# Requires: curl, jq
# =============================================================================

set -euo pipefail

BASE_URL="${1:-http://localhost:8080}"
ENDPOINT="${2:-}"
API_URL="$BASE_URL/api/v1"

ACCESS_TOKEN=""

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
MAGENTA='\033[0;35m'
GRAY='\033[0;37m'
NC='\033[0m'

# Log file
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
LOG_DIR="$SCRIPT_DIR/log"
mkdir -p "$LOG_DIR"

LOG_FILE="$LOG_DIR/log.md"
if [ -f "$LOG_FILE" ]; then
    counter=1
    while [ -f "$LOG_DIR/log${counter}.md" ]; do
        counter=$((counter + 1))
    done
    LOG_FILE="$LOG_DIR/log${counter}.md"
fi

LOG_CONTENT=""

# Check dependencies
if ! command -v jq &> /dev/null; then
    echo "❌ jq is required. Install: apt install jq / brew install jq"
    exit 1
fi

log() {
    local message="$1"
    local color="${2:-$NC}"
    echo -e "${color}${message}${NC}"
    LOG_CONTENT="${LOG_CONTENT}${message}\n"
}

write_header() {
    log "" "$CYAN"
    log "$(printf '=%.0s' {1..80})" "$CYAN"
    log "  $1" "$CYAN"
    log "$(printf '=%.0s' {1..80})" "$CYAN"
}

write_subheader() {
    log "" "$GRAY"
    log "$(printf -- '-%.0s' {1..60})" "$GRAY"
    log "  $1" "$YELLOW"
    log "$(printf -- '-%.0s' {1..60})" "$GRAY"
}

api_inspect() {
    local method="$1"
    local endpoint="$2"
    local body="${3:-}"
    local use_auth="${4:-false}"

    local url="${API_URL}${endpoint}"
    local curl_args=(-s -w "\n%{http_code}" -X "$method" -H "Content-Type: application/json")

    if [ "$use_auth" = "true" ] && [ -n "$ACCESS_TOKEN" ]; then
        curl_args+=(-H "Authorization: Bearer $ACCESS_TOKEN")
    fi

    write_subheader "$method $endpoint"

    log "" "$GREEN"
    log "REQUEST:" "$GREEN"
    log "  URL: $url" "$GRAY"
    log "  Method: $method" "$GRAY"
    if [ "$use_auth" = "true" ] && [ -n "$ACCESS_TOKEN" ]; then
        log "  Authorization: Bearer <token>" "$GRAY"
    fi

    if [ -n "$body" ] && [ "$method" != "GET" ]; then
        log "  Body:" "$GRAY"
        log "$(echo "$body" | jq '.' 2>/dev/null || echo "$body")" "$YELLOW"
        curl_args+=(-d "$body")
    fi

    local response
    response=$(curl "${curl_args[@]}" "$url" 2>/dev/null || echo -e "\n000")

    local http_code
    http_code=$(echo "$response" | tail -1)
    local response_body
    response_body=$(echo "$response" | sed '$d')

    log "" "$GREEN"
    log "RESPONSE:" "$GREEN"

    if [ "$http_code" -lt 400 ] 2>/dev/null; then
        log "  Status: $http_code" "$GREEN"
    else
        log "  Status: $http_code" "$RED"
    fi

    if [ -n "$response_body" ]; then
        log "  Body:" "$GRAY"
        log "$(echo "$response_body" | jq '.' 2>/dev/null || echo "$response_body")"
    fi

    echo "$http_code|$response_body"
}

do_login() {
    local result
    result=$(api_inspect "POST" "/auth/login" '{"username":"admin","password":"password"}')
    local body
    body=$(echo "$result" | cut -d'|' -f2-)
    ACCESS_TOKEN=$(echo "$body" | jq -r '.data.access_token // empty' 2>/dev/null)
    if [ -n "$ACCESS_TOKEN" ]; then
        log "" "$GREEN"
        log "  [Logged in successfully, token saved]" "$GREEN"
    fi
}

inspect_all() {
    write_header "GRAFIKARSA API INSPECTOR"
    log "Base URL: $API_URL" "$MAGENTA"
    log "Timestamp: $(date '+%Y-%m-%d %H:%M:%S')" "$MAGENTA"

    # Check server
    if ! curl -s -o /dev/null "$API_URL/health" 2>/dev/null; then
        log "ERROR: Cannot connect to API server" "$RED"
        exit 1
    fi

    # PUBLIC
    write_header "PUBLIC ENDPOINTS"
    api_inspect "GET" "/jurusan" > /dev/null
    api_inspect "GET" "/kelas" > /dev/null
    api_inspect "GET" "/tags" > /dev/null
    api_inspect "GET" "/users?limit=3" > /dev/null
    api_inspect "GET" "/portfolios?limit=3" > /dev/null
    api_inspect "GET" "/users/admin" > /dev/null
    api_inspect "GET" "/search/users?q=admin" > /dev/null
    api_inspect "GET" "/search/portfolios?q=web" > /dev/null

    # AUTH
    write_header "AUTHENTICATION"
    api_inspect "POST" "/auth/login" '{"username":"invalid","password":"wrong"}' > /dev/null
    do_login

    if [ -n "$ACCESS_TOKEN" ]; then
        write_header "PROFILE (AUTHENTICATED)"
        api_inspect "GET" "/me" "" "true" > /dev/null
        api_inspect "GET" "/me/check-username?username=testuser123" "" "true" > /dev/null
        api_inspect "GET" "/me/portfolios?limit=3" "" "true" > /dev/null

        write_header "FEED"
        api_inspect "GET" "/feed?limit=3" "" "true" > /dev/null

        write_header "SESSIONS"
        api_inspect "GET" "/auth/sessions" "" "true" > /dev/null

        write_header "ADMIN ENDPOINTS"
        api_inspect "GET" "/admin/dashboard/stats" "" "true" > /dev/null
        api_inspect "GET" "/admin/users?limit=3" "" "true" > /dev/null

        write_header "ADMIN JURUSAN CRUD"
        api_inspect "GET" "/admin/jurusan" "" "true" > /dev/null

        local create_result
        create_result=$(api_inspect "POST" "/admin/jurusan" '{"nama":"Test Jurusan Inspect","kode":"testinsp"}' "true")
        local jurusan_id
        jurusan_id=$(echo "$create_result" | cut -d'|' -f2- | jq -r '.data.id // empty' 2>/dev/null)
        if [ -n "$jurusan_id" ]; then
            api_inspect "PATCH" "/admin/jurusan/$jurusan_id" '{"nama":"Test Jurusan Updated"}' "true" > /dev/null
            api_inspect "DELETE" "/admin/jurusan/$jurusan_id" "" "true" > /dev/null
        fi

        write_header "ADMIN PORTFOLIOS"
        api_inspect "GET" "/admin/portfolios?limit=3" "" "true" > /dev/null
        api_inspect "GET" "/admin/portfolios/pending?limit=3" "" "true" > /dev/null

        write_header "PORTFOLIO CRUD DEMO"
        local portfolio_result
        portfolio_result=$(api_inspect "POST" "/portfolios" '{"judul":"Demo Portfolio untuk Inspect"}' "true")
        local portfolio_id
        portfolio_id=$(echo "$portfolio_result" | cut -d'|' -f2- | jq -r '.data.id // empty' 2>/dev/null)

        if [ -n "$portfolio_id" ]; then
            api_inspect "GET" "/portfolios/id/$portfolio_id" "" "true" > /dev/null
            api_inspect "PATCH" "/portfolios/$portfolio_id" '{"judul":"Demo Portfolio Updated"}' "true" > /dev/null

            write_header "CONTENT BLOCKS - TEXT & IMAGE"
            api_inspect "POST" "/portfolios/$portfolio_id/blocks" '{"block_type":"text","block_order":0,"payload":{"content":"<p>Ini content block text</p>"}}' "true" > /dev/null
            api_inspect "POST" "/portfolios/$portfolio_id/blocks" '{"block_type":"image","block_order":1,"payload":{"url":"https://picsum.photos/800/600","caption":"Screenshot"}}' "true" > /dev/null

            write_header "PORTFOLIO STATUS"
            api_inspect "POST" "/portfolios/$portfolio_id/archive" "" "true" > /dev/null
            api_inspect "POST" "/portfolios/$portfolio_id/unarchive" "" "true" > /dev/null
            api_inspect "DELETE" "/portfolios/$portfolio_id" "" "true" > /dev/null
        fi

        write_header "LOGOUT"
        api_inspect "POST" "/auth/logout" "" "true" > /dev/null
    fi

    write_header "INSPECTION COMPLETE"
}

inspect_single() {
    local path="$1"
    write_header "SINGLE ENDPOINT INSPECTION"

    # Try login first
    local login_result
    login_result=$(api_inspect "POST" "/auth/login" '{"username":"admin","password":"password"}')
    ACCESS_TOKEN=$(echo "$login_result" | cut -d'|' -f2- | jq -r '.data.access_token // empty' 2>/dev/null)

    local use_auth="false"
    [ -n "$ACCESS_TOKEN" ] && use_auth="true"

    api_inspect "GET" "$path" "" "$use_auth" > /dev/null
}

# MAIN
if [ -z "$ENDPOINT" ]; then
    inspect_all
else
    inspect_single "$ENDPOINT"
fi

# Save log
cat > "$LOG_FILE" << EOF
# Grafikarsa API Inspection Log

Generated: $(date '+%Y-%m-%d %H:%M:%S')

\`\`\`
$(echo -e "$LOG_CONTENT")
\`\`\`
EOF

echo ""
echo -e "${GREEN}Log saved to: $LOG_FILE${NC}"
