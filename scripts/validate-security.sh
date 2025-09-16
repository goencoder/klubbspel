#!/bin/bash

# Security validation script for Klubbspel backend
# Validates that major gosec security issues have been addressed

echo "🔒 Security Validation Script"
echo "============================="

BACKEND_DIR="backend"
PASSED=0
FAILED=0

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

check_pass() {
    echo -e "${GREEN}✅ PASS${NC}: $1"
    ((PASSED++))
}

check_fail() {
    echo -e "${RED}❌ FAIL${NC}: $1"
    ((FAILED++))
}

check_warn() {
    echo -e "${YELLOW}⚠️  WARN${NC}: $1"
}

echo ""
echo "Checking for gosec security issue fixes..."
echo ""

# Check 1: G101 - No hardcoded credentials
echo "1. Checking for hardcoded credentials (G101)..."
if grep -r "default-key-should-be-changed" "$BACKEND_DIR" >/dev/null 2>&1; then
    check_fail "Found hardcoded default encryption key"
else
    check_pass "No hardcoded encryption keys found"
fi

# Check 2: G501 - Conditional insecure credentials
echo ""
echo "2. Checking for insecure gRPC credentials (G501)..."
if grep -r "insecure.NewCredentials()" "$BACKEND_DIR" >/dev/null 2>&1; then
    # Check if it's conditional on environment
    if grep -A5 -B5 "insecure.NewCredentials()" "$BACKEND_DIR/internal/server/gateway.go" | grep -q "development\|environment"; then
        check_pass "Insecure credentials are environment-conditional"
    else
        check_fail "Insecure credentials used unconditionally"
    fi
else
    check_pass "No insecure credentials found"
fi

# Check 3: G104 - Error handling in main function
echo ""
echo "3. Checking for unhandled errors in main function (G104)..."
if grep -r "_ = .*Shutdown" "$BACKEND_DIR/cmd/api/main.go" >/dev/null 2>&1; then
    check_fail "Found unhandled shutdown errors in main function"
else
    check_pass "Proper error handling in main function"
fi

# Check 4: Environment variable requirement for encryption key
echo ""
echo "4. Checking for GDPR encryption key validation..."
if grep -r "GDPR_ENCRYPTION_KEY" "$BACKEND_DIR/internal/config/config.go" >/dev/null 2>&1; then
    check_pass "GDPR encryption key loaded from environment"
else
    check_fail "GDPR encryption key not configured from environment"
fi

# Check 5: Key validation in encryption service
echo ""
echo "5. Checking for encryption key validation..."
if grep -r "encryption key cannot be empty" "$BACKEND_DIR/internal/gdpr/gdpr_manager.go" >/dev/null 2>&1; then
    check_pass "Encryption key validation implemented"
else
    check_fail "Missing encryption key validation"
fi

# Check 6: TLS configuration for production
echo ""
echo "6. Checking for TLS configuration in production..."
if grep -r "credentials.NewTLS" "$BACKEND_DIR/internal/server/gateway.go" >/dev/null 2>&1; then
    check_pass "TLS credentials configured for production"
else
    check_warn "TLS credentials may not be configured"
fi

# Summary
echo ""
echo "============================="
echo "Security Validation Summary"
echo "============================="
echo -e "Checks passed: ${GREEN}${PASSED}${NC}"
echo -e "Checks failed: ${RED}${FAILED}${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}🎉 All security checks passed!${NC}"
    echo ""
    echo "Major gosec issues addressed:"
    echo "  • G101: Hardcoded credentials removed"
    echo "  • G104: Error handling improved"  
    echo "  • G501: Insecure credentials made conditional"
    echo ""
    echo "Next steps:"
    echo "  • Set GDPR_ENCRYPTION_KEY environment variable"
    echo "  • Run full gosec scan: make security"
    echo "  • Test in development environment"
    exit 0
else
    echo -e "${RED}🚨 ${FAILED} security issue(s) need attention${NC}"
    exit 1
fi