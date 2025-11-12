# 🔐 Security Enhancements Guide

This document describes the comprehensive security improvements implemented in NOFX to protect existing users and their trading funds.

## 📋 Table of Contents

- [Overview](#overview)
- [Implemented Security Features](#implemented-security-features)
- [Configuration](#configuration)
- [Security Best Practices](#security-best-practices)
- [Monitoring & Alerts](#monitoring--alerts)
- [Incident Response](#incident-response)

---

## Overview

NOFX now includes enterprise-grade security features to protect user accounts, API keys, and trading funds:

### ✅ Critical Security Improvements

1. **AES-256-GCM Encryption** for API keys and secrets
2. **Rate Limiting** to prevent brute force attacks
3. **Improved CORS** with origin whitelisting
4. **Security Headers** (CSP, HSTS, X-Frame-Options, etc.)
5. **Security Event Logging** for audit and monitoring
6. **Input Sanitization** to prevent XSS/injection attacks
7. **Enhanced Password Requirements**

---

## Implemented Security Features

### 1. 🔐 API Key Encryption (AES-256-GCM)

All sensitive credentials are now encrypted at rest:

**Encrypted Data:**
- Exchange API keys (Binance, Hyperliquid, Aster)
- Exchange secret keys
- Blockchain private keys (DEX wallets)
- AI model API keys (DeepSeek, Qwen, custom)
- OTP secrets (2FA)

**Encryption Details:**
- **Algorithm:** AES-256-GCM (authenticated encryption)
- **Key Derivation:** PBKDF2 with SHA-256 (100,000 iterations)
- **Unique Nonces:** Each encryption uses a unique nonce
- **Automatic Migration:** Existing plaintext keys are encrypted on first access

**Configuration:**

```bash
# Set encryption master secret (REQUIRED for production)
export NOFX_ENCRYPTION_SECRET="your-strong-random-secret-min-32-chars"

# Or use a file-based secret
export NOFX_ENCRYPTION_SECRET_FILE="/path/to/secret/file"
```

**Auto-generation:**
If no secret is provided, the system generates one automatically and stores it in `.nofx_encryption_secret` (file permissions: 0600).

⚠️ **Important:**
- Keep your encryption secret safe - losing it means losing access to all encrypted credentials
- Back up your encryption secret securely
- Never commit the secret to version control

---

### 2. 🛡️ Rate Limiting

Protection against brute force attacks on authentication endpoints:

**Rate Limits:**

| Endpoint | Limit | Window |
|----------|-------|--------|
| Login | 5 attempts | 1 minute |
| Registration | 3 attempts | 1 hour |
| OTP Verification | 5 attempts | 1 minute |
| General API | 100 requests | 1 minute |

**Features:**
- IP-based rate limiting
- Automatic violation tracking
- Security alerts after 10 consecutive violations
- Configurable limits per endpoint

**Responses:**
```json
{
  "error": "登录请求过于频繁，请稍后再试",
  "code": "RATE_LIMIT_EXCEEDED"
}
```

---

### 3. 🌐 Improved CORS Configuration

Secure cross-origin resource sharing with origin whitelisting:

**Default Allowed Origins:**
- `http://localhost:3000`
- `http://localhost:5173`
- `http://127.0.0.1:3000`
- `http://127.0.0.1:5173`

**Configuration:**

```bash
# Whitelist specific origins (recommended for production)
export NOFX_ALLOWED_ORIGINS="https://yourdomain.com,https://app.yourdomain.com"

# For development only (allows all origins)
export NOFX_ALLOWED_ORIGINS="*"
```

**Security Benefits:**
- Prevents unauthorized websites from accessing your API
- Mitigates CSRF attacks
- Supports credentials with `Access-Control-Allow-Credentials`

---

### 4. 🔒 Security Headers

Comprehensive HTTP security headers protect against common web attacks:

**Implemented Headers:**

```http
X-Frame-Options: DENY
X-Content-Type-Options: nosniff
X-XSS-Protection: 1; mode=block
Referrer-Policy: strict-origin-when-cross-origin
Content-Security-Policy: default-src 'self'; ...
Permissions-Policy: geolocation=(), microphone=(), camera=()
Strict-Transport-Security: max-age=31536000; includeSubDomains (HTTPS only)
```

**Protection Against:**
- ✅ Clickjacking attacks
- ✅ MIME type confusion
- ✅ Cross-site scripting (XSS)
- ✅ Information leakage via referrer
- ✅ Unauthorized feature access (camera, microphone, etc.)

---

### 5. 📊 Security Event Logging

Comprehensive logging of all security-related events:

**Logged Events:**
- Login attempts (success/failure)
- OTP verifications (success/failure)
- Password changes
- Rate limit violations
- Unauthorized access attempts
- Token expiration/invalidation
- Suspicious activity

**Log Format:**
```json
{
  "timestamp": "2025-01-15T10:30:00Z",
  "event_type": "LOGIN_FAILURE",
  "user_id": "uuid",
  "email": "user@example.com",
  "ip_address": "192.168.1.100",
  "user_agent": "Mozilla/5.0...",
  "success": false,
  "message": "Invalid password"
}
```

**Log Location:** `logs/security.log`

**Configuration:**
```bash
# Enable security logging (enabled by default)
export NOFX_SECURITY_LOGGING=true

# Disable security logging (not recommended)
export NOFX_SECURITY_LOGGING=false
```

**Critical Alerts:**
Certain events trigger immediate console alerts:
- Brute force attacks (10+ consecutive violations)
- Unauthorized access attempts
- Account lockouts
- Suspicious activity

---

### 6. 🧹 Input Sanitization

All user inputs are validated and sanitized to prevent attacks:

**Email Validation:**
- Format validation (RFC-compliant regex)
- Case normalization (lowercase)
- HTML escaping
- Null byte removal

**Password Requirements:**
- Minimum 8 characters
- Maximum 128 characters
- Must contain at least 3 of:
  - Uppercase letters (A-Z)
  - Lowercase letters (a-z)
  - Numbers (0-9)
  - Special characters (!@#$%^&*...)

**Trader Names:**
- Alphanumeric + spaces, hyphens, underscores only
- Maximum 100 characters
- Automatic sanitization

**Trading Symbols:**
- Uppercase letters and numbers only
- 4-20 characters
- Format validation (e.g., BTCUSDT)

**API Keys & Private Keys:**
- Length validation (16-256 characters)
- Format validation (alphanumeric + hyphens/underscores)
- Ethereum address validation (0x + 40 hex chars)

---

## Configuration

### Environment Variables

All security settings can be configured via environment variables:

```bash
# Encryption
export NOFX_ENCRYPTION_SECRET="your-secret-min-32-chars"

# CORS
export NOFX_ALLOWED_ORIGINS="https://yourdomain.com,https://app.yourdomain.com"

# Security Logging
export NOFX_SECURITY_LOGGING=true

# JWT Secret (recommended to set in production)
export JWT_SECRET="your-jwt-secret-min-64-chars-recommended"
```

### Docker Deployment

Add to `.env` file:

```env
NOFX_ENCRYPTION_SECRET=your-secret-min-32-chars
NOFX_ALLOWED_ORIGINS=https://yourdomain.com
NOFX_SECURITY_LOGGING=true
JWT_SECRET=your-jwt-secret-min-64-chars
```

Update `docker-compose.yml`:

```yaml
services:
  backend:
    environment:
      - NOFX_ENCRYPTION_SECRET=${NOFX_ENCRYPTION_SECRET}
      - NOFX_ALLOWED_ORIGINS=${NOFX_ALLOWED_ORIGINS}
      - NOFX_SECURITY_LOGGING=${NOFX_SECURITY_LOGGING}
      - JWT_SECRET=${JWT_SECRET}
```

---

## Security Best Practices

### For Users

1. **Strong Encryption Secret**
   ```bash
   # Generate a strong random secret (Linux/Mac)
   openssl rand -base64 48

   # Use this as your NOFX_ENCRYPTION_SECRET
   export NOFX_ENCRYPTION_SECRET="generated-secret-here"
   ```

2. **Unique JWT Secret**
   ```bash
   # Generate a strong JWT secret
   openssl rand -hex 64

   # Set in config.json or environment
   export JWT_SECRET="generated-jwt-secret-here"
   ```

3. **Restrict CORS Origins**
   ```bash
   # Production: Only allow your domain
   export NOFX_ALLOWED_ORIGINS="https://trade.yourdomain.com"

   # NEVER use "*" in production
   ```

4. **Secure Database Files**
   ```bash
   # Set proper file permissions
   chmod 600 config.db
   chmod 600 .nofx_encryption_secret

   # Verify permissions
   ls -la config.db .nofx_encryption_secret
   ```

5. **Regular Backups**
   ```bash
   # Backup database and encryption secret
   tar -czf nofx-backup-$(date +%Y%m%d).tar.gz \
     config.db .nofx_encryption_secret

   # Store backup securely off-site
   ```

6. **Monitor Security Logs**
   ```bash
   # Watch for suspicious activity
   tail -f logs/security.log | grep -E "(FAILURE|EXCEEDED|UNAUTHORIZED)"

   # Check for brute force attempts
   grep "BRUTE_FORCE" logs/security.log
   ```

7. **Use HTTPS in Production**
   - Never expose NOFX API over HTTP in production
   - Use a reverse proxy (nginx, Caddy) with TLS certificates
   - Consider using Let's Encrypt for free SSL certificates

8. **Network Security**
   ```bash
   # Firewall: Only allow localhost access to API
   iptables -A INPUT -p tcp --dport 8080 -s 127.0.0.1 -j ACCEPT
   iptables -A INPUT -p tcp --dport 8080 -j DROP
   ```

9. **Use Binance Subaccounts**
   - Create dedicated subaccounts for NOFX trading
   - Limit maximum balance
   - Restrict withdrawal permissions
   - Use IP whitelist on Binance API settings

10. **Enable 2FA Everywhere**
    - NOFX enforces mandatory 2FA (Google Authenticator)
    - Also enable 2FA on your exchange accounts
    - Keep backup codes secure

---

## Monitoring & Alerts

### Real-time Security Monitoring

**Critical Events Logged to Console:**
```
🚨 [SECURITY] 检测到可能的暴力攻击: IP=192.168.1.100, 端点=/api/login, 连续违规=12次
```

**Security Log Analysis:**

```bash
# Count failed login attempts by IP
cat logs/security.log | grep "LOGIN_FAILURE" | \
  jq -r '.ip_address' | sort | uniq -c | sort -rn

# Find accounts with multiple failed OTP attempts
cat logs/security.log | grep "OTP_VERIFY_FAILURE" | \
  jq -r '.email' | sort | uniq -c | sort -rn

# List rate limit violations
cat logs/security.log | grep "RATE_LIMIT_EXCEEDED" | \
  jq -r '{time: .timestamp, ip: .ip_address, endpoint: .message}'
```

### Automated Monitoring Script

Create `scripts/monitor_security.sh`:

```bash
#!/bin/bash
SECURITY_LOG="logs/security.log"
ALERT_EMAIL="admin@yourdomain.com"

# Monitor for brute force attacks
BRUTE_FORCE=$(grep -c "BRUTE_FORCE" "$SECURITY_LOG")
if [ "$BRUTE_FORCE" -gt 0 ]; then
    echo "⚠️  Detected $BRUTE_FORCE brute force attacks!" | mail -s "NOFX Security Alert" "$ALERT_EMAIL"
fi

# Monitor for unauthorized access
UNAUTHORIZED=$(grep -c "UNAUTHORIZED_ACCESS" "$SECURITY_LOG")
if [ "$UNAUTHORIZED" -gt 10 ]; then
    echo "⚠️  Detected $UNAUTHORIZED unauthorized access attempts!" | mail -s "NOFX Security Alert" "$ALERT_EMAIL"
fi
```

Run periodically with cron:
```cron
*/15 * * * * /path/to/scripts/monitor_security.sh
```

---

## Incident Response

### If You Suspect a Breach

1. **Immediate Actions**
   - Stop all traders: `docker-compose down` or kill the process
   - Change all passwords immediately
   - Rotate all API keys on exchanges
   - Review security logs for unauthorized access

2. **Investigation**
   ```bash
   # Check recent security events
   tail -n 1000 logs/security.log | grep -E "(FAILURE|UNAUTHORIZED)"

   # Check database for suspicious traders
   sqlite3 config.db "SELECT * FROM traders WHERE created_at > datetime('now', '-1 day');"

   # Check for unauthorized user registrations
   sqlite3 config.db "SELECT * FROM users WHERE created_at > datetime('now', '-1 day');"
   ```

3. **Recovery Steps**
   - Generate new encryption secret
   - Re-encrypt all API keys with new secret
   - Update JWT secret
   - Force all users to re-login
   - Audit all active trading positions

4. **Prevention**
   - Enable stricter CORS (remove wildcard)
   - Reduce rate limits further
   - Add IP whitelisting
   - Enable additional logging
   - Consider adding webhook notifications for critical events

---

## Security Checklist

Before deploying to production, ensure:

- [ ] `NOFX_ENCRYPTION_SECRET` is set to a strong random value
- [ ] `JWT_SECRET` is set to a strong random value
- [ ] `NOFX_ALLOWED_ORIGINS` does NOT include "*"
- [ ] HTTPS/TLS is configured (reverse proxy with SSL certificate)
- [ ] Database files have restrictive permissions (600)
- [ ] `.nofx_encryption_secret` file has permissions 600
- [ ] Security logging is enabled
- [ ] Firewall rules restrict API access
- [ ] Regular backups are configured
- [ ] Security monitoring is active
- [ ] Binance API keys use IP whitelist
- [ ] Binance subaccounts are used (not main account)
- [ ] All exchange API keys have withdrawal disabled

---

## Additional Resources

**Security Documentation:**
- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [Binance API Security](https://www.binance.com/en/support/faq/360002502072)
- [NIST Password Guidelines](https://pages.nist.gov/800-63-3/sp800-63b.html)

**Related Files:**
- `security/encryption.go` - Encryption implementation
- `security/ratelimit.go` - Rate limiting
- `security/sanitize.go` - Input validation
- `security/logger.go` - Security event logging
- `SECURITY.md` - Vulnerability reporting

---

## Support

**For Security Issues:**
- 🐦 Twitter DM: [@Web3Tinkle](https://x.com/Web3Tinkle)
- 📧 Email: security@nofx.ai (if available)
- See [SECURITY.md](../SECURITY.md) for responsible disclosure

**For General Questions:**
- 💬 [Telegram Community](https://t.me/nofx_dev_community)
- 📚 [Documentation](../README.md)

---

**Last Updated:** 2025-01-15
**Version:** 3.1.0 (Security Enhanced)

🔐 **Stay secure, trade safely!**
