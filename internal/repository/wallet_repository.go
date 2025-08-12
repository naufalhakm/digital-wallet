package repository

import (
	"context"
	"errors"
	"fmt"
	"go-digital-wallet/internal/entity"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type WalletRepository interface {
	Create(ctx context.Context, wallet *entity.Wallet) error
	GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error)
	GetByUserIDForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*entity.Wallet, error)
	UpdateBalance(ctx context.Context, tx *gorm.DB, walletID uuid.UUID, newBalance float64, version int) error
	CreateTransaction(ctx context.Context, tx *gorm.DB, transaction *entity.Transaction) error
	UpdateTransactionStatus(ctx context.Context, tx *gorm.DB, transactionID uuid.UUID, transaction *entity.Transaction) error
	GetTransactionsByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*entity.Transaction, error)
	CountTransactionsByWalletID(ctx context.Context, walletID uuid.UUID) (int64, error)
	BeginTx(ctx context.Context) *gorm.DB
	WithTx(tx *gorm.DB) WalletRepository
}

type WalletRepositoryImpl struct {
	db     *gorm.DB
	logger *logrus.Logger
}

func NewWalletRepository(db *gorm.DB, logger *logrus.Logger) WalletRepository {
	return &WalletRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

func (r *WalletRepositoryImpl) Create(ctx context.Context, wallet *entity.Wallet) error {
	if err := r.db.WithContext(ctx).Create(wallet).Error; err != nil {
		r.logger.WithError(err).Error("Failed to create wallet in database")
		return fmt.Errorf("failed to create wallet: %w", err)
	}
	return nil
}

func (r *WalletRepositoryImpl) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error) {
	var wallet entity.Wallet

	err := r.db.WithContext(ctx).Where("user_id = ?", userID).First(&wallet).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		r.logger.WithError(err).WithField("user_id", userID).Error("Failed to get wallet by user ID")
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}

	return &wallet, nil
}

func (r *WalletRepositoryImpl) GetByUserIDForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*entity.Wallet, error) {
	var wallet entity.Wallet

	// Use the transaction if provided, otherwise use main db connection
	db := r.db
	if tx != nil {
		db = tx
	}

	err := db.WithContext(ctx).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("user_id = ?", userID).
		First(&wallet).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, gorm.ErrRecordNotFound
		}
		r.logger.WithError(err).WithField("user_id", userID).Error("Failed to get wallet by user ID for update")
		return nil, fmt.Errorf("failed to get wallet for update: %w", err)
	}

	return &wallet, nil
}

func (r *WalletRepositoryImpl) UpdateBalance(ctx context.Context, tx *gorm.DB, walletID uuid.UUID, newBalance float64, version int) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	// Update with optimistic locking
	result := db.WithContext(ctx).
		Model(&entity.Wallet{}).
		Where("id = ? AND version = ?", walletID, version-1).
		Updates(map[string]interface{}{
			"balance": newBalance,
			"version": version,
		})

	if result.Error != nil {
		r.logger.WithError(result.Error).WithField("wallet_id", walletID).Error("Failed to update wallet balance")
		return fmt.Errorf("failed to update wallet balance: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("optimistic lock error: wallet was modified by another transaction")
	}

	return nil
}

func (r *WalletRepositoryImpl) CreateTransaction(ctx context.Context, tx *gorm.DB, transaction *entity.Transaction) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	if err := db.WithContext(ctx).Create(transaction).Error; err != nil {
		r.logger.WithError(err).Error("Failed to create transaction in database")
		return fmt.Errorf("failed to create transaction: %w", err)
	}

	return nil
}

func (r *WalletRepositoryImpl) UpdateTransactionStatus(ctx context.Context, tx *gorm.DB, transactionID uuid.UUID, transaction *entity.Transaction) error {
	db := r.db
	if tx != nil {
		db = tx
	}

	if err := db.WithContext(ctx).
		Model(&entity.Transaction{}).
		Where("id = ?", transactionID).
		Update("status", transaction.Status).Error; err != nil {
		r.logger.WithError(err).WithField("transaction_id", transactionID).
			Error("Failed to update transaction status")
		return fmt.Errorf("failed to update transaction status: %w", err)
	}

	return nil
}

func (r *WalletRepositoryImpl) GetTransactionsByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*entity.Transaction, error) {
	var transactions []*entity.Transaction

	err := r.db.WithContext(ctx).
		Where("wallet_id = ?", walletID).
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&transactions).Error

	if err != nil {
		r.logger.WithError(err).WithField("wallet_id", walletID).Error("Failed to get transactions")
		return nil, fmt.Errorf("failed to get transactions: %w", err)
	}

	return transactions, nil
}

func (r *WalletRepositoryImpl) CountTransactionsByWalletID(ctx context.Context, walletID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&entity.Transaction{}).
		Where("wallet_id = ?", walletID).
		Count(&count).Error
	if err != nil {
		return 0, fmt.Errorf("failed to count transactions: %w", err)
	}
	return count, nil
}

func (r *WalletRepositoryImpl) BeginTx(ctx context.Context) *gorm.DB {
	return r.db.WithContext(ctx).Begin()
}

func (r *WalletRepositoryImpl) WithTx(tx *gorm.DB) WalletRepository {
	return &WalletRepositoryImpl{
		db:     tx,
		logger: r.logger,
	}
}
