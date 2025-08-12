package usecase_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-digital-wallet/internal/entity"
	"go-digital-wallet/internal/params"
	"go-digital-wallet/internal/repository"
	"go-digital-wallet/internal/usecase"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTest(t *testing.T) (*repository.MockWalletRepository, *miniredis.Miniredis, *redis.Client, usecase.WalletUsecase, *gorm.DB) {
	mockRepo := new(repository.MockWalletRepository)

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})

	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to in-memory database: %v", err)
	}

	wu := usecase.NewWalletUsecase(mockRepo, logger, rdb)

	return mockRepo, mr, rdb, wu, db
}

func TestCreateWallet_Success(t *testing.T) {
	mockRepo, _, _, uc, _ := setupTest(t)

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
	mockRepo, _, _, uc, _ := setupTest(t)

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
	mockRepo, _, _, uc, _ := setupTest(t)

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
	mockRepo, _, _, uc, _ := setupTest(t)

	userID := uuid.New()
	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, gorm.ErrRecordNotFound)

	resp, err := uc.GetBalance(context.Background(), userID)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "wallet not found", err.Message)

	mockRepo.AssertExpectations(t)
}
func TestGetBalance_RepositoryError(t *testing.T) {
	mockRepo, _, _, uc, _ := setupTest(t)

	userID := uuid.New()
	expectedErr := errors.New("database is down")

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, expectedErr)

	balance, customErr := uc.GetBalance(context.Background(), userID)

	assert.Nil(t, balance)
	assert.NotNil(t, customErr)
	assert.Equal(t, "failed to get wallet", customErr.Message)

	mockRepo.AssertExpectations(t)
}

func TestWithdraw_Success(t *testing.T) {
	mockRepo, _, _, uc, db := setupTest(t)

	userID := uuid.New()
	walletID := uuid.New()
	withdrawAmount := 500.0
	initialBalance := 1000.0

	req := &params.WithdrawRequest{Amount: withdrawAmount, Description: "test withdraw"}

	mockWallet := &entity.Wallet{
		ID:       walletID,
		UserID:   userID,
		Balance:  initialBalance,
		Currency: "IDR",
		Version:  1,
	}

	realTx := db.Begin()

	defer realTx.Rollback()

	mockRepo.On("BeginTx", mock.Anything).Return(realTx)

	mockRepo.On("WithTx", realTx).Return(mockRepo)
	mockRepo.On("GetByUserIDForUpdate", mock.Anything, realTx, userID).Return(mockWallet, nil)
	mockRepo.On("CreateTransaction", mock.Anything, realTx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	mockRepo.On("UpdateBalance", mock.Anything, realTx, walletID, initialBalance-withdrawAmount, mockWallet.Version+1).Return(nil)
	mockRepo.On("UpdateTransactionStatus", mock.Anything, realTx, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("*entity.Transaction")).Return(nil)

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, initialBalance-withdrawAmount, resp.NewBalance)
	assert.Equal(t, entity.TransactionStatusCompleted, resp.Status)

	mockRepo.AssertExpectations(t)
}

func TestWithdraw_InvalidAmount(t *testing.T) {
	_, _, _, uc, _ := setupTest(t)
	userID := uuid.New()
	req := &params.WithdrawRequest{Amount: -100}

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "invalid amount", err.Message)
}

func TestWithdraw_BeginTxFails(t *testing.T) {
	mockRepo, _, _, uc, _ := setupTest(t)
	userID := uuid.New()
	req := &params.WithdrawRequest{Amount: 100.0}

	mockTxWithError := &gorm.DB{Error: errors.New("failed to connect")}

	mockRepo.On("BeginTx", mock.Anything).Return(mockTxWithError)

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to begin transaction", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestWithdraw_InsufficientBalance(t *testing.T) {
	mockRepo, _, _, uc, db := setupTest(t)
	userID := uuid.New()
	req := &params.WithdrawRequest{Amount: 1500.0}
	mockWallet := &entity.Wallet{Balance: 1000.0}
	realTx := db.Begin()
	defer realTx.Rollback()

	mockRepo.On("BeginTx", mock.Anything).Return(realTx)
	mockRepo.On("WithTx", realTx).Return(mockRepo)
	mockRepo.On("GetByUserIDForUpdate", mock.Anything, realTx, userID).Return(mockWallet, nil)

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "insufficient balance", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestWithdraw_WalletNotFound(t *testing.T) {
	mockRepo, _, _, uc, db := setupTest(t)
	userID := uuid.New()
	req := &params.WithdrawRequest{Amount: 100.0}
	realTx := db.Begin()
	defer realTx.Rollback()

	mockRepo.On("BeginTx", mock.Anything).Return(realTx)
	mockRepo.On("WithTx", realTx).Return(mockRepo)
	mockRepo.On("GetByUserIDForUpdate", mock.Anything, realTx, userID).Return(nil, gorm.ErrRecordNotFound)

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "wallet not found", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestWithdraw_GetForUpdateFails_GenericError(t *testing.T) {
	mockRepo, _, _, uc, db := setupTest(t)
	userID := uuid.New()
	req := &params.WithdrawRequest{Amount: 100.0}
	realTx := db.Begin()
	defer realTx.Rollback()

	mockRepo.On("BeginTx", mock.Anything).Return(realTx)
	mockRepo.On("WithTx", realTx).Return(mockRepo)
	mockRepo.On("GetByUserIDForUpdate", mock.Anything, realTx, userID).Return(nil, errors.New("unexpected db error"))

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to get wallet for update", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestWithdraw_CreateTransactionFails(t *testing.T) {
	mockRepo, _, _, uc, db := setupTest(t)
	userID, walletID := uuid.New(), uuid.New()
	req := &params.WithdrawRequest{Amount: 500.0}
	mockWallet := &entity.Wallet{ID: walletID, UserID: userID, Balance: 1000.0, Version: 1}
	realTx := db.Begin()
	defer realTx.Rollback()

	mockRepo.On("BeginTx", mock.Anything).Return(realTx)
	mockRepo.On("WithTx", realTx).Return(mockRepo)
	mockRepo.On("GetByUserIDForUpdate", mock.Anything, realTx, userID).Return(mockWallet, nil)
	mockRepo.On("CreateTransaction", mock.Anything, realTx, mock.AnythingOfType("*entity.Transaction")).Return(errors.New("db write error"))

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to create transaction", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestWithdraw_UpdateBalanceFails(t *testing.T) {
	mockRepo, _, _, uc, db := setupTest(t)
	userID, walletID := uuid.New(), uuid.New()
	req := &params.WithdrawRequest{Amount: 500.0}
	mockWallet := &entity.Wallet{ID: walletID, UserID: userID, Balance: 1000.0, Version: 1}
	realTx := db.Begin()
	defer realTx.Rollback()

	mockRepo.On("BeginTx", mock.Anything).Return(realTx)
	mockRepo.On("WithTx", realTx).Return(mockRepo)
	mockRepo.On("GetByUserIDForUpdate", mock.Anything, realTx, userID).Return(mockWallet, nil)
	mockRepo.On("CreateTransaction", mock.Anything, realTx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	mockRepo.On("UpdateBalance", mock.Anything, realTx, walletID, 500.0, 2).Return(errors.New("db conflict"))

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to update wallet balance", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestWithdraw_UpdateTransactionStatusFails(t *testing.T) {
	mockRepo, _, _, uc, db := setupTest(t)
	userID, walletID := uuid.New(), uuid.New()
	req := &params.WithdrawRequest{Amount: 500.0}
	mockWallet := &entity.Wallet{ID: walletID, UserID: userID, Balance: 1000.0, Version: 1}
	realTx := db.Begin()
	defer realTx.Rollback()

	mockRepo.On("BeginTx", mock.Anything).Return(realTx)
	mockRepo.On("WithTx", realTx).Return(mockRepo)
	mockRepo.On("GetByUserIDForUpdate", mock.Anything, realTx, userID).Return(mockWallet, nil)
	mockRepo.On("CreateTransaction", mock.Anything, realTx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	mockRepo.On("UpdateBalance", mock.Anything, realTx, walletID, 500.0, 2).Return(nil)
	mockRepo.On("UpdateTransactionStatus", mock.Anything, realTx, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("*entity.Transaction")).Return(errors.New("db status update error"))

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to update transaction status", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestWithdraw_CommitFails(t *testing.T) {
	mockRepo, _, _, uc, db := setupTest(t)
	userID, walletID := uuid.New(), uuid.New()
	req := &params.WithdrawRequest{Amount: 500.0}
	mockWallet := &entity.Wallet{ID: walletID, UserID: userID, Balance: 1000.0, Version: 1}
	realTx := db.Begin()

	mockRepo.On("BeginTx", mock.Anything).Return(realTx)
	mockRepo.On("WithTx", realTx).Return(mockRepo)
	mockRepo.On("GetByUserIDForUpdate", mock.Anything, realTx, userID).Return(mockWallet, nil)
	mockRepo.On("CreateTransaction", mock.Anything, realTx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	mockRepo.On("UpdateBalance", mock.Anything, realTx, walletID, 500.0, 2).Return(nil)
	mockRepo.On("UpdateTransactionStatus", mock.Anything, realTx, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("*entity.Transaction")).Return(nil)

	realTx.Rollback()

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to commit transaction", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestWithdraw_CacheInvalidationFails(t *testing.T) {
	mockRepo, mr, _, uc, db := setupTest(t)
	userID, walletID := uuid.New(), uuid.New()
	req := &params.WithdrawRequest{Amount: 500.0}
	mockWallet := &entity.Wallet{ID: walletID, UserID: userID, Balance: 1000.0, Version: 1}
	realTx := db.Begin()
	defer realTx.Rollback()

	mockRepo.On("BeginTx", mock.Anything).Return(realTx)
	mockRepo.On("WithTx", realTx).Return(mockRepo)
	mockRepo.On("GetByUserIDForUpdate", mock.Anything, realTx, userID).Return(mockWallet, nil)
	mockRepo.On("CreateTransaction", mock.Anything, realTx, mock.AnythingOfType("*entity.Transaction")).Return(nil)
	mockRepo.On("UpdateBalance", mock.Anything, realTx, walletID, 500.0, 2).Return(nil)
	mockRepo.On("UpdateTransactionStatus", mock.Anything, realTx, mock.AnythingOfType("uuid.UUID"), mock.AnythingOfType("*entity.Transaction")).Return(nil)

	mr.SetError("redis is down")

	resp, err := uc.Withdraw(context.Background(), userID, req)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 500.0, resp.NewBalance)
	mockRepo.AssertExpectations(t)
}

func TestGetTransactionHistory_CacheHit(t *testing.T) {
	mockRepo, _, rdb, uc, _ := setupTest(t)
	userID := uuid.New()
	limit, offset, page := 10, 0, 1
	cacheKey := fmt.Sprintf("transactions:%s:%d:%d", userID.String(), page, limit)

	expectedResp := &params.TransactionHistoryResponse{Total: 1, Page: page}
	cachedData, _ := json.Marshal(expectedResp)
	rdb.Set(context.Background(), cacheKey, cachedData, time.Minute)

	resp, err := uc.GetTransactionHistory(context.Background(), userID, limit, offset)

	assert.Nil(t, err)
	assert.Equal(t, expectedResp.Total, resp.Total)
	mockRepo.AssertNotCalled(t, "GetByUserID")
}

func TestGetTransactionHistory_CacheMiss_Success(t *testing.T) {
	mockRepo, _, rdb, uc, _ := setupTest(t)
	userID, walletID := uuid.New(), uuid.New()
	limit, offset, page := 10, 0, 1
	cacheKey := fmt.Sprintf("transactions:%s:%d:%d", userID.String(), page, limit)

	mockWallet := &entity.Wallet{ID: walletID}
	mockTransactions := []*entity.Transaction{{ID: uuid.New(), Amount: 100}}
	var totalCount int64 = 1

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(mockWallet, nil)
	mockRepo.On("GetTransactionsByWalletID", mock.Anything, walletID, limit, offset).Return(mockTransactions, nil)
	mockRepo.On("CountTransactionsByWalletID", mock.Anything, walletID).Return(totalCount, nil)

	resp, err := uc.GetTransactionHistory(context.Background(), userID, limit, offset)

	assert.Nil(t, err)
	assert.NotNil(t, resp)
	assert.Len(t, resp.Transactions, 1)
	assert.Equal(t, totalCount, resp.Total)
	mockRepo.AssertExpectations(t)

	cachedVal, cacheErr := rdb.Get(context.Background(), cacheKey).Result()
	assert.NoError(t, cacheErr)
	assert.NotEmpty(t, cachedVal)
}

func TestGetTransactionHistory_WalletNotFound(t *testing.T) {
	mockRepo, _, _, uc, _ := setupTest(t)
	userID := uuid.New()
	limit, offset := 10, 0

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, gorm.ErrRecordNotFound)

	resp, err := uc.GetTransactionHistory(context.Background(), userID, limit, offset)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "wallet not found", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestGetTransactionHistory_GetForUpdateFails_GenericError(t *testing.T) {
	mockRepo, _, _, uc, _ := setupTest(t)
	userID := uuid.New()
	limit, offset := 10, 0

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(nil, errors.New("unexpected db error"))

	resp, err := uc.GetTransactionHistory(context.Background(), userID, limit, offset)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to get wallet", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestGetTransactionHistory_GetTransactionsFails(t *testing.T) {
	mockRepo, _, _, uc, _ := setupTest(t)
	userID, walletID := uuid.New(), uuid.New()
	limit, offset := 10, 0
	mockWallet := &entity.Wallet{ID: walletID}

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(mockWallet, nil)
	mockRepo.On("GetTransactionsByWalletID", mock.Anything, walletID, limit, offset).Return(nil, errors.New("db error"))

	resp, err := uc.GetTransactionHistory(context.Background(), userID, limit, offset)

	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to get transaction history", err.Message)
	mockRepo.AssertExpectations(t)
}

func TestGetTransactionHistory_CountFails(t *testing.T) {
	mockRepo, mr, _, uc, _ := setupTest(t)
	userID, walletID := uuid.New(), uuid.New()
	limit, offset := 10, 0
	mockWallet := &entity.Wallet{ID: walletID}

	mr.SetError("cache miss")

	mockRepo.On("GetByUserID", mock.Anything, userID).Return(mockWallet, nil)
	mockRepo.On("GetTransactionsByWalletID", mock.Anything, walletID, limit, offset).Return([]*entity.Transaction{}, nil)
	mockRepo.On("CountTransactionsByWalletID", mock.Anything, walletID).Return(int64(0), errors.New("db count error"))

	resp, err := uc.GetTransactionHistory(context.Background(), userID, limit, offset)
	assert.Nil(t, resp)
	assert.NotNil(t, err)
	assert.Equal(t, "failed to get total transactions", err.Message)
	mockRepo.AssertExpectations(t)
}
