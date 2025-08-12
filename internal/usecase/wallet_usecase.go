package usecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"go-digital-wallet/internal/commons/response"
	"go-digital-wallet/internal/entity"
	"go-digital-wallet/internal/params"
	"go-digital-wallet/internal/repository"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

type WalletUsecase interface {
	CreateWallet(ctx context.Context, req *params.CreateWalletRequest) (*params.WalletResponse, *response.CustomError)
	GetBalance(ctx context.Context, userID uuid.UUID) (*params.BalanceResponse, *response.CustomError)
	Withdraw(ctx context.Context, userID uuid.UUID, req *params.WithdrawRequest) (*params.WithdrawResponse, *response.CustomError)
	Deposit(ctx context.Context, userID uuid.UUID, req *params.DepositRequest) (*params.DepositResponse, *response.CustomError)
	GetTransactionHistory(ctx context.Context, userID uuid.UUID, limit, offset int) (*params.TransactionHistoryResponse, *response.CustomError)
}

type WalletUsecaseImpl struct {
	repo   repository.WalletRepository
	logger *logrus.Logger
	mutex  sync.RWMutex
	cache  *redis.Client
}

func NewWalletUsecase(repo repository.WalletRepository, logger *logrus.Logger, cache *redis.Client) WalletUsecase {
	return &WalletUsecaseImpl{
		repo:   repo,
		logger: logger,
		cache:  cache,
	}
}

func (u *WalletUsecaseImpl) CreateWallet(ctx context.Context, req *params.CreateWalletRequest) (*params.WalletResponse, *response.CustomError) {
	wallet := &entity.Wallet{
		UserID:   req.UserID,
		Balance:  0.0,
		Currency: req.Currency,
		Version:  1,
	}

	if err := u.repo.Create(ctx, wallet); err != nil {
		u.logger.WithError(err).Error("Failed to create wallet")
		return nil, response.RepositoryError("failed to create wallet")
	}

	return &params.WalletResponse{
		ID:        wallet.ID,
		UserID:    wallet.UserID,
		Balance:   wallet.Balance,
		Currency:  wallet.Currency,
		CreatedAt: wallet.CreatedAt,
		UpdatedAt: wallet.UpdatedAt,
	}, nil
}

func (u *WalletUsecaseImpl) GetBalance(ctx context.Context, userID uuid.UUID) (*params.BalanceResponse, *response.CustomError) {
	u.mutex.RLock()
	defer u.mutex.RUnlock()

	wallet, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFoundError("wallet not found")
		}
		u.logger.WithError(err).WithField("user_id", userID).Error("Failed to get wallet")
		return nil, response.RepositoryError("failed to get wallet")
	}

	return &params.BalanceResponse{
		UserID:    wallet.UserID,
		Balance:   wallet.Balance,
		Currency:  wallet.Currency,
		Timestamp: time.Now(),
	}, nil
}

func (u *WalletUsecaseImpl) Withdraw(ctx context.Context, userID uuid.UUID, req *params.WithdrawRequest) (*params.WithdrawResponse, *response.CustomError) {
	if req.Amount <= 0 {
		return nil, response.BadRequestError("invalid amount")
	}

	tx := u.repo.BeginTx(ctx)
	if tx.Error != nil {
		u.logger.WithError(tx.Error).Error("Failed to begin transaction")
		return nil, response.GeneralError("failed to begin transaction")
	}

	txRepo := u.repo.WithTx(tx)

	var transaction *entity.Transaction

	defer tx.Rollback()

	wallet, err := txRepo.GetByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFoundError("wallet not found")
		}
		u.logger.WithError(err).WithField("user_id", userID).Error("Failed to get wallet for update")
		return nil, response.RepositoryError("failed to get wallet for update")
	}

	if wallet.Balance < req.Amount {
		u.logger.WithFields(logrus.Fields{
			"user_id":         userID,
			"current_balance": wallet.Balance,
			"withdraw_amount": req.Amount,
		}).Warn("Insufficient balance for withdrawal")
		return nil, response.BadRequestError("insufficient balance")
	}

	newBalance := wallet.Balance - req.Amount
	newVersion := wallet.Version + 1

	transaction = &entity.Transaction{
		ID:          uuid.New(),
		WalletID:    wallet.ID,
		Type:        entity.TransactionTypeWithdraw,
		Amount:      req.Amount,
		Status:      entity.TransactionStatusPending,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := txRepo.CreateTransaction(ctx, tx, transaction); err != nil {
		u.logger.WithError(err).Error("Failed to create transaction")
		return nil, response.RepositoryError("failed to create transaction")
	}

	if err := txRepo.UpdateBalance(ctx, tx, wallet.ID, newBalance, newVersion); err != nil {
		u.logger.WithError(err).Error("Failed to update wallet balance")
		return nil, response.RepositoryError("failed to update wallet balance")
	}

	transaction.Status = entity.TransactionStatusCompleted

	if err := txRepo.UpdateTransactionStatus(ctx, tx, transaction.ID, transaction); err != nil {
		u.logger.WithError(err).Error("Failed to update transaction status")
		return nil, response.RepositoryError("failed to update transaction status")
	}

	if err := tx.Commit().Error; err != nil {
		u.logger.WithError(err).Error("Failed to commit transaction")
		return nil, response.RepositoryError("failed to commit transaction")
	}

	cachePattern := fmt.Sprintf("transactions:%s:*", userID.String())
	if keys, err := u.cache.Keys(ctx, cachePattern).Result(); err == nil {
		if len(keys) > 0 {
			if err := u.cache.Del(ctx, keys...).Err(); err != nil {
				u.logger.WithError(err).Warn("Failed to invalidate transaction cache")
			} else {
				u.logger.WithField("cache_keys", keys).Info("Invalidated transaction cache after withdrawal")
			}
		}
	} else {
		u.logger.WithError(err).Warn("Failed to fetch transaction cache keys for invalidation")
	}

	u.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"transaction_id": transaction.ID,
		"amount":         req.Amount,
		"new_balance":    newBalance,
	}).Info("Withdrawal completed successfully")

	return &params.WithdrawResponse{
		TransactionID: transaction.ID,
		Amount:        req.Amount,
		NewBalance:    newBalance,
		Status:        transaction.Status,
		Timestamp:     transaction.UpdatedAt,
	}, nil
}

func (u *WalletUsecaseImpl) Deposit(ctx context.Context, userID uuid.UUID, req *params.DepositRequest) (*params.DepositResponse, *response.CustomError) {
	if req.Amount <= 0 {
		return nil, response.BadRequestError("invalid deposit amount")
	}

	tx := u.repo.BeginTx(ctx)
	if tx.Error != nil {
		u.logger.WithError(tx.Error).Error("Failed to begin transaction")
		return nil, response.GeneralError("failed to begin transaction")
	}
	txRepo := u.repo.WithTx(tx)
	defer tx.Rollback()

	wallet, err := txRepo.GetByUserIDForUpdate(ctx, tx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFoundError("wallet not found")
		}
		u.logger.WithError(err).WithField("user_id", userID).Error("Failed to get wallet for update")
		return nil, response.RepositoryError("failed to get wallet for update")
	}

	newBalance := wallet.Balance + req.Amount
	newVersion := wallet.Version + 1

	transaction := &entity.Transaction{
		ID:          uuid.New(),
		WalletID:    wallet.ID,
		Type:        entity.TransactionTypeDeposit,
		Amount:      req.Amount,
		Status:      entity.TransactionStatusPending,
		Description: req.Description,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := txRepo.CreateTransaction(ctx, tx, transaction); err != nil {
		u.logger.WithError(err).Error("Failed to create transaction")
		return nil, response.RepositoryError("failed to create transaction")
	}

	if err := txRepo.UpdateBalance(ctx, tx, wallet.ID, newBalance, newVersion); err != nil {
		u.logger.WithError(err).Error("Failed to update wallet balance")
		return nil, response.RepositoryError("failed to update wallet balance")
	}

	transaction.Status = entity.TransactionStatusCompleted
	if err := txRepo.UpdateTransactionStatus(ctx, tx, transaction.ID, transaction); err != nil {
		u.logger.WithError(err).Error("Failed to update transaction status")
		return nil, response.RepositoryError("failed to update transaction status")
	}

	if err := tx.Commit().Error; err != nil {
		u.logger.WithError(err).Error("Failed to commit transaction")
		return nil, response.RepositoryError("failed to commit transaction")
	}

	cachePattern := fmt.Sprintf("transactions:%s:*", userID.String())
	if keys, err := u.cache.Keys(ctx, cachePattern).Result(); err == nil {
		if len(keys) > 0 {
			if err := u.cache.Del(ctx, keys...).Err(); err != nil {
				u.logger.WithError(err).Warn("Failed to invalidate transaction cache")
			}
		}
	}

	u.logger.WithFields(logrus.Fields{
		"user_id":        userID,
		"transaction_id": transaction.ID,
		"amount":         req.Amount,
		"new_balance":    newBalance,
	}).Info("Deposit completed successfully")

	return &params.DepositResponse{
		TransactionID: transaction.ID,
		Amount:        req.Amount,
		NewBalance:    newBalance,
		Status:        transaction.Status,
		Timestamp:     transaction.UpdatedAt,
	}, nil
}

func (u *WalletUsecaseImpl) GetTransactionHistory(ctx context.Context, userID uuid.UUID, limit, offset int) (*params.TransactionHistoryResponse, *response.CustomError) {
	page := (offset / limit) + 1
	cacheKey := fmt.Sprintf("transactions:%s:%d:%d", userID, page, limit)

	if val, err := u.cache.Get(ctx, cacheKey).Result(); err == nil {
		var cached params.TransactionHistoryResponse
		if json.Unmarshal([]byte(val), &cached) == nil {
			u.logger.WithField("cache_key", cacheKey).Info("Cache hit for transaction history")
			return &cached, nil
		}
	}

	wallet, err := u.repo.GetByUserID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, response.NotFoundError("wallet not found")
		}
		return nil, response.RepositoryError("failed to get wallet")
	}

	transactions, err := u.repo.GetTransactionsByWalletID(ctx, wallet.ID, limit, offset)
	if err != nil {
		u.logger.WithError(err).Error("Failed to get transaction history")
		return nil, response.RepositoryError("failed to get transaction history")
	}

	total, err := u.repo.CountTransactionsByWalletID(ctx, wallet.ID)
	if err != nil {
		u.logger.WithError(err).Error("Failed to get total transactions")
		return nil, response.RepositoryError("failed to get total transactions")
	}

	transactionResponses := make([]*params.TransactionResponse, len(transactions))
	for i, t := range transactions {
		transactionResponses[i] = &params.TransactionResponse{
			ID:          t.ID,
			Type:        t.Type,
			Amount:      t.Amount,
			Description: &t.Description,
			Status:      t.Status,
			CreatedAt:   t.CreatedAt,
			UpdatedAt:   t.UpdatedAt,
		}
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	resp := &params.TransactionHistoryResponse{
		Transactions: transactionResponses,
		Total:        total,
		Page:         page,
		Limit:        limit,
		TotalPages:   totalPages,
	}

	if data, err := json.Marshal(resp); err == nil {
		if err := u.cache.Set(ctx, cacheKey, data, 5*time.Minute).Err(); err != nil {
			u.logger.WithError(err).Warn("Failed to cache transaction history")
		}
	}

	return resp, nil
}
