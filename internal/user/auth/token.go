package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var (
	ErrEmptyTokenSecret = errors.New("token secret is empty")
	ErrInvalidTokenTTL  = errors.New("token ttl must be positive")
	ErrEmptyUserID      = errors.New("user id is empty")
	ErrInvalidToken     = errors.New("invalid token")
)

type TokenManager struct {
	secret []byte
	ttl    time.Duration
}

type accessClaims struct {
	jwt.RegisteredClaims
}

func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	if secret == "" {
		return nil, ErrEmptyTokenSecret
	}

	if ttl <= 0 {
		return nil, ErrInvalidTokenTTL
	}

	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
	}, nil
}

func (m *TokenManager) Generate(userID string) (string, error) {
	if userID == "" {
		return "", ErrEmptyUserID
	}

	now := time.Now()
	claims := accessClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(m.secret)
	if err != nil {
		return "", fmt.Errorf("sign access token: %w", err)
	}

	return signed, nil
}

func (m *TokenManager) Validate(accessToken string) (string, error) {
	claims := accessClaims{}
	parsed, err := jwt.ParseWithClaims(
		accessToken,
		&claims,
		func(token *jwt.Token) (any, error) {
			return m.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrInvalidToken, err)
	}

	if !parsed.Valid || claims.Subject == "" {
		return "", ErrInvalidToken
	}

	return claims.Subject, nil
}
