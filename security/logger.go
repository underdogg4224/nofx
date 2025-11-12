package security

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"
)

// SecurityEventType represents different types of security events
type SecurityEventType string

const (
	EventLoginSuccess        SecurityEventType = "LOGIN_SUCCESS"
	EventLoginFailure        SecurityEventType = "LOGIN_FAILURE"
	EventLoginBruteForce     SecurityEventType = "LOGIN_BRUTE_FORCE"
	EventRegistration        SecurityEventType = "REGISTRATION"
	EventOTPVerifySuccess    SecurityEventType = "OTP_VERIFY_SUCCESS"
	EventOTPVerifyFailure    SecurityEventType = "OTP_VERIFY_FAILURE"
	EventOTPBruteForce       SecurityEventType = "OTP_BRUTE_FORCE"
	EventUnauthorizedAccess  SecurityEventType = "UNAUTHORIZED_ACCESS"
	EventTokenExpired        SecurityEventType = "TOKEN_EXPIRED"
	EventTokenInvalid        SecurityEventType = "TOKEN_INVALID"
	EventRateLimitExceeded   SecurityEventType = "RATE_LIMIT_EXCEEDED"
	EventAPIKeyAccess        SecurityEventType = "API_KEY_ACCESS"
	EventPasswordChange      SecurityEventType = "PASSWORD_CHANGE"
	EventAccountLocked       SecurityEventType = "ACCOUNT_LOCKED"
	EventSuspiciousActivity  SecurityEventType = "SUSPICIOUS_ACTIVITY"
)

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	Timestamp   time.Time         `json:"timestamp"`
	EventType   SecurityEventType `json:"event_type"`
	UserID      string            `json:"user_id,omitempty"`
	Email       string            `json:"email,omitempty"`
	IPAddress   string            `json:"ip_address,omitempty"`
	UserAgent   string            `json:"user_agent,omitempty"`
	Success     bool              `json:"success"`
	Message     string            `json:"message,omitempty"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// SecurityLogger handles logging of security events
type SecurityLogger struct {
	logFile *os.File
	enabled bool
}

var defaultLogger *SecurityLogger

// InitializeSecurityLogger initializes the security event logger
func InitializeSecurityLogger(logPath string, enabled bool) error {
	if !enabled {
		defaultLogger = &SecurityLogger{enabled: false}
		return nil
	}

	if logPath == "" {
		logPath = "logs/security.log"
	}

	// Create logs directory if it doesn't exist
	if err := os.MkdirAll("logs", 0700); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}

	// Open log file with append mode
	file, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("failed to open security log file: %w", err)
	}

	defaultLogger = &SecurityLogger{
		logFile: file,
		enabled: true,
	}

	return nil
}

// LogEvent logs a security event
func LogEvent(event SecurityEvent) {
	if defaultLogger == nil || !defaultLogger.enabled {
		return
	}

	event.Timestamp = time.Now()

	// Marshal to JSON
	jsonData, err := json.Marshal(event)
	if err != nil {
		log.Printf("Failed to marshal security event: %v", err)
		return
	}

	// Write to log file
	if _, err := defaultLogger.logFile.Write(append(jsonData, '\n')); err != nil {
		log.Printf("Failed to write security event: %v", err)
	}

	// Also log critical events to stdout
	if !event.Success && isCriticalEvent(event.EventType) {
		log.Printf("[SECURITY ALERT] %s: %s (User: %s, IP: %s)",
			event.EventType, event.Message, event.Email, event.IPAddress)
	}
}

// LogLoginAttempt logs a login attempt
func LogLoginAttempt(email, ipAddress, userAgent string, success bool, reason string) {
	eventType := EventLoginSuccess
	if !success {
		eventType = EventLoginFailure
	}

	LogEvent(SecurityEvent{
		EventType: eventType,
		Email:     email,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
		Message:   reason,
	})
}

// LogOTPAttempt logs an OTP verification attempt
func LogOTPAttempt(userID, email, ipAddress string, success bool) {
	eventType := EventOTPVerifySuccess
	if !success {
		eventType = EventOTPVerifyFailure
	}

	LogEvent(SecurityEvent{
		EventType: eventType,
		UserID:    userID,
		Email:     email,
		IPAddress: ipAddress,
		Success:   success,
	})
}

// LogUnauthorizedAccess logs an unauthorized access attempt
func LogUnauthorizedAccess(ipAddress, userAgent, endpoint string) {
	LogEvent(SecurityEvent{
		EventType: EventUnauthorizedAccess,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   false,
		Message:   fmt.Sprintf("Unauthorized access to %s", endpoint),
	})
}

// LogRateLimitExceeded logs when rate limit is exceeded
func LogRateLimitExceeded(ipAddress, endpoint string) {
	LogEvent(SecurityEvent{
		EventType: EventRateLimitExceeded,
		IPAddress: ipAddress,
		Success:   false,
		Message:   fmt.Sprintf("Rate limit exceeded for %s", endpoint),
	})
}

// isCriticalEvent checks if an event type is critical
func isCriticalEvent(eventType SecurityEventType) bool {
	criticalEvents := []SecurityEventType{
		EventLoginBruteForce,
		EventOTPBruteForce,
		EventUnauthorizedAccess,
		EventSuspiciousActivity,
		EventAccountLocked,
	}

	for _, critical := range criticalEvents {
		if eventType == critical {
			return true
		}
	}
	return false
}

// Close closes the security logger
func CloseSecurityLogger() error {
	if defaultLogger != nil && defaultLogger.logFile != nil {
		return defaultLogger.logFile.Close()
	}
	return nil
}
