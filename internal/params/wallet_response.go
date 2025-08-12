package params

import (
	"go-digital-wallet/internal/entity"
	"time"

	"github.com/google/uuid"
)

type BalanceResponse struct {
	UserID    uuid.UUID `json:"user_id"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	Timestamp time.Time `json:"timestamp"`
}

type WithdrawResponse struct {
	TransactionID uuid.UUID                `json:"transaction_id"`
	Amount        float64                  `json:"amount"`
	NewBalance    float64                  `json:"new_balance"`
	Status        entity.TransactionStatus `json:"status"`
	Timestamp     time.Time                `json:"timestamp"`
}

type DepositResponse struct {
	TransactionID uuid.UUID                `json:"transaction_id"`
	Amount        float64                  `json:"amount"`
	NewBalance    float64                  `json:"new_balance"`
	Status        entity.TransactionStatus `json:"status"`
	Timestamp     time.Time                `json:"timestamp"`
}

type WalletResponse struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	Balance   float64   `json:"balance"`
	Currency  string    `json:"currency"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
