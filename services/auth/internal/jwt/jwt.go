package jwt

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrInvalidToken = errors.New("invalid token")
	ErrExpiredToken = errors.New("token has expired")
)

// Claims represents the JWT claims for access tokens
type Claims struct {
	UserID   int32  `json:"user_id"`
	Email    string `json:"email"`
	Username string `json:"username"`
	jwt.RegisteredClaims
}

// TokenPair contains access and refresh tokens
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // Access token expiry in seconds
}

// Config holds JWT configuration
type Config struct {
	Secret             string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

// Manager handles JWT token operations
type Manager struct {
	config Config
}

// NewManager creates a new JWT manager
func NewManager() *Manager {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "dev-secret-change-in-production" // Default for development
	}

	return &Manager{
		config: Config{
			Secret:             secret,
			AccessTokenExpiry:  15 * time.Minute,   // Access token expires in 15 minutes
			RefreshTokenExpiry: 7 * 24 * time.Hour, // Refresh token expires in 7 days
			Issuer:             "exchange-auth-service",
		},
	}
}

// GenerateTokenPair generates both access and refresh tokens
func (m *Manager) GenerateTokenPair(userID int32, email, username string) (*TokenPair, error) {
	accessToken, err := m.generateAccessToken(userID, email, username)
	if err != nil {
		return nil, err
	}

	refreshToken, err := m.generateRefreshToken()
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(m.config.AccessTokenExpiry.Seconds()),
	}, nil
}

// generateAccessToken creates a new JWT access token
func (m *Manager) generateAccessToken(userID int32, email, username string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		Email:    email,
		Username: username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(m.config.AccessTokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.config.Issuer,
			Subject:   email,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(m.config.Secret))
}

// generateRefreshToken creates a cryptographically secure random refresh token
func (m *Manager) generateRefreshToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// ValidateAccessToken validates an access token and returns the claims
func (m *Manager) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return []byte(m.config.Secret), nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrExpiredToken
		}
		return nil, ErrInvalidToken
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidToken
	}

	return claims, nil
}

// GetRefreshTokenExpiry returns the refresh token expiry duration
func (m *Manager) GetRefreshTokenExpiry() time.Duration {
	return m.config.RefreshTokenExpiry
}

// GetAccessTokenExpiry returns the access token expiry in seconds
func (m *Manager) GetAccessTokenExpiry() int64 {
	return int64(m.config.AccessTokenExpiry.Seconds())
}
