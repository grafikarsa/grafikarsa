#!/bin/bash

# Load helper functions and tokens
source "$(dirname "$0")/lib/test-helpers.sh"
source /tmp/grafikarsa_test_tokens.sh 2>/dev/null || true
source /tmp/grafikarsa_test_ids.sh 2>/dev/null || true

print_header "SOCIAL & LIKES ENDPOINTS TESTING"

print_section "1. POST /users/:username/follow - Follow User"
if [ -n "$ADMIN_TOKEN" ] && [ -n "$TEST_USERNAME" ] && [ "$TEST_USERNAME" != "null" ] && [ "$TEST_USERNAME" != "$ADMIN_USERNAME" ]; then
    make_request "POST" "/users/$TEST_USERNAME/follow" "Follow user" "" "$ADMIN_TOKEN"
else
    print_warning "Token atau username tidak tersedia, skip test"
fi

print_section "2. POST /users/:username/follow - Already Following (Error Case)"
if [ -n "$ADMIN_TOKEN" ] && [ -n "$TEST_USERNAME" ] && [ "$TEST_USERNAME" != "null" ]; then
    make_request "POST" "/users/$TEST_USERNAME/follow" "Follow user yang sudah di-follow" "" "$ADMIN_TOKEN"
else
    print_warning "Token atau username tidak tersedia, skip test"
fi

print_section "3. POST /users/:username/follow - Cannot Follow Self (Error Case)"
if [ -n "$ADMIN_TOKEN" ] && [ -n "$ADMIN_USERNAME" ]; then
    make_request "POST" "/users/$ADMIN_USERNAME/follow" "Follow diri sendiri (error)" "" "$ADMIN_TOKEN"
else
    print_warning "Token tidak tersedia, skip test"
fi

print_section "4. POST /users/:username/follow - User Not Found (Error Case)"
if [ -n "$ADMIN_TOKEN" ]; then
    make_request "POST" "/users/nonexistent_user_12345/follow" "Follow user yang tidak ada" "" "$ADMIN_TOKEN"
else
    print_warning "Token tidak tersedia, skip test"
fi

print_section "5. DELETE /users/:username/follow - Unfollow User"
if [ -n "$ADMIN_TOKEN" ] && [ -n "$TEST_USERNAME" ] && [ "$TEST_USERNAME" != "null" ]; then
    make_request "DELETE" "/users/$TEST_USERNAME/follow" "Unfollow user" "" "$ADMIN_TOKEN"
else
    print_warning "Token atau username tidak tersedia, skip test"
fi

print_section "6. DELETE /users/:username/follow - Not Following (Error Case)"
if [ -n "$ADMIN_TOKEN" ] && [ -n "$TEST_USERNAME" ] && [ "$TEST_USERNAME" != "null" ]; then
    make_request "DELETE" "/users/$TEST_USERNAME/follow" "Unfollow user yang belum di-follow" "" "$ADMIN_TOKEN"
else
    print_warning "Token atau username tidak tersedia, skip test"
fi

print_section "7. POST /portfolios/:id/like - Like Portfolio"
if [ -n "$ADMIN_TOKEN" ] && [ -n "$SAMPLE_PORTFOLIO_ID" ] && [ "$SAMPLE_PORTFOLIO_ID" != "null" ]; then
    make_request "POST" "/portfolios/$SAMPLE_PORTFOLIO_ID/like" "Like portfolio" "" "$ADMIN_TOKEN"
else
    print_warning "Token atau portfolio ID tidak tersedia, skip test"
fi

print_section "8. POST /portfolios/:id/like - Already Liked (Error Case)"
if [ -n "$ADMIN_TOKEN" ] && [ -n "$SAMPLE_PORTFOLIO_ID" ] && [ "$SAMPLE_PORTFOLIO_ID" != "null" ]; then
    make_request "POST" "/portfolios/$SAMPLE_PORTFOLIO_ID/like" "Like portfolio yang sudah di-like" "" "$ADMIN_TOKEN"
else
    print_warning "Token atau portfolio ID tidak tersedia, skip test"
fi

print_section "9. POST /portfolios/:id/like - Portfolio Not Found (Error Case)"
if [ -n "$ADMIN_TOKEN" ]; then
    make_request "POST" "/portfolios/00000000-0000-0000-0000-000000000000/like" "Like portfolio yang tidak ada" "" "$ADMIN_TOKEN"
else
    print_warning "Token tidak tersedia, skip test"
fi

print_section "10. DELETE /portfolios/:id/like - Unlike Portfolio"
if [ -n "$ADMIN_TOKEN" ] && [ -n "$SAMPLE_PORTFOLIO_ID" ] && [ "$SAMPLE_PORTFOLIO_ID" != "null" ]; then
    make_request "DELETE" "/portfolios/$SAMPLE_PORTFOLIO_ID/like" "Unlike portfolio" "" "$ADMIN_TOKEN"
else
    print_warning "Token atau portfolio ID tidak tersedia, skip test"
fi

print_section "11. DELETE /portfolios/:id/like - Not Liked (Error Case)"
if [ -n "$ADMIN_TOKEN" ] && [ -n "$SAMPLE_PORTFOLIO_ID" ] && [ "$SAMPLE_PORTFOLIO_ID" != "null" ]; then
    make_request "DELETE" "/portfolios/$SAMPLE_PORTFOLIO_ID/like" "Unlike portfolio yang belum di-like" "" "$ADMIN_TOKEN"
else
    print_warning "Token atau portfolio ID tidak tersedia, skip test"
fi

print_summary
