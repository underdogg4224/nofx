package api

import (
	"fmt"
	"net"
	"net/url"
	"nofx/config"
	"os"
	"strings"
)

// MaskedSecretSentinel is returned to API clients instead of raw secrets so UI
// truthiness checks remain intact without exposing credentials to the browser.
const MaskedSecretSentinel = "********"

// MaskSecret returns MaskedSecretSentinel when a secret is set, or "" when empty.
func MaskSecret(secret string) string {
	if strings.TrimSpace(secret) == "" {
		return ""
	}
	return MaskedSecretSentinel
}

// IsMaskedSecret reports whether value is the masked placeholder returned by GET endpoints.
func IsMaskedSecret(value string) bool {
	return strings.TrimSpace(value) == MaskedSecretSentinel
}

// resolveSecretUpdate determines the secret value to persist.
// Masked sentinels always preserve the existing secret; blank values preserve the
// existing secret while enabled=true so partial edits never wipe stored credentials.
func resolveSecretUpdate(incoming, existing string, enabled bool) string {
	if IsMaskedSecret(incoming) {
		return existing
	}
	if enabled && strings.TrimSpace(incoming) == "" && existing != "" {
		return existing
	}
	return incoming
}

func sanitizeAIModels(models []*config.AIModelConfig) []*config.AIModelConfig {
	sanitized := make([]*config.AIModelConfig, len(models))
	for i, m := range models {
		copyModel := *m
		copyModel.APIKey = MaskSecret(m.APIKey)
		sanitized[i] = &copyModel
	}
	return sanitized
}

func sanitizeExchanges(exchanges []*config.ExchangeConfig) []*config.ExchangeConfig {
	sanitized := make([]*config.ExchangeConfig, len(exchanges))
	for i, e := range exchanges {
		copyExchange := *e
		copyExchange.APIKey = MaskSecret(e.APIKey)
		copyExchange.SecretKey = MaskSecret(e.SecretKey)
		copyExchange.AsterPrivateKey = MaskSecret(e.AsterPrivateKey)
		sanitized[i] = &copyExchange
	}
	return sanitized
}

// ValidateExternalURL validates user-provided external URLs against SSRF targets.
// Set ALLOW_PRIVATE_URLS=true to permit loopback/private hosts (e.g. local Ollama).
func ValidateExternalURL(rawURL string) error {
	trimmed := strings.TrimSpace(rawURL)
	if trimmed == "" {
		return nil
	}

	// Custom AI URLs may use a trailing '#' to indicate full-URL mode.
	trimmed = strings.TrimSuffix(trimmed, "#")

	parsed, err := url.Parse(trimmed)
	if err != nil {
		return fmt.Errorf("无效的URL格式: %w", err)
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("仅支持 http 或 https 协议")
	}

	host := strings.ToLower(parsed.Hostname())
	if host == "" {
		return fmt.Errorf("URL缺少主机名")
	}

	if os.Getenv("ALLOW_PRIVATE_URLS") == "true" {
		return nil
	}

	if isBlockedHostname(host) {
		return fmt.Errorf("不允许访问内网或本地地址: %s", host)
	}

	if ip := net.ParseIP(host); ip != nil && isBlockedIP(ip) {
		return fmt.Errorf("不允许访问内网或保留IP地址: %s", host)
	}

	return nil
}

func isBlockedHostname(host string) bool {
	if host == "localhost" || host == "metadata.google.internal" || host == "metadata" {
		return true
	}
	return strings.HasSuffix(host, ".localhost") ||
		strings.HasSuffix(host, ".local") ||
		strings.HasSuffix(host, ".internal")
}

func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast() ||
		ip.IsUnspecified()
}
