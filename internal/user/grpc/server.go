package grpc

import (
	"context"
	"errors"

	userv1 "github.com/DekuMidBak/gofintracker/gen/go/user/v1"
	"github.com/DekuMidBak/gofintracker/internal/user"
	"github.com/DekuMidBak/gofintracker/internal/user/auth"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type Service interface {
	Register(ctx context.Context, params user.RegisterParams) (user.AuthResult, error)
	Login(ctx context.Context, params user.LoginParams) (user.AuthResult, error)
	ValidateToken(accessToken string) (string, error)
}

type Server struct {
	userv1.UnimplementedUserServiceServer

	service Service
}

var _ userv1.UserServiceServer = (*Server)(nil)

func NewServer(service Service) *Server {
	return &Server{service: service}
}

func (s *Server) Register(ctx context.Context, req *userv1.RegisterRequest) (*userv1.RegisterResponse, error) {
	result, err := s.service.Register(ctx, user.RegisterParams{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &userv1.RegisterResponse{
		UserId:      result.User.ID,
		AccessToken: result.AccessToken,
		CreatedAt:   timestamppb.New(result.User.CreatedAt),
	}, nil
}

func (s *Server) Login(ctx context.Context, req *userv1.LoginRequest) (*userv1.LoginResponse, error) {
	result, err := s.service.Login(ctx, user.LoginParams{
		Email:    req.GetEmail(),
		Password: req.GetPassword(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return &userv1.LoginResponse{
		UserId:      result.User.ID,
		AccessToken: result.AccessToken,
	}, nil
}

func (s *Server) ValidateToken(_ context.Context, req *userv1.ValidateTokenRequest) (*userv1.ValidateTokenResponse, error) {
	userID, err := s.service.ValidateToken(req.GetAccessToken())
	if err != nil {
		return nil, mapError(err)
	}

	return &userv1.ValidateTokenResponse{
		UserId: userID,
	}, nil
}

func mapError(err error) error {
	switch {
	case errors.Is(err, user.ErrInvalidEmail),
		errors.Is(err, user.ErrInvalidPassword):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, user.ErrEmailExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, user.ErrInvalidCredentials),
		errors.Is(err, auth.ErrInvalidToken):
		return status.Error(codes.Unauthenticated, err.Error())
	default:
		return status.Error(codes.Internal, "internal error")
	}
}
