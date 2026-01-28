#!/bin/bash
# test-sync-ignore.sh
# Test sync-ignore pattern matching
# sync-ignore パターンマッチングのテスト

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
WORKSPACE="${WORKSPACE:-/workspace}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

# Counters
TESTS_PASSED=0
TESTS_FAILED=0

# Test helpers
pass() { echo -e "${GREEN}PASS${NC}: $1"; ((TESTS_PASSED++)) || true; }
fail() { echo -e "${RED}FAIL${NC}: $1"; ((TESTS_FAILED++)) || true; }
info() { echo -e "${YELLOW}INFO${NC}: $1"; }

# ============================================================
# Test: sync-ignore file exists
# ============================================================
test_sync_ignore_exists() {
    echo ""
    echo "=== Testing sync-ignore file ==="

    local ignore_file="$WORKSPACE/.sandbox/config/sync-ignore"

    if [ -f "$ignore_file" ]; then
        pass "sync-ignore file exists"

        # Check that it contains expected patterns
        if grep -q '^\*\*/\*\.example$' "$ignore_file"; then
            pass "sync-ignore contains **/*.example pattern"
        else
            fail "sync-ignore should contain **/*.example pattern"
        fi

        if grep -q '^\*\*/\*\.sample$' "$ignore_file"; then
            pass "sync-ignore contains **/*.sample pattern"
        else
            fail "sync-ignore should contain **/*.sample pattern"
        fi
    else
        fail "sync-ignore file does not exist"
    fi
}

# ============================================================
# Test: Pattern loading
# ============================================================
test_pattern_loading() {
    echo ""
    echo "=== Testing pattern loading ==="

    # Source the common functions
    # shellcheck source=/dev/null
    source "$WORKSPACE/.sandbox/scripts/_startup_common.sh"

    local patterns
    patterns=$(load_sync_ignore_patterns)

    if [ -n "$patterns" ]; then
        pass "load_sync_ignore_patterns returns patterns"

        # Check that comments are filtered out
        if echo "$patterns" | grep -q '^#'; then
            fail "load_sync_ignore_patterns should filter out comments"
        else
            pass "load_sync_ignore_patterns filters out comments"
        fi

        # Check that empty lines are filtered out
        if echo "$patterns" | grep -q '^$'; then
            fail "load_sync_ignore_patterns should filter out empty lines"
        else
            pass "load_sync_ignore_patterns filters out empty lines"
        fi
    else
        fail "load_sync_ignore_patterns returned empty"
    fi
}

# ============================================================
# Test: Pattern matching
# ============================================================
test_pattern_matching() {
    echo ""
    echo "=== Testing pattern matching ==="

    # Source the common functions
    # shellcheck source=/dev/null
    source "$WORKSPACE/.sandbox/scripts/_startup_common.sh"

    # Test 1: **/*.example pattern should match .example files
    if matches_sync_ignore "$WORKSPACE/demo-apps/securenote-api/.env.example"; then
        pass "**/*.example matches demo-apps/securenote-api/.env.example"
    else
        fail "**/*.example should match .env.example files"
    fi

    # Test 2: **/*.example should match nested .example files
    if matches_sync_ignore "$WORKSPACE/foo/bar/baz/test.example"; then
        pass "**/*.example matches deeply nested .example files"
    else
        fail "**/*.example should match deeply nested .example files"
    fi

    # Test 3: **/*.sample pattern should match .sample files
    if matches_sync_ignore "$WORKSPACE/config/settings.sample"; then
        pass "**/*.sample matches config/settings.sample"
    else
        fail "**/*.sample should match .sample files"
    fi

    # Test 4: Regular .env file should NOT match (not in ignore patterns)
    if matches_sync_ignore "$WORKSPACE/demo-apps/securenote-api/.env"; then
        fail ".env should NOT match ignore patterns"
    else
        pass ".env does not match ignore patterns (correct)"
    fi

    # Test 5: Regular secrets file should NOT match
    if matches_sync_ignore "$WORKSPACE/demo-apps/securenote-api/secrets/jwt-secret.key"; then
        fail "secrets/jwt-secret.key should NOT match ignore patterns"
    else
        pass "secrets/jwt-secret.key does not match ignore patterns (correct)"
    fi
}

# ============================================================
# Test: Fallback when file doesn't exist
# ============================================================
test_fallback() {
    echo ""
    echo "=== Testing fallback behavior ==="

    # Temporarily rename sync-ignore
    local ignore_file="$WORKSPACE/.sandbox/config/sync-ignore"
    local backup_file="$WORKSPACE/.sandbox/config/sync-ignore.backup.test"

    # Setup cleanup trap to restore file on any exit
    cleanup_fallback() {
        if [ -f "$backup_file" ]; then
            mv "$backup_file" "$ignore_file" 2>/dev/null || true
        fi
    }
    trap cleanup_fallback EXIT

    if [ -f "$ignore_file" ]; then
        mv "$ignore_file" "$backup_file"
    fi

    # Source the common functions (need to reload with file missing)
    # shellcheck source=/dev/null
    source "$WORKSPACE/.sandbox/scripts/_startup_common.sh"

    local patterns
    patterns=$(load_sync_ignore_patterns)

    if [ -z "$patterns" ]; then
        pass "load_sync_ignore_patterns returns empty when file doesn't exist"
    else
        fail "load_sync_ignore_patterns should return empty when file doesn't exist"
    fi

    # Files should NOT match when no ignore file exists
    if matches_sync_ignore "$WORKSPACE/test.example"; then
        fail "Should not match when sync-ignore doesn't exist"
    else
        pass "No matches when sync-ignore doesn't exist (correct)"
    fi

    # Restore sync-ignore
    cleanup_fallback
    trap - EXIT  # Remove trap
}

# ============================================================
# Main
# ============================================================
main() {
    echo "========================================"
    echo "Sync-Ignore Pattern Tests"
    echo "========================================"

    test_sync_ignore_exists
    test_pattern_loading
    test_pattern_matching
    test_fallback

    echo ""
    echo "========================================"
    echo "Test Results"
    echo "========================================"
    echo -e "Passed: ${GREEN}${TESTS_PASSED}${NC}"
    echo -e "Failed: ${RED}${TESTS_FAILED}${NC}"
    echo ""

    if [ $TESTS_FAILED -gt 0 ]; then
        exit 1
    fi
    exit 0
}

main "$@"
