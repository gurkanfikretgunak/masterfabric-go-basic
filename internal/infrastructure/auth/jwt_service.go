package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	infraRedis "github.com/masterfabric/masterfabric_go_basic/internal/infrastructure/redis"
	"github.com/masterfabric/masterfabric_go_basic/internal/shared/config"
	domainErr "github.com/masterfabric/masterfabric_go_basic/internal/shared/errors"
)

// TokenPair holds a short-lived access token and long-lived refresh token.
type TokenPair struct {
	AccessToken  string
	RefreshToken string
	ExpiresIn    int64 // seconds until access token expiry
}

// Claims is the JWT claims payload.
type Claims struct {
	UserID string `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// JWTService handles JWT creation and validation plus refresh token storage in
// the cache. It delegates all Redis I/O to CacheHandler so no raw Redis
// client escapes this package boundary.
type JWTService struct {
	cfg   config.JWTConfig
	cache *infraRedis.CacheHandler
}

// NewJWTService creates a JWTService. cache may wrap a nil Redis client
// (tokens won't be blacklisted / refresh tokens won't be stored).
func NewJWTService(cfg config.JWTConfig, cache *infraRedis.CacheHandler) *JWTService {
	return &JWTService{cfg: cfg, cache: cache}
}

// GenerateTokenPair creates an access + refresh token pair for the given user.
func (s *JWTService) GenerateTokenPair(ctx context.Context, userID uuid.UUID, email, role string) (*TokenPair, error) {
	now := time.Now()

	// Access token
	accessClaims := Claims{
		UserID: userID.String(),
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.cfg.AccessTokenTTL)),
		},
	}
	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, accessClaims)
	accessSigned, err := accessToken.SignedString([]byte(s.cfg.Secret))
	if err != nil {
		return nil, fmt.Errorf("sign access token: %w", err)
	}

	// Refresh token (opaque UUID stored in cache as "email:role")
	refreshToken := uuid.New().String()
	if s.cache.Available() {
		key := infraRedis.RefreshTokenKey(userID.String(), refreshToken)
		value := email + ":" + role
		if err := s.cache.Set(ctx, key, value, s.cfg.RefreshTokenTTL); err != nil {
			return nil, fmt.Errorf("store refresh token: %w", err)
		}
	}

	return &TokenPair{
		AccessToken:  accessSigned,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(s.cfg.AccessTokenTTL.Seconds()),
	}, nil
}

// ValidateAccessToken parses and validates a JWT access token.
func (s *JWTService) ValidateAccessToken(ctx context.Context, tokenStr string) (*Claims, error) {
	// Check blacklist first
	if s.cache.Available() {
		blacklisted, err := s.cache.Exists(ctx, infraRedis.TokenBlacklistKey(tokenStr))
		if err == nil && blacklisted {
			return nil, domainErr.ErrTokenInvalid
		}
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(s.cfg.Secret), nil
	})
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, domainErr.ErrTokenExpired
		}
		return nil, domainErr.ErrTokenInvalid
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, domainErr.ErrTokenInvalid
	}
	return claims, nil
}

// RefreshTokens validates a refresh token and issues a new pair.
func (s *JWTService) RefreshTokens(ctx context.Context, userID uuid.UUID, refreshToken string) (*TokenPair, error) {
	if !s.cache.Available() {
		return nil, domainErr.ErrTokenInvalid
	}

	key := infraRedis.RefreshTokenKey(userID.String(), refreshToken)
	value, found, err := s.cache.Get(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("get refresh token: %w", err)
	}
	if !found {
		return nil, domainErr.ErrTokenInvalid
	}

	// value is "email:role"
	parts := strings.SplitN(value, ":", 2)
	email := parts[0]
	role := "user"
	if len(parts) == 2 {
		role = parts[1]
	}

	// Rotate: delete old refresh token
	_ = s.cache.Del(ctx, key)

	return s.GenerateTokenPair(ctx, userID, email, role)
}

// RevokeTokens blacklists the access token and deletes the refresh token.
func (s *JWTService) RevokeTokens(ctx context.Context, userID uuid.UUID, accessToken, refreshToken string) error {
	if !s.cache.Available() {
		return nil
	}

	// Parse to get remaining TTL for blacklist expiry
	claims, err := s.ValidateAccessToken(ctx, accessToken)
	if err == nil && claims != nil {
		ttl := time.Until(claims.ExpiresAt.Time)
		if ttl > 0 {
			_ = s.cache.Set(ctx, infraRedis.TokenBlacklistKey(accessToken), "1", ttl)
		}
	}

	// Delete refresh token
	_ = s.cache.Del(ctx, infraRedis.RefreshTokenKey(userID.String(), refreshToken))
	return nil
}
