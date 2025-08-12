package usecase_test

import (
	"context"
	"errors"
	"go-digital-wallet/internal/entity"
	"go-digital-wallet/internal/params"
	"go-digital-wallet/internal/repository"
	"go-digital-wallet/internal/usecase"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

func setupTest(t *testing.T) (*repository.MockWalletRepository, *redis.Client, usecase.WalletUsecase) {
	mockRepo := new(repository.MockWalletRepository)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	wu := usecase.NewWalletUsecase(mockRepo, logger, rdb)

	return mockRepo, rdb, wu
}

func TestCreateWallet_Success(t *testing.T) {
	mockRepo, _, uc := setupTest(t)

	userID := uuid.New()
	req := &params.CreateWalletRequest{
		UserID:   userID,
		Currency: "IDR",
	}

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Wallet")).Return(nil)

	resp, err := uc.CreateWallet(context.Background(), req)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, userID, resp.UserID)

	mockRepo.AssertExpectations(t)
}

func TestCreateWallet_Fail(t *testing.T) {
	mockRepo, _, uc := setupTest(t)

	userID := uuid.New()
	req := &params.CreateWalletRequest{
		UserID:   userID,
		Currency: "IDR",
	}

	mockRepo.On("Create", mock.Anything, mock.AnythingOfType("*entity.Wallet")).Return(errors.New("db error"))

	resp, err := uc.CreateWallet(context.Background(), req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to create wallet", err.Message)

	mockRepo.AssertExpectations(t)
}

func TestGetBalance_Success(t *testing.T) {
	mockRepo, _, uc := setupTest(t)

	userID := uuid.New()
	mockWallet := &entity.Wallet{
		ID:       uuid.New(),
		UserID:   userID,
		Balance:  10000.0,
		Currency: "IDR",
	}

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(mockWallet, nil)

	resp, err := uc.GetBalance(context.Background(), userID)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 10000.0, resp.Balance)
	assert.Equal(t, "IDR", resp.Currency)

	mockRepo.AssertExpectations(t)
}

func TestGetBalance_NotFound(t *testing.T) {
	mockRepo, _, uc := setupTest(t)

	userID := uuid.New()
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, gorm.ErrRecordNotFound)

	resp, err := uc.GetBalance(context.Background(), userID)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "wallet not found", err.Message)

	mockRepo.AssertExpectations(t)
}
func TestGetBalance_RepositoryError(t *testing.T) {
	mockRepo, _, uc := setupTest(t)

	userID := uuid.New()
	expectedErr := errors.New("database is down")

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, expectedErr)

	balance, customErr := uc.GetBalance(context.Background(), userID)

	assert.Nil(t, balance)
	assert.NotNil(t, customErr)
	assert.Equal(t, "failed to get wallet", customErr.Message)

	mockRepo.AssertExpectations(t)
}
