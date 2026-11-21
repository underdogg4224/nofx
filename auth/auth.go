package auth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"golang.org/x/crypto/bcrypt"
)

const (
	tokenPurposeAccess    = "access"
	tokenPurposeOTPVerify = "otp_verify"
	otpChallengeTTL       = 5 * time.Minute
)

// JWTSecret JWT密钥，将从配置中动态设置
var JWTSecret []byte

// AdminMode 管理员模式标志
var AdminMode bool = false

// OTPIssuer OTP发行者名称
const OTPIssuer = "nofxAI"

// SetJWTSecret 设置JWT密钥
func SetJWTSecret(secret string) {
	JWTSecret = []byte(secret)
}

// SetAdminMode 设置管理员模式
func SetAdminMode(enabled bool) {
	AdminMode = enabled
}

// IsAdminMode 检查是否为管理员模式
func IsAdminMode() bool {
	return AdminMode
}

// GenerateRandomSecret generates a cryptographically secure hex-encoded secret.
func GenerateRandomSecret(numBytes int) (string, error) {
	if numBytes < 16 {
		numBytes = 32
	}
	buf := make([]byte, numBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

// Claims JWT声明
type Claims struct {
	UserID  string `json:"user_id"`
	Email   string `json:"email"`
	Purpose string `json:"purpose,omitempty"`
	jwt.RegisteredClaims
}

// HashPassword 哈希密码
func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	return string(bytes), err
}

// CheckPassword 验证密码
func CheckPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateOTPSecret 生成OTP密钥
func GenerateOTPSecret() (string, error) {
	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      OTPIssuer,
		AccountName: uuid.New().String(),
	})
	if err != nil {
		return "", err
	}

	return key.Secret(), nil
}

// VerifyOTP 验证OTP码
func VerifyOTP(secret, code string) bool {
	return totp.Validate(code, secret)
}

// GenerateJWT 生成JWT token
func GenerateJWT(userID, email string) (string, error) {
	if len(JWTSecret) == 0 {
		return "", fmt.Errorf("JWT密钥未配置")
	}

	claims := Claims{
		UserID:  userID,
		Email:   email,
		Purpose: tokenPurposeAccess,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24小时过期
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    OTPIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// GenerateOTPChallenge issues a short-lived signed token proving password authentication succeeded.
func GenerateOTPChallenge(userID, email string) (string, error) {
	if len(JWTSecret) == 0 {
		return "", fmt.Errorf("JWT密钥未配置")
	}

	now := time.Now()
	claims := Claims{
		UserID:  userID,
		Email:   email,
		Purpose: tokenPurposeOTPVerify,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(otpChallengeTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    OTPIssuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// ValidateJWT 验证JWT token
func ValidateJWT(tokenString string) (*Claims, error) {
	if len(JWTSecret) == 0 {
		return nil, fmt.Errorf("JWT密钥未配置")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return JWTSecret, nil
	}, jwt.WithIssuer(OTPIssuer))

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		// Reject OTP challenge tokens from being used as access tokens.
		if claims.Purpose == tokenPurposeOTPVerify {
			return nil, fmt.Errorf("无效的token用途")
		}
		return claims, nil
	}

	return nil, fmt.Errorf("无效的token")
}

// ValidateOTPChallenge verifies a short-lived OTP challenge token for the expected user.
func ValidateOTPChallenge(tokenString, expectedUserID string) (*Claims, error) {
	if len(JWTSecret) == 0 {
		return nil, fmt.Errorf("JWT密钥未配置")
	}

	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("意外的签名方法: %v", token.Header["alg"])
		}
		return JWTSecret, nil
	}, jwt.WithIssuer(OTPIssuer))

	if err != nil {
		return nil, err
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid || claims.Purpose != tokenPurposeOTPVerify || claims.UserID != expectedUserID {
		return nil, fmt.Errorf("无效的OTP验证会话")
	}

	return claims, nil
}

// GetOTPQRCodeURL 获取OTP二维码URL
func GetOTPQRCodeURL(secret, email string) string {
	return fmt.Sprintf("otpauth://totp/%s:%s?secret=%s&issuer=%s", OTPIssuer, email, secret, OTPIssuer)
}
