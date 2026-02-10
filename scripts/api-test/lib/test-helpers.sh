#!/bin/bash

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color
BOLD='\033[1m'

# Test counters
TOTAL_TESTS=0
PASSED_TESTS=0
FAILED_TESTS=0

# Error tracking
declare -a FAILED_TEST_DETAILS=()

# Base URL
BASE_URL="${API_BASE_URL:-http://localhost:8080/api/v1}"

# Print colored output
print_header() {
    echo -e "\n${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${CYAN}  $1${NC}"
    echo -e "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_section() {
    echo -e "\n${BOLD}${MAGENTA}▶ $1${NC}\n"
}

print_test() {
    echo -e "${BOLD}${BLUE}TEST:${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓ PASS${NC} - $1"
    ((PASSED_TESTS++))
    ((TOTAL_TESTS++))
}

print_fail() {
    local message="$1"
    local endpoint="$2"
    local reason="$3"
    
    echo -e "${RED}✗ FAIL${NC} - $message"
    
    # Store failure details
    if [ -n "$endpoint" ]; then
        FAILED_TEST_DETAILS+=("${endpoint}|${message}|${reason}")
    fi
    
    ((FAILED_TESTS++))
    ((TOTAL_TESTS++))
}

print_info() {
    echo -e "${CYAN}ℹ INFO:${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠ WARNING:${NC} $1"
}

print_error() {
    echo -e "${RED}✗ ERROR:${NC} $1"
}

# Print request details
print_request() {
    local method=$1
    local path=$2
    local description=$3
    
    echo -e "${BOLD}${YELLOW}┌─ REQUEST${NC}"
    echo -e "${YELLOW}│${NC} ${BOLD}Method:${NC} $method"
    echo -e "${YELLOW}│${NC} ${BOLD}Path:${NC} $path"
    echo -e "${YELLOW}│${NC} ${BOLD}Description:${NC} $description"
    echo -e "${YELLOW}└─${NC}"
}

# Print request parameters
print_params() {
    local type=$1
    shift
    local params="$@"
    
    if [ -n "$params" ]; then
        echo -e "${BOLD}${YELLOW}┌─ ${type}${NC}"
        echo -e "${YELLOW}│${NC} $params"
        echo -e "${YELLOW}└─${NC}"
    fi
}

# Print JSON beautifully
print_json() {
    local label=$1
    local json=$2
    
    echo -e "${BOLD}${YELLOW}┌─ ${label}${NC}"
    echo "$json" | jq '.' 2>/dev/null || echo "$json"
    echo -e "${YELLOW}└─${NC}"
}

# Make HTTP request and print details
make_request() {
    local method=$1
    local path=$2
    local description=$3
    local data=$4
    local auth_token=$5
    local query_params=$6
    
    print_request "$method" "$path" "$description"
    
    # Print query parameters if any
    if [ -n "$query_params" ]; then
        print_params "QUERY PARAMS" "$query_params"
    fi
    
    # Print request body if any
    if [ -n "$data" ] && [ "$data" != "null" ]; then
        print_json "REQUEST BODY" "$data"
    fi
    
    # Build curl command
    local url="${BASE_URL}${path}"
    if [ -n "$query_params" ]; then
        url="${url}?${query_params}"
    fi
    
    local curl_cmd="curl -s -w '\n%{http_code}' -X $method '$url'"
    
    if [ -n "$auth_token" ]; then
        curl_cmd="$curl_cmd -H 'Authorization: Bearer $auth_token'"
    fi
    
    if [ -n "$data" ] && [ "$data" != "null" ]; then
        curl_cmd="$curl_cmd -H 'Content-Type: application/json' -d '$data'"
    fi
    
    # Execute request
    local response=$(eval $curl_cmd 2>&1)
    local exit_code=$?
    
    # Check if curl failed
    if [ $exit_code -ne 0 ]; then
        echo -e "${BOLD}${RED}┌─ ERROR${NC}"
        echo -e "${RED}│${NC} ${BOLD}Curl failed with exit code:${NC} $exit_code"
        echo -e "${RED}│${NC} ${BOLD}Error:${NC} $response"
        echo -e "${RED}└─${NC}"
        
        # Track error
        print_fail "Request failed" "$method $path" "Curl error: $response"
        echo ""
        return 1
    fi
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')
    
    # Print response
    echo -e "${BOLD}${GREEN}┌─ RESPONSE${NC}"
    echo -e "${GREEN}│${NC} ${BOLD}Status:${NC} $http_code"
    echo -e "${GREEN}└─${NC}"
    
    print_json "RESPONSE BODY" "$body"
    
    # Check for error status codes
    if [[ "$http_code" =~ ^5 ]]; then
        print_fail "Server error (5xx)" "$method $path" "HTTP $http_code - Server error"
    elif [ "$http_code" = "000" ]; then
        print_fail "Connection failed" "$method $path" "Could not connect to server"
    fi
    
    echo ""
    
    # Return response for further processing
    echo "$body"
}

# Test endpoint with expected status code
test_endpoint() {
    local method=$1
    local path=$2
    local description=$3
    local expected_status=$4
    local data=$5
    local auth_token=$6
    local query_params=$7
    
    print_test "$description"
    
    # Build curl command
    local url="${BASE_URL}${path}"
    if [ -n "$query_params" ]; then
        url="${url}?${query_params}"
    fi
    
    local curl_cmd="curl -s -w '\n%{http_code}' -X $method '$url'"
    
    if [ -n "$auth_token" ]; then
        curl_cmd="$curl_cmd -H 'Authorization: Bearer $auth_token'"
    fi
    
    if [ -n "$data" ] && [ "$data" != "null" ]; then
        curl_cmd="$curl_cmd -H 'Content-Type: application/json' -d '$data'"
    fi
    
    # Execute request
    local response=$(eval $curl_cmd 2>&1)
    local exit_code=$?
    
    # Check if curl failed
    if [ $exit_code -ne 0 ]; then
        print_fail "Request failed" "$method $path" "Curl error (exit code: $exit_code)"
        return 1
    fi
    
    local http_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | sed '$d')
    
    # Check status code
    if [ "$http_code" = "$expected_status" ]; then
        print_success "Expected status $expected_status, got $http_code"
    else
        local error_msg=$(echo "$body" | jq -r '.error.message // .message // "Unknown error"' 2>/dev/null || echo "Unknown error")
        local error_code=$(echo "$body" | jq -r '.error.code // "UNKNOWN"' 2>/dev/null || echo "UNKNOWN")
        
        print_fail "Expected status $expected_status, got $http_code" "$method $path" "Error: $error_code - $error_msg"
        print_json "Response" "$body"
    fi
    
    echo ""
}

# Print test summary
print_summary() {
    echo -e "\n${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${BOLD}${CYAN}  TEST SUMMARY${NC}"
    echo -e "${BOLD}${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    
    echo -e "${BOLD}Total Tests:${NC} $TOTAL_TESTS"
    echo -e "${GREEN}${BOLD}Passed:${NC} $PASSED_TESTS"
    echo -e "${RED}${BOLD}Failed:${NC} $FAILED_TESTS"
    
    # Show failed test details if any
    if [ $FAILED_TESTS -gt 0 ]; then
        echo -e "\n${BOLD}${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
        echo -e "${BOLD}${RED}  FAILED TESTS DETAILS${NC}"
        echo -e "${BOLD}${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
        
        local counter=1
        for detail in "${FAILED_TEST_DETAILS[@]}"; do
            IFS='|' read -r endpoint message reason <<< "$detail"
            
            echo -e "${RED}${BOLD}$counter.${NC} ${BOLD}Endpoint:${NC} ${YELLOW}$endpoint${NC}"
            echo -e "   ${BOLD}Test:${NC} $message"
            echo -e "   ${BOLD}Reason:${NC} $reason"
            echo ""
            
            ((counter++))
        done
        
        echo -e "${BOLD}${RED}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
    fi
    
    if [ $FAILED_TESTS -eq 0 ]; then
        echo -e "\n${GREEN}${BOLD}✓ ALL TESTS PASSED!${NC}\n"
    else
        echo -e "\n${RED}${BOLD}✗ SOME TESTS FAILED${NC}"
        echo -e "${YELLOW}Please check the details above for more information.${NC}\n"
    fi
}

# Extract value from JSON response
extract_json_value() {
    local json=$1
    local key=$2
    echo "$json" | jq -r "$key" 2>/dev/null
}

# Wait for user confirmation
wait_for_confirmation() {
    echo -e "\n${YELLOW}Press Enter to continue...${NC}"
    read
}
