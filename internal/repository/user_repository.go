package repository

import (
	"fmt"
	"go-digital-wallet/internal/entity"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(user *entity.User) error
	GetByEmail(email string) (*entity.User, error)
	GetByID(id uuid.UUID) (*entity.User, error)
}

type UserRepositoryImpl struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewUserRepository(db *gorm.DB, logger *logrus.Logger) UserRepository {
	return &UserRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *UserRepositoryImpl) Create(user *entity.User) error {
	if err := r.db.Create(user).Error; err != nil {
		r.logger.WithError(err).WithField("email", user.Email).Error("Failed to create user")
		return fmt.Errorf("failed to create user: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"user_id": user.ID,
		"name":    user.Name,
		"email":   user.Email,
	}).Info("User created successfully")

	return nil
}

func (r *UserRepositoryImpl) GetByEmail(email string) (*entity.User, error) {
	var user entity.User
	err := r.db.Where("email = ?", email).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		r.logger.WithError(err).WithField("email", email).Error("Failed to get user by email")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}

func (r *UserRepositoryImpl) GetByID(id uuid.UUID) (*entity.User, error) {
	var user entity.User
	err := r.db.Where("id = ?", id).First(&user).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("user not found")
		}
		r.logger.WithError(err).WithField("user_id", id).Error("Failed to get user by ID")
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return &user, nil
}
