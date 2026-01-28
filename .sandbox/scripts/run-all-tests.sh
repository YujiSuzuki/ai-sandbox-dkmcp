#!/bin/bash
# run-all-tests.sh
# Run all test scripts in .sandbox/scripts/
# .sandbox/scripts/ 内の全テストスクリプトを実行
#
# Usage: ./.sandbox/scripts/run-all-tests.sh
# 使用方法: ./.sandbox/scripts/run-all-tests.sh
#
# Environment: DevContainer (requires /workspace)
# 実行環境: DevContainer（/workspace が必要）

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Colors for output
# 出力用の色定義
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
NC='\033[0m' # No Color / 色なし

TOTAL=0
PASSED=0
FAILED=0
FAILED_SCRIPTS=()

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BOLD}  All Tests Runner${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

for test_script in "$SCRIPT_DIR"/test-*.sh; do
    [ -f "$test_script" ] || continue

    script_name="$(basename "$test_script")"
    TOTAL=$((TOTAL + 1))

    echo ""
    echo "══════════════════════════════════════════════════════════════"
    echo -e "${BOLD}  ▶ $script_name${NC}"
    echo "══════════════════════════════════════════════════════════════"

    # Special handling for test-advanced-features.sh
    # Run with --basic -y to execute DevContainer-compatible tests (sections 1-4)
    # test-advanced-features.sh は特別扱い
    # --basic -y で DevContainer 互換テスト（セクション1-4）を実行
    if [ "$script_name" = "test-advanced-features.sh" ]; then
        if bash "$test_script" --basic -y; then
            PASSED=$((PASSED + 1))
        else
            FAILED=$((FAILED + 1))
            FAILED_SCRIPTS+=("$script_name")
        fi
    else
        if bash "$test_script"; then
            PASSED=$((PASSED + 1))
        else
            FAILED=$((FAILED + 1))
            FAILED_SCRIPTS+=("$script_name")
        fi
    fi
done

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "${BOLD}  Overall Results: $PASSED/$TOTAL scripts passed${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

if [ "$FAILED" -gt 0 ]; then
    echo ""
    echo -e "${RED}  Failed scripts:${NC}"
    for s in "${FAILED_SCRIPTS[@]}"; do
        echo -e "${RED}    ✗ $s${NC}"
    done
    echo ""
    exit 1
else
    echo ""
    echo -e "${GREEN}  All test scripts passed!${NC}"
    echo ""
fi
