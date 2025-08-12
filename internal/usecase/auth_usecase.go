package usecase

import (
	"go-digital-wallet/internal/commons/response"
	"go-digital-wallet/internal/entity"
	"go-digital-wallet/internal/params"
	"go-digital-wallet/internal/repository"
	"go-digital-wallet/pkg/token"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/bcrypt"
)

type AuthUsecase interface {
	Register(req *params.RegisterRequest) (*params.AuthResponse, *response.CustomError)
	Login(req *params.LoginRequest) (*params.AuthResponse, *response.CustomError)
}

type AuthUsecaseImpl struct {
	userRepo   repository.UserRepository
	logger     *logrus.Logger
	jwtManager *token.TokenManager
}

func NewAuthUsecase(userRepo repository.UserRepository, logger *logrus.Logger, jwtManager *token.TokenManager) AuthUsecase {
	return &AuthUsecaseImpl{
		userRepo:   userRepo,
		logger:     logger,
		jwtManager: jwtManager,
	}
}

func (s *AuthUsecaseImpl) Register(req *params.RegisterRequest) (*params.AuthResponse, *response.CustomError) {
	// Check if user already exists by email
	if _, err := s.userRepo.GetByEmail(req.Email); err == nil {
		s.logger.WithField("email", req.Email).Warn("Registration attempt with existing email")
		return nil, response.BadRequestError("user with this email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.WithError(err).Error("Failed to hash password")
		return nil, response.GeneralError("failed to hash password")
	}

	// Create user
	user := &entity.User{
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hashedPassword),
	}

	if err := s.userRepo.Create(user); err != nil {
		s.logger.WithError(err).WithField("email", req.Email).Error("Failed to create user")
		return nil, response.RepositoryError("failed to create user")
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID).Error("Failed to generate token")
		return nil, response.GeneralError("failed to generate token")
	}

	response := &params.AuthResponse{
		Token: token,
	}
	response.User.ID = user.ID
	response.User.Name = user.Name
	response.User.Email = user.Email

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"name":    user.Name,
		"email":   user.Email,
	}).Info("User registered successfully")

	return response, nil
}

func (s *AuthUsecaseImpl) Login(req *params.LoginRequest) (*params.AuthResponse, *response.CustomError) {
	// Get user by email
	user, err := s.userRepo.GetByEmail(req.Email)
	if err != nil {
		s.logger.WithField("email", req.Email).Warn("Login attempt with non-existing email")
		return nil, response.BadRequestError("invalid email or password")
	}

	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.logger.WithFields(logrus.Fields{
			"user_id": user.ID,
			"email":   req.Email,
		}).Warn("Login attempt with invalid password")
		return nil, response.BadRequestError("invalid email or password")
	}

	// Generate JWT token
	token, err := s.jwtManager.GenerateToken(user.ID)
	if err != nil {
		s.logger.WithError(err).WithField("user_id", user.ID).Error("Failed to generate token")
		return nil, response.GeneralError("failed to generate token")
	}

	response := &params.AuthResponse{
		Token: token,
	}
	response.User.ID = user.ID
	response.User.Name = user.Name
	response.User.Email = user.Email

	s.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"name":    user.Name,
		"email":   user.Email,
	}).Info("User logged in successfully")

	return response, nil
}
