package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type TransactionType string

const (
	TransactionTypeWithdraw TransactionType = "withdraw"
	TransactionTypeDeposit  TransactionType = "deposit"
)

type TransactionStatus string

const (
	TransactionStatusPending   TransactionStatus = "pending"
	TransactionStatusCompleted TransactionStatus = "completed"
	TransactionStatusFailed    TransactionStatus = "failed"
)

type Transaction struct {
	ID          uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	WalletID    uuid.UUID         `gorm:"type:uuid;not null;index" json:"wallet_id"`
	Type        TransactionType   `gorm:"type:varchar(20);not null;check:type IN ('withdraw','deposit')" json:"type"`
	Amount      float64           `gorm:"type:decimal(15,2);not null;check:amount > 0" json:"amount"`
	Status      TransactionStatus `gorm:"type:varchar(20);not null;default:'pending';check:status IN ('pending','completed','failed')" json:"status"`
	Description string            `gorm:"type:text" json:"description"`
	CreatedAt   time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP;index:idx_transactions_created_at,sort:desc" json:"created_at"`
	UpdatedAt   time.Time         `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	Wallet Wallet `gorm:"foreignKey:WalletID;constraint:OnDelete:CASCADE" json:"wallet,omitempty"`
}

func (t *Transaction) BeforeCreate(tx *gorm.DB) error {
	if t.ID == uuid.Nil {
		t.ID = uuid.New()
	}
	return nil
}

func (Transaction) TableName() string {
	return "transactions"
}
