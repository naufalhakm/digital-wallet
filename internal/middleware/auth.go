package middleware

import (
	"go-digital-wallet/internal/commons/response"
	"go-digital-wallet/pkg/token"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type AuthMiddleware struct {
	secretKey  string
	logger     *logrus.Logger
	jwtManager *token.TokenManager
}

func NewAuthMiddleware(secretKey string, logger *logrus.Logger, jwtManager *token.TokenManager) *AuthMiddleware {
	return &AuthMiddleware{
		secretKey:  secretKey,
		logger:     logger,
		jwtManager: jwtManager,
	}
}

func (m *AuthMiddleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			resp := response.UnauthorizedErrorWithAdditionalInfo(nil, "Authorization header is required")
			c.AbortWithStatusJSON(resp.StatusCode, resp)
			return
		}

		bearerToken := strings.Split(authHeader, "Bearer ")

		if len(bearerToken) != 2 {
			resp := response.UnauthorizedErrorWithAdditionalInfo(nil, "len token must be 2")
			c.AbortWithStatusJSON(resp.StatusCode, resp)
			return
		}

		payload, err := m.jwtManager.ValidateToken(bearerToken[1])
		if err != nil {
			resp := response.UnauthorizedErrorWithAdditionalInfo(err.Error())
			c.AbortWithStatusJSON(resp.StatusCode, resp)
			return
		}

		userID, err := uuid.Parse(payload.AuthId)
		if err != nil {
			resp := response.UnauthorizedErrorWithAdditionalInfo(nil, "Invalid user ID in token")
			c.AbortWithStatusJSON(resp.StatusCode, resp)
			return
		}

		c.Set("user_id", userID)
		c.Next()
	}
}
