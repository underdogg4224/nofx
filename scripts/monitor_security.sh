#!/bin/bash

# Security Monitoring Script for NOFX
# Monitors security logs for suspicious activity and sends alerts

# Configuration
SECURITY_LOG="${SECURITY_LOG:-logs/security.log}"
ALERT_THRESHOLD_BRUTE_FORCE=5
ALERT_THRESHOLD_UNAUTHORIZED=10
ALERT_THRESHOLD_RATE_LIMIT=20

# Colors for terminal output
RED='\033[0;31m'
YELLOW='\033[1;33m'
GREEN='\033[0;32m'
NC='\033[0m' # No Color

# Check if security log exists
if [ ! -f "$SECURITY_LOG" ]; then
    echo -e "${YELLOW}⚠️  Security log not found: $SECURITY_LOG${NC}"
    echo "   Security logging may be disabled or no events have been logged yet."
    exit 0
fi

echo "🔍 Analyzing security logs..."
echo ""

# Function to count events in the last hour
count_recent_events() {
    local event_type="$1"
    local time_window="${2:-1 hour ago}"

    # Get timestamp from 1 hour ago
    HOUR_AGO=$(date -u -d "$time_window" '+%Y-%m-%dT%H:%M:%S' 2>/dev/null || date -u -v-1H '+%Y-%m-%dT%H:%M:%S')

    # Count events after that timestamp
    grep "\"event_type\":\"$event_type\"" "$SECURITY_LOG" 2>/dev/null | \
        awk -v cutoff="$HOUR_AGO" -F'"timestamp":"' '{print $2}' | \
        awk -F'"' '{print $1}' | \
        awk -v cutoff="$cutoff" '$1 >= cutoff' | wc -l
}

# Function to get top IPs for a specific event
get_top_ips() {
    local event_type="$1"
    local limit="${2:-5}"

    grep "\"event_type\":\"$event_type\"" "$SECURITY_LOG" 2>/dev/null | \
        grep -o '"ip_address":"[^"]*"' | \
        cut -d'"' -f4 | \
        sort | uniq -c | sort -rn | head -n "$limit"
}

# 1. Check for brute force attacks
echo "🔨 Brute Force Attack Detection"
echo "───────────────────────────────────────────"

LOGIN_FAILURES=$(count_recent_events "LOGIN_FAILURE")
OTP_FAILURES=$(count_recent_events "OTP_VERIFY_FAILURE")

if [ "$LOGIN_FAILURES" -gt "$ALERT_THRESHOLD_BRUTE_FORCE" ]; then
    echo -e "${RED}🚨 ALERT: $LOGIN_FAILURES failed login attempts in the last hour!${NC}"
    echo "   Top IPs:"
    get_top_ips "LOGIN_FAILURE" 3 | while read count ip; do
        echo "   - $ip: $count attempts"
    done
else
    echo -e "${GREEN}✅ No brute force attacks detected${NC} ($LOGIN_FAILURES failed logins)"
fi

if [ "$OTP_FAILURES" -gt "$ALERT_THRESHOLD_BRUTE_FORCE" ]; then
    echo -e "${RED}🚨 ALERT: $OTP_FAILURES failed OTP attempts in the last hour!${NC}"
    echo "   Top IPs:"
    get_top_ips "OTP_VERIFY_FAILURE" 3 | while read count ip; do
        echo "   - $ip: $count attempts"
    done
else
    echo -e "${GREEN}✅ No OTP brute force detected${NC} ($OTP_FAILURES failed OTP verifications)"
fi

echo ""

# 2. Check for unauthorized access
echo "🚫 Unauthorized Access Attempts"
echo "───────────────────────────────────────────"

UNAUTHORIZED=$(count_recent_events "UNAUTHORIZED_ACCESS")

if [ "$UNAUTHORIZED" -gt "$ALERT_THRESHOLD_UNAUTHORIZED" ]; then
    echo -e "${RED}🚨 ALERT: $UNAUTHORIZED unauthorized access attempts in the last hour!${NC}"
    echo "   Top IPs:"
    get_top_ips "UNAUTHORIZED_ACCESS" 3 | while read count ip; do
        echo "   - $ip: $count attempts"
    done
else
    echo -e "${GREEN}✅ No unusual unauthorized access${NC} ($UNAUTHORIZED attempts)"
fi

echo ""

# 3. Check for rate limit violations
echo "⏱️  Rate Limit Violations"
echo "───────────────────────────────────────────"

RATE_LIMIT=$(count_recent_events "RATE_LIMIT_EXCEEDED")

if [ "$RATE_LIMIT" -gt "$ALERT_THRESHOLD_RATE_LIMIT" ]; then
    echo -e "${YELLOW}⚠️  WARNING: $RATE_LIMIT rate limit violations in the last hour${NC}"
    echo "   Top IPs:"
    get_top_ips "RATE_LIMIT_EXCEEDED" 3 | while read count ip; do
        echo "   - $ip: $count violations"
    done
else
    echo -e "${GREEN}✅ Normal rate limit activity${NC} ($RATE_LIMIT violations)"
fi

echo ""

# 4. Successful logins summary
echo "✅ Successful Logins (Last Hour)"
echo "───────────────────────────────────────────"

LOGIN_SUCCESS=$(count_recent_events "LOGIN_SUCCESS")
OTP_SUCCESS=$(count_recent_events "OTP_VERIFY_SUCCESS")

echo "   Password verified: $LOGIN_SUCCESS"
echo "   OTP verified: $OTP_SUCCESS"
echo ""

# 5. Recent registrations
echo "👤 Recent Registrations (Last 24 Hours)"
echo "───────────────────────────────────────────"

REGISTRATIONS=$(count_recent_events "REGISTRATION" "24 hours ago")
echo "   New users: $REGISTRATIONS"
echo ""

# 6. Summary statistics
echo "📊 Overall Statistics (Last 24 Hours)"
echo "───────────────────────────────────────────"

TOTAL_EVENTS=$(tail -n 10000 "$SECURITY_LOG" 2>/dev/null | wc -l)
FAILED_EVENTS=$(grep '"success":false' "$SECURITY_LOG" 2>/dev/null | tail -n 1000 | wc -l)

echo "   Total security events: $TOTAL_EVENTS"
echo "   Failed events (recent): $FAILED_EVENTS"
echo ""

# 7. Recommendations
echo "💡 Recommendations"
echo "───────────────────────────────────────────"

CRITICAL_ISSUES=0

if [ "$LOGIN_FAILURES" -gt "$ALERT_THRESHOLD_BRUTE_FORCE" ]; then
    echo -e "${RED}❗ Block IPs with excessive login failures${NC}"
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + 1))
fi

if [ "$OTP_FAILURES" -gt "$ALERT_THRESHOLD_BRUTE_FORCE" ]; then
    echo -e "${RED}❗ Investigate OTP brute force attempts${NC}"
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + 1))
fi

if [ "$UNAUTHORIZED" -gt "$ALERT_THRESHOLD_UNAUTHORIZED" ]; then
    echo -e "${RED}❗ Review unauthorized access patterns${NC}"
    CRITICAL_ISSUES=$((CRITICAL_ISSUES + 1))
fi

if [ "$RATE_LIMIT" -gt "$ALERT_THRESHOLD_RATE_LIMIT" ]; then
    echo -e "${YELLOW}⚠️  Consider reducing rate limits or implementing IP bans${NC}"
fi

if [ "$CRITICAL_ISSUES" -eq 0 ]; then
    echo -e "${GREEN}✅ No critical security issues detected${NC}"
fi

echo ""
echo "───────────────────────────────────────────"
echo "🔍 Analysis complete. Review full logs at: $SECURITY_LOG"
echo ""

# Exit with error code if critical issues found (for automated monitoring)
if [ "$CRITICAL_ISSUES" -gt 0 ]; then
    exit 1
fi

exit 0
