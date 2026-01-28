#!/bin/bash

# Color output
# カラー出力
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
GRAY='\033[0;90m'
NC='\033[0m'

# Container detection
# コンテナ検出
is_running_in_container() {
    if [ -f /.dockerenv ]; then
        return 0
    fi

    if [ -f /proc/1/cgroup ]; then
        if grep -qE 'docker|lxc|containerd' /proc/1/cgroup 2>/dev/null; then
            return 0
        fi
    fi

    if echo "$HOSTNAME" | grep -qE '^[0-9a-f]{12,}$'; then
        return 0
    fi

    return 1
}

# Container check
# コンテナチェック
if ! is_running_in_container; then
    echo -e "${RED}╔════════════════════════════════════════╗${NC}"
    echo -e "${RED}║     ERROR: Not in Container           ║${NC}"
    echo -e "${RED}╚════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${YELLOW}This security test must be run inside a Docker container.${NC}"
    echo ""
    echo "To run this test:"
    echo -e "  ${BLUE}1.${NC} ./cli_sandbox/ai_sandbox.sh bash"
    echo -e "  ${BLUE}2.${NC} cd ./cli_sandbox"
    echo -e "  ${BLUE}3.${NC} ./test-sudo-security.sh"
    echo ""
    exit 1
fi

# Test counter
# テストカウンター
PASS=0
FAIL=0
TOTAL=0

echo "╔════════════════════════════════════════╗"
echo "║    AI Sandbox - Security Test         ║"
echo "╚════════════════════════════════════════╝"
echo ""

test_should_succeed() {
    local cmd="$1"
    local desc="$2"
    ((TOTAL++))

    echo -e "${BLUE}[$TOTAL]${NC} $desc"
    echo -e "${GRAY}    $ $cmd${NC}"

    if eval "$cmd" >/dev/null 2>&1; then
        echo -e "${GREEN}    ✓ PASS (no password required)${NC}"
        ((PASS++))
    else
        echo -e "${RED}    ✗ FAIL (should work without password)${NC}"
        ((FAIL++))
    fi
    echo ""
}

test_should_fail() {
    local cmd="$1"
    local desc="$2"
    ((TOTAL++))

    echo -e "${BLUE}[$TOTAL]${NC} $desc"
    echo -e "${GRAY}    $ sudo -n $cmd${NC}"

    local output=$(sudo -n $cmd 2>&1)
    local exit_code=$?

    if echo "$output" | grep -q "password is required\|not allowed"; then
        echo -e "${GREEN}    ✓ PASS (blocked)${NC}"
        ((PASS++))
    elif [ $exit_code -ne 0 ]; then
        echo -e "${GREEN}    ✓ PASS (blocked)${NC}"
        ((PASS++))
    else
        echo -e "${RED}    ✗ FAIL (should be blocked)${NC}"
        ((FAIL++))
    fi
    echo ""
}

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "1. CONFIGURATION CHECK"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

echo -e "${GREEN}✓ Running in container${NC}"
echo -e "${BLUE}Current user:${NC} $(whoami)"
echo -e "${BLUE}User ID:${NC} $(id)"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "2. ALLOWED COMMANDS (no password required)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_should_succeed "sudo -n apt-get --version" "apt-get"
test_should_succeed "sudo -n apt --version" "apt"
test_should_succeed "sudo -n dpkg --version" "dpkg"
test_should_succeed "sudo -n pip3 --version" "pip3"
test_should_succeed "sudo -n npm --version" "npm"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "3. DENIED COMMANDS (should be blocked)"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

test_should_fail "rm --version" "rm"
test_should_fail "chmod --version" "chmod"
test_should_fail "chown --version" "chown"
test_should_fail "su --version" "su"
test_should_fail "bash --version" "bash"
test_should_fail "cat --version" "cat"
test_should_fail "mv --version" "mv"
test_should_fail "cp --version" "cp"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "TEST SUMMARY"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo -e "Total:  ${BLUE}$TOTAL${NC}"
echo -e "Passed: ${GREEN}$PASS${NC}"
echo -e "Failed: ${RED}$FAIL${NC}"
echo ""

if [ $FAIL -eq 0 ]; then
    echo -e "${GREEN}╔═══════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║     ALL TESTS PASSED! ✓               ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════╝${NC}"
    echo ""
    echo "Security Status:"
    echo "  ✓ Package managers: apt-get, pip3, npm (allowed)"
    echo "  ✓ System commands: rm, chmod, su, bash (blocked)"
    echo ""
    echo "Your AI sandbox is secure!"
    exit 0
else
    echo -e "${RED}╔════════════════════════╗${NC}"
    echo -e "${RED}║  SOME TESTS FAILED! ✗  ║${NC}"
    echo -e "${RED}╚════════════════════════╝${NC}"
    exit 1
fi
