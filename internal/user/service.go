package user

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"

	"github.com/DekuMidBak/gofintracker/internal/user/auth"
)

var (
	ErrInvalidEmail       = errors.New("invalid email")
	ErrInvalidPassword    = errors.New("invalid password")
	ErrInvalidCredentials = errors.New("invalid credentials")
)

type TokenManager interface {
	Generate(userID string) (string, error)
	Validate(accessToken string) (string, error)
}

type Service struct {
	repository   Repository
	tokenManager TokenManager
}

type RegisterParams struct {
	Email    string
	Password string
}

type LoginParams struct {
	Email    string
	Password string
}

type AuthResult struct {
	User        User
	AccessToken string
}

func NewService(repository Repository, tokenManager TokenManager) *Service {
	return &Service{
		repository:   repository,
		tokenManager: tokenManager,
	}
}

func (s *Service) Register(ctx context.Context, params RegisterParams) (AuthResult, error) {
	email, err := normalizeEmail(params.Email)
	if err != nil {
		return AuthResult{}, err
	}

	if params.Password == "" {
		return AuthResult{}, ErrInvalidPassword
	}

	passwordHash, err := auth.HashPassword(params.Password)
	if err != nil {
		return AuthResult{}, fmt.Errorf("hash password: %w", err)
	}

	created, err := s.repository.Create(ctx, CreateUserParams{
		Email:        email,
		PasswordHash: passwordHash,
	})
	if err != nil {
		return AuthResult{}, err
	}

	accessToken, err := s.tokenManager.Generate(created.ID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate access token: %w", err)
	}

	return AuthResult{
		User:        created,
		AccessToken: accessToken,
	}, nil
}

func (s *Service) Login(ctx context.Context, params LoginParams) (AuthResult, error) {
	email, err := normalizeEmail(params.Email)
	if err != nil {
		return AuthResult{}, err
	}

	found, err := s.repository.FindByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return AuthResult{}, ErrInvalidCredentials
		}

		return AuthResult{}, err
	}

	if err := auth.CheckPassword(found.PasswordHash, params.Password); err != nil {
		return AuthResult{}, ErrInvalidCredentials
	}

	accessToken, err := s.tokenManager.Generate(found.ID)
	if err != nil {
		return AuthResult{}, fmt.Errorf("generate access token: %w", err)
	}

	return AuthResult{
		User:        found,
		AccessToken: accessToken,
	}, nil
}

func (s *Service) ValidateToken(accessToken string) (string, error) {
	return s.tokenManager.Validate(accessToken)
}

func normalizeEmail(email string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(email))
	if normalized == "" {
		return "", ErrInvalidEmail
	}

	parsed, err := mail.ParseAddress(normalized)
	if err != nil || parsed.Address != normalized {
		return "", ErrInvalidEmail
	}

	return normalized, nil
}
