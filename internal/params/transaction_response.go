package params

import (
	"go-digital-wallet/internal/entity"
	"time"

	"github.com/google/uuid"
)

type TransactionResponse struct {
	ID          uuid.UUID                `json:"id"`
	Type        entity.TransactionType   `json:"type"`
	Amount      float64                  `json:"amount"`
	Description *string                  `json:"description,omitempty"`
	Status      entity.TransactionStatus `json:"status"`
	CreatedAt   time.Time                `json:"created_at"`
	UpdatedAt   time.Time                `json:"updated_at"`
}

type TransactionHistoryResponse struct {
	Transactions []*TransactionResponse `json:"transactions"`
	Total        int64                  `json:"total"`
	Page         int                    `json:"page"`
	Limit        int                    `json:"limit"`
	TotalPages   int                    `json:"total_pages"`
}
