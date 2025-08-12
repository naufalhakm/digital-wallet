package token

import "time"

type Token struct {
	AuthId  string
	Expired time.Time
	Role    string
}
