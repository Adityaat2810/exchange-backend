package service

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"time"

	"exchange-backedn/proto/gen/auth"
	jwtpkg "exchange-backedn/services/auth/internal/jwt"

	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var (
	ErrUserExists         = errors.New("user already exists")
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrInvalidToken       = errors.New("invalid or expired refresh token")
)

// Test comment for lazygit 
// yo yo honey singh

// AuthService implements the gRPC AuthService
type AuthService struct {
	auth.UnimplementedAuthServiceServer
	db         *sql.DB
	jwtManager *jwtpkg.Manager
}

// NewAuthService creates a new AuthService
func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{
		db:         db,
		jwtManager: jwtpkg.NewManager(),
	}
}

// Signup creates a new user account
func (s *AuthService) Signup(ctx context.Context, req *auth.SignupRequest) (*auth.SignupResponse, error) {
	// Validate input
	if req.Username == "" || req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "username, email, and password are required")
	}

	if len(req.Password) < 8 {
		return nil, status.Error(codes.InvalidArgument, "password must be at least 8 characters")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to hash password")
	}

	// Insert user into database
	var userID int32
	err = s.db.QueryRowContext(ctx,
		`INSERT INTO users (username, email, password_hash) 
		 VALUES ($1, $2, $3) 
		 RETURNING id`,
		req.Username, req.Email, string(hashedPassword),
	).Scan(&userID)

	if err != nil {
		// Check for unique constraint violation
		if isUniqueViolation(err) {
			return nil, status.Error(codes.AlreadyExists, "user with this email or username already exists")
		}
		return nil, status.Errorf(codes.Internal, "failed to create user: %v", err)
	}

	return &auth.SignupResponse{
		UserId:  userID,
		Message: "User created successfully",
	}, nil
}

// Login authenticates a user and returns JWT tokens
func (s *AuthService) Login(ctx context.Context, req *auth.LoginRequest) (*auth.LoginResponse, error) {
	// Validate input
	if req.Email == "" || req.Password == "" {
		return nil, status.Error(codes.InvalidArgument, "email and password are required")
	}

	// Fetch user from database
	var userID int32
	var username, email, passwordHash string
	err := s.db.QueryRowContext(ctx,
		`SELECT id, username, email, password_hash FROM users WHERE email = $1`,
		req.Email,
	).Scan(&userID, &username, &email, &passwordHash)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.Unauthenticated, "invalid email or password")
		}
		return nil, status.Errorf(codes.Internal, "failed to fetch user: %v", err)
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(passwordHash), []byte(req.Password)); err != nil {
		return nil, status.Error(codes.Unauthenticated, "invalid email or password")
	}

	// Generate token pair
	tokenPair, err := s.jwtManager.GenerateTokenPair(userID, email, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate tokens: %v", err)
	}

	// Store refresh token in database
	if err := s.storeRefreshToken(ctx, userID, tokenPair.RefreshToken); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store refresh token: %v", err)
	}

	return &auth.LoginResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// RefreshToken generates new access and refresh tokens using a valid refresh token
func (s *AuthService) RefreshToken(ctx context.Context, req *auth.RefreshTokenRequest) (*auth.RefreshTokenResponse, error) {
	if req.RefreshToken == "" {
		return nil, status.Error(codes.InvalidArgument, "refresh token is required")
	}

	// Hash the refresh token to look it up
	tokenHash := hashToken(req.RefreshToken)

	// Find the refresh token and associated user
	var userID int32
	var tokenID int32
	var expiresAt time.Time
	err := s.db.QueryRowContext(ctx,
		`SELECT rt.id, rt.user_id, rt.expires_at 
		 FROM refresh_tokens rt 
		 WHERE rt.token_hash = $1`,
		tokenHash,
	).Scan(&tokenID, &userID, &expiresAt)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, status.Error(codes.Unauthenticated, "invalid refresh token")
		}
		return nil, status.Errorf(codes.Internal, "failed to validate refresh token: %v", err)
	}

	// Check if token is expired
	if time.Now().After(expiresAt) {
		// Delete expired token
		_, _ = s.db.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE id = $1`, tokenID)
		return nil, status.Error(codes.Unauthenticated, "refresh token has expired")
	}

	// Fetch user details
	var username, email string
	err = s.db.QueryRowContext(ctx,
		`SELECT username, email FROM users WHERE id = $1`,
		userID,
	).Scan(&username, &email)

	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to fetch user: %v", err)
	}

	// Generate new token pair (token rotation)
	tokenPair, err := s.jwtManager.GenerateTokenPair(userID, email, username)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to generate tokens: %v", err)
	}

	// Delete old refresh token and store new one (atomic rotation)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to start transaction: %v", err)
	}
	defer tx.Rollback()

	// Delete old token
	if _, err := tx.ExecContext(ctx, `DELETE FROM refresh_tokens WHERE id = $1`, tokenID); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to delete old refresh token: %v", err)
	}

	// Store new refresh token
	newTokenHash := hashToken(tokenPair.RefreshToken)
	expiresAt = time.Now().Add(s.jwtManager.GetRefreshTokenExpiry())
	if _, err := tx.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, newTokenHash, expiresAt,
	); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to store new refresh token: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, status.Errorf(codes.Internal, "failed to commit transaction: %v", err)
	}

	return &auth.RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// storeRefreshToken stores a refresh token in the database
func (s *AuthService) storeRefreshToken(ctx context.Context, userID int32, token string) error {
	tokenHash := hashToken(token)
	expiresAt := time.Now().Add(s.jwtManager.GetRefreshTokenExpiry())

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO refresh_tokens (user_id, token_hash, expires_at) VALUES ($1, $2, $3)`,
		userID, tokenHash, expiresAt,
	)
	return err
}

// hashToken creates a SHA-256 hash of a token
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}

// isUniqueViolation checks if the error is a unique constraint violation
func isUniqueViolation(err error) bool {
	// PostgreSQL unique violation error code is 23505
	return err != nil && (contains(err.Error(), "23505") || contains(err.Error(), "duplicate key"))
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
