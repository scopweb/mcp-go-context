package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret     string        `json:"secret"`
	Expiry     time.Duration `json:"expiry"`
	Issuer     string        `json:"issuer"`
	Algorithm  string        `json:"algorithm"`
}

// JWTClaims represents the JWT payload
type JWTClaims struct {
	Subject   string `json:"sub"`
	Issuer    string `json:"iss"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	NotBefore int64  `json:"nbf"`
	JTI       string `json:"jti"` // JWT ID for revocation support
}

// JWTManager handles JWT operations
type JWTManager struct {
	config JWTConfig
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(config JWTConfig) *JWTManager {
	if config.Algorithm == "" {
		config.Algorithm = "HS256"
	}
	if config.Issuer == "" {
		config.Issuer = "mcp-go-context"
	}
	if config.Expiry == 0 {
		config.Expiry = time.Hour
	}
	
	return &JWTManager{
		config: config,
	}
}

// GenerateToken creates a new JWT token
func (j *JWTManager) GenerateToken(subject string) (string, error) {
	if j.config.Secret == "" {
		return "", fmt.Errorf("JWT secret not configured")
	}

	now := time.Now()
	claims := JWTClaims{
		Subject:   subject,
		Issuer:    j.config.Issuer,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(j.config.Expiry).Unix(),
		NotBefore: now.Unix(),
		JTI:       generateJTI(),
	}

	// Create header
	header := map[string]interface{}{
		"alg": j.config.Algorithm,
		"typ": "JWT",
	}

	// Encode header and claims
	headerJSON, err := json.Marshal(header)
	if err != nil {
		return "", fmt.Errorf("failed to marshal header: %w", err)
	}

	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("failed to marshal claims: %w", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsJSON)

	// Create signature
	message := headerB64 + "." + claimsB64
	signature, err := j.sign(message)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return message + "." + signature, nil
}

// ValidateToken validates and parses a JWT token
func (j *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	if j.config.Secret == "" {
		return nil, fmt.Errorf("JWT secret not configured")
	}

	// Split token into parts
	parts := strings.Split(tokenString, ".")
	if len(parts) != 3 {
		return nil, fmt.Errorf("invalid token format")
	}

	headerB64, claimsB64, signatureB64 := parts[0], parts[1], parts[2]

	// Verify signature
	message := headerB64 + "." + claimsB64
	expectedSig, err := j.sign(message)
	if err != nil {
		return nil, fmt.Errorf("failed to generate signature: %w", err)
	}

	if signatureB64 != expectedSig {
		return nil, fmt.Errorf("invalid signature")
	}

	// Decode and validate header
	headerJSON, err := base64.RawURLEncoding.DecodeString(headerB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode header: %w", err)
	}

	var header map[string]interface{}
	if err := json.Unmarshal(headerJSON, &header); err != nil {
		return nil, fmt.Errorf("failed to unmarshal header: %w", err)
	}

	if header["alg"] != j.config.Algorithm {
		return nil, fmt.Errorf("unsupported algorithm: %v", header["alg"])
	}

	// Decode claims
	claimsJSON, err := base64.RawURLEncoding.DecodeString(claimsB64)
	if err != nil {
		return nil, fmt.Errorf("failed to decode claims: %w", err)
	}

	var claims JWTClaims
	if err := json.Unmarshal(claimsJSON, &claims); err != nil {
		return nil, fmt.Errorf("failed to unmarshal claims: %w", err)
	}

	// Validate claims
	now := time.Now().Unix()
	
	if claims.ExpiresAt < now {
		return nil, fmt.Errorf("token expired")
	}
	
	if claims.NotBefore > now {
		return nil, fmt.Errorf("token not yet valid")
	}
	
	if claims.Issuer != j.config.Issuer {
		return nil, fmt.Errorf("invalid issuer")
	}

	return &claims, nil
}

// IsEnabled returns true if JWT authentication is enabled
func (j *JWTManager) IsEnabled() bool {
	return j.config.Secret != ""
}

// sign creates HMAC-SHA256 signature for the message
func (j *JWTManager) sign(message string) (string, error) {
	mac := hmac.New(sha256.New, []byte(j.config.Secret))
	mac.Write([]byte(message))
	signature := mac.Sum(nil)
	return base64.RawURLEncoding.EncodeToString(signature), nil
}

// generateJTI creates a unique JWT ID
func generateJTI() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", fmt.Errorf("missing authorization header")
	}

	if !strings.HasPrefix(authHeader, "Bearer ") {
		return "", fmt.Errorf("invalid authorization header format")
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == "" {
		return "", fmt.Errorf("missing token")
	}

	return token, nil
}