package params

import "github.com/google/uuid"

type WithdrawRequest struct {
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Description string  `json:"description,omitempty" validate:"max=500"`
}

type DepositRequest struct {
	Amount      float64 `json:"amount" validate:"required,gt=0"`
	Description string  `json:"description,omitempty" validate:"max=500"`
}

type CreateWalletRequest struct {
	UserID   uuid.UUID `json:"user_id" `
	Currency string    `json:"currency"  validate:"required,len=3"`
}
