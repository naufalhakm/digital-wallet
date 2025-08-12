package params

import "github.com/google/uuid"

type AuthResponse struct {
	Token string `json:"token"`
	User  struct {
		ID    uuid.UUID `json:"id"`
		Name  string    `json:"name"`
		Email string    `json:"email"`
	} `json:"user"`
}
