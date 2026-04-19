package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/JBK2116/vaulthook/internal/config"
	"github.com/golang-jwt/jwt/v5"
	"github.com/rs/zerolog"
)

// Sentinel errors returned by AuthService methods.
var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTokenNotFound      = errors.New("token not found")
	ErrTokenKeyMissing    = errors.New("missing key in token claims")
)

// AuthService handles JWT issuance, validation, and refresh token rotation.
type AuthService struct {
	jwtSecret        []byte
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	refreshTokenRepo *RefreshTokenRepo
	logger           *zerolog.Logger
}

// NewAuthService returns an AuthService configured with the provided secret,
// token TTLs, repository, and logger.
//
// accessTokenTTL is interpreted as minutes; refreshTokenTTL as hours.
func NewAuthService(jwtSecret string, accessTokenTTL int, refreshTokenTTL int, refreshTokenRepo *RefreshTokenRepo, logger *zerolog.Logger) *AuthService {
	return &AuthService{
		jwtSecret:        []byte(jwtSecret),
		accessTokenTTL:   time.Duration(accessTokenTTL) * time.Minute,
		refreshTokenTTL:  time.Duration(refreshTokenTTL) * time.Hour,
		refreshTokenRepo: refreshTokenRepo,
		logger:           logger,
	}
}

// Login validates the provided credentials against the configured user,
// issues a new access and refresh token pair, and persists the refresh token.
//
// It returns ErrInvalidCredentials if the email or password do not match.
func (s *AuthService) Login(ctx context.Context, email string, password string) (string, string, error) {
	if email != config.Envs.USER_EMAIL || password != config.Envs.USER_PASSWORD {
		return "", "", ErrInvalidCredentials
	}

	now := time.Now()
	accessStr, err := s.generateAccessToken(email, now.Add(s.accessTokenTTL), now)
	if err != nil {
		return "", "", err
	}

	refreshStr, err := s.generateRefreshToken(email, now.Add(s.refreshTokenTTL), now)
	if err != nil {
		return "", "", err
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*5)
	defer cancel()

	_, err = s.refreshTokenRepo.Create(ctx, refreshStr, now.Add(s.refreshTokenTTL), now)
	if err != nil {
		return "", "", err
	}

	return accessStr, refreshStr, nil
}

// RefreshToken validates the provided refresh token, rotates it by deleting
// the old record and persisting a new one, and returns a fresh access and
// refresh token pair.
//
// If the token does not exist in the database, ErrTokenNotFound is returned
// and no deletion is attempted. If the token exists but fails validation,
// it is deleted before returning the error to prevent reuse.
func (s *AuthService) RefreshToken(ctx context.Context, token string) (string, string, error) {
	claims, err := s.validateRefreshToken(ctx, token)
	if err != nil {
		if errors.Is(err, ErrTokenNotFound) {
			return "", "", ErrTokenNotFound
		}
		if delErr := s.refreshTokenRepo.Delete(ctx, token); delErr != nil {
			s.logger.Error().Stack().Err(delErr).Msg("error deleting token from database")
		}
		return "", "", err
	}

	email, ok := claims["email"].(string)
	if !ok {
		return "", "", ErrTokenKeyMissing
	}

	if err := s.refreshTokenRepo.Delete(ctx, token); err != nil {
		return "", "", err
	}

	now := time.Now()
	accessStr, err := s.generateAccessToken(email, now.Add(s.accessTokenTTL), now)
	if err != nil {
		return "", "", err
	}

	refreshStr, err := s.generateRefreshToken(email, now.Add(s.refreshTokenTTL), now)
	if err != nil {
		return "", "", err
	}

	_, err = s.refreshTokenRepo.Create(ctx, refreshStr, now.Add(s.refreshTokenTTL), now)
	if err != nil {
		return "", "", err
	}

	return accessStr, refreshStr, nil
}

// generateAccessToken creates a signed HS256 JWT access token for the given
// email, expiry, and issued-at time.
func (s *AuthService) generateAccessToken(email string, exp time.Time, iat time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":   fmt.Sprintf("access token | %s", email),
		"email": email,
		"exp":   exp.Unix(),
		"iat":   iat.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

// generateRefreshToken creates a signed HS256 JWT refresh token for the given
// email, expiry, and issued-at time.
func (s *AuthService) generateRefreshToken(email string, exp time.Time, iat time.Time) (string, error) {
	claims := jwt.MapClaims{
		"sub":   fmt.Sprintf("refresh token | %s", email),
		"email": email,
		"exp":   exp.Unix(),
		"iat":   iat.Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

// validateAccessToken parses and validates a JWT access token, enforcing
// the HS256 signing method. It returns the token claims on success.
//
// It returns jwt.ErrTokenExpired if the token has expired, or ErrInvalidToken
// for any other validation failure.
func (s *AuthService) validateAccessToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, jwt.ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}

// validateRefreshToken checks that the token exists in the database, then
// parses and validates it as a signed HS256 JWT. It returns the token claims
// on success.
//
// It returns ErrTokenNotFound if the token is absent from the database,
// jwt.ErrTokenExpired if expired, or ErrInvalidToken for any other failure.
func (s *AuthService) validateRefreshToken(ctx context.Context, tokenStr string) (jwt.MapClaims, error) {
	exists, err := s.refreshTokenRepo.Exists(ctx, tokenStr)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, ErrTokenNotFound
	}

	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		return s.jwtSecret, nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, jwt.ErrTokenExpired
		}
		return nil, ErrInvalidToken
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		return claims, nil
	}
	return nil, ErrInvalidToken
}
