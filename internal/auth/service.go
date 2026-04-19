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

var (
	ErrInvalidToken       = errors.New("invalid token")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrTokenNotFound      = errors.New("token not found")
	ErrTokenKeyMissing    = errors.New("missing key in token claims")
)

type AuthService struct {
	jwtSecret        []byte
	accessTokenTTL   time.Duration
	refreshTokenTTL  time.Duration
	refreshTokenRepo *RefreshTokenRepo
	logger           *zerolog.Logger
}

func NewAuthService(jwtSecret string, accessTokenTTL int, refreshTokenTTL int, refreshTokenRepo *RefreshTokenRepo, logger *zerolog.Logger) *AuthService {
	return &AuthService{
		jwtSecret:        []byte(jwtSecret),
		accessTokenTTL:   time.Duration(accessTokenTTL) * time.Minute,
		refreshTokenTTL:  time.Duration(refreshTokenTTL) * time.Hour,
		refreshTokenRepo: refreshTokenRepo,
		logger:           logger,
	}
}

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

func (s *AuthService) RefreshToken(ctx context.Context, token string) (string, string, error) {
	// validate token exp, key, algorithm
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
	// validate token claims
	email, ok := claims["email"].(string)
	if !ok {
		return "", "", ErrTokenKeyMissing
	}
	// rotate the refresh token by deleting the old one and returning a new access-refresh pair
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
