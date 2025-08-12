package entity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Wallet struct {
	ID        uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;uniqueIndex" json:"user_id"`
	Balance   float64   `gorm:"type:decimal(15,2);not null;default:0.00;check:balance >= 0" json:"balance"`
	Currency  string    `gorm:"type:varchar(3);not null;default:'IDR'" json:"currency"`
	Version   int       `gorm:"not null;default:1" json:"version"`
	CreatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null;default:CURRENT_TIMESTAMP" json:"updated_at"`

	// Relations
	Transactions []Transaction `gorm:"foreignKey:WalletID;constraint:OnDelete:CASCADE" json:"transactions,omitempty"`
}

func (w *Wallet) BeforeCreate(tx *gorm.DB) error {
	if w.ID == uuid.Nil {
		w.ID = uuid.New()
	}
	return nil
}

func (Wallet) TableName() string {
	return "wallets"
}
