package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type TokenManager struct {
	secret string
	expiry time.Duration
}

func NewTokenManager(secret string, expiryHours int) *TokenManager {
	return &TokenManager{
		secret: secret,
		expiry: time.Duration(expiryHours) * time.Hour,
	}
}

func (tm *TokenManager) GenerateToken(userID uuid.UUID) (string, error) {
	payload := Token{
		AuthId:  userID.String(),
		Expired: time.Now().Add(tm.expiry),
	}
	claims := jwt.MapClaims{
		"payload": payload,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(tm.secret))
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}

func (tm *TokenManager) ValidateToken(tokenString string) (*Token, error) {
	token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(tm.secret), nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		payloadInterface := claims["payload"]

		payloadToken := Token{}
		payloadByte, err := json.Marshal(payloadInterface)
		if err != nil {
			return nil, err
		}
		err = json.Unmarshal(payloadByte, &payloadToken)
		if err != nil {
			return nil, err
		}
		if time.Now().After(payloadToken.Expired) {
			return nil, errors.New("token expired")
		}
		return &payloadToken, nil
	}
	return nil, errors.New("unauthorized")
}
