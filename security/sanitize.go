package security

import (
	"html"
	"regexp"
	"strings"
)

// SanitizeInput sanitizes user input to prevent XSS and injection attacks
func SanitizeInput(input string) string {
	// Remove any null bytes
	input = strings.ReplaceAll(input, "\x00", "")

	// HTML escape
	input = html.EscapeString(input)

	return input
}

// SanitizeEmail validates and sanitizes email addresses
func SanitizeEmail(email string) string {
	email = strings.TrimSpace(strings.ToLower(email))
	return SanitizeInput(email)
}

// ValidateEmail checks if an email is valid
func ValidateEmail(email string) bool {
	// Basic email regex
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}$`)
	return emailRegex.MatchString(email)
}

// ValidatePassword checks if a password meets security requirements
func ValidatePassword(password string) (bool, string) {
	if len(password) < 8 {
		return false, "Password must be at least 8 characters long"
	}

	if len(password) > 128 {
		return false, "Password must be less than 128 characters"
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	hasNumber := regexp.MustCompile(`[0-9]`).MatchString(password)
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	strengthCount := 0
	if hasUpper {
		strengthCount++
	}
	if hasLower {
		strengthCount++
	}
	if hasNumber {
		strengthCount++
	}
	if hasSpecial {
		strengthCount++
	}

	if strengthCount < 3 {
		return false, "Password must contain at least 3 of: uppercase, lowercase, numbers, special characters"
	}

	return true, ""
}

// SanitizeTraderName sanitizes trader names
func SanitizeTraderName(name string) string {
	name = strings.TrimSpace(name)

	// Remove any characters that aren't alphanumeric, spaces, hyphens, or underscores
	reg := regexp.MustCompile(`[^a-zA-Z0-9 _\-]`)
	name = reg.ReplaceAllString(name, "")

	// Limit length
	if len(name) > 100 {
		name = name[:100]
	}

	return name
}

// SanitizeSymbol sanitizes trading symbol inputs
func SanitizeSymbol(symbol string) string {
	// Symbols should only contain uppercase letters and numbers
	symbol = strings.ToUpper(strings.TrimSpace(symbol))

	reg := regexp.MustCompile(`[^A-Z0-9]`)
	symbol = reg.ReplaceAllString(symbol, "")

	return symbol
}

// ValidateSymbol validates a trading symbol
func ValidateSymbol(symbol string) bool {
	// Most symbols should be 6-12 characters (e.g., BTCUSDT)
	if len(symbol) < 4 || len(symbol) > 20 {
		return false
	}

	// Should only contain uppercase letters and numbers
	matched, _ := regexp.MatchString(`^[A-Z0-9]+$`, symbol)
	return matched
}

// ValidateLeverage validates leverage values
func ValidateLeverage(leverage int, isBTCETH bool) bool {
	if leverage < 1 {
		return false
	}

	if isBTCETH {
		return leverage <= 50
	}

	return leverage <= 20
}

// SanitizeJSON removes potentially dangerous characters from JSON strings
func SanitizeJSON(input string) string {
	// Remove control characters except newlines and tabs
	reg := regexp.MustCompile(`[\x00-\x08\x0B\x0C\x0E-\x1F]`)
	return reg.ReplaceAllString(input, "")
}

// IsValidAPIKey checks if an API key format is valid
func IsValidAPIKey(apiKey string) bool {
	// API keys should be reasonable length and alphanumeric
	if len(apiKey) < 16 || len(apiKey) > 256 {
		return false
	}

	// Should only contain alphanumeric and common special chars
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9\-_]+$`, apiKey)
	return matched
}

// IsValidPrivateKey checks if a private key format is valid (for blockchain)
func IsValidPrivateKey(privateKey string) bool {
	// Remove 0x prefix if present
	privateKey = strings.TrimPrefix(privateKey, "0x")

	// Should be 64 hex characters
	if len(privateKey) != 64 {
		return false
	}

	matched, _ := regexp.MatchString(`^[a-fA-F0-9]+$`, privateKey)
	return matched
}

// IsValidWalletAddress checks if an Ethereum wallet address is valid
func IsValidWalletAddress(address string) bool {
	// Must start with 0x
	if !strings.HasPrefix(address, "0x") {
		return false
	}

	// Remove 0x and check length
	address = strings.TrimPrefix(address, "0x")
	if len(address) != 40 {
		return false
	}

	// Should be hex
	matched, _ := regexp.MatchString(`^[a-fA-F0-9]+$`, address)
	return matched
}

// TruncateSensitive truncates sensitive data for logging
func TruncateSensitive(sensitive string, visibleChars int) string {
	if len(sensitive) <= visibleChars {
		return strings.Repeat("*", len(sensitive))
	}

	visible := sensitive[:visibleChars]
	hidden := strings.Repeat("*", len(sensitive)-visibleChars)
	return visible + hidden
}
