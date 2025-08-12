package repository

import (
	"context"

	"go-digital-wallet/internal/entity"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

type MockWalletRepository struct {
	mock.Mock
}

func (m *MockWalletRepository) Create(ctx context.Context, wallet *entity.Wallet) error {
	args := m.Called(ctx, wallet)
	return args.Error(0)
}

func (m *MockWalletRepository) GetByUserID(ctx context.Context, userID uuid.UUID) (*entity.Wallet, error) {
	args := m.Called(ctx, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Wallet), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWalletRepository) GetByUserIDForUpdate(ctx context.Context, tx *gorm.DB, userID uuid.UUID) (*entity.Wallet, error) {
	args := m.Called(ctx, tx, userID)
	if args.Get(0) != nil {
		return args.Get(0).(*entity.Wallet), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWalletRepository) UpdateBalance(ctx context.Context, tx *gorm.DB, walletID uuid.UUID, newBalance float64, version int) error {
	args := m.Called(ctx, tx, walletID, newBalance, version)
	return args.Error(0)
}

func (m *MockWalletRepository) CreateTransaction(ctx context.Context, tx *gorm.DB, transaction *entity.Transaction) error {
	args := m.Called(ctx, tx, transaction)
	return args.Error(0)
}

func (m *MockWalletRepository) UpdateTransactionStatus(ctx context.Context, tx *gorm.DB, transactionID uuid.UUID, transaction *entity.Transaction) error {
	args := m.Called(ctx, tx, transactionID, transaction)
	return args.Error(0)
}

func (m *MockWalletRepository) GetTransactionsByWalletID(ctx context.Context, walletID uuid.UUID, limit, offset int) ([]*entity.Transaction, error) {
	args := m.Called(ctx, walletID, limit, offset)
	if args.Get(0) != nil {
		return args.Get(0).([]*entity.Transaction), args.Error(1)
	}
	return nil, args.Error(1)
}

func (m *MockWalletRepository) CountTransactionsByWalletID(ctx context.Context, walletID uuid.UUID) (int64, error) {
	args := m.Called(ctx, walletID)
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockWalletRepository) BeginTx(ctx context.Context) *gorm.DB {
	args := m.Called(ctx)
	if args.Get(0) != nil {
		return args.Get(0).(*gorm.DB)
	}
	return nil
}

func (m *MockWalletRepository) WithTx(tx *gorm.DB) WalletRepository {
	args := m.Called(tx)
	if args.Get(0) != nil {
		return args.Get(0).(WalletRepository)
	}
	return nil
}
