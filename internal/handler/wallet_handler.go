package handler

import (
	"go-digital-wallet/internal/commons/response"
	"go-digital-wallet/internal/params"
	"go-digital-wallet/internal/usecase"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type WalletHandler interface {
	CreateWallet(c *gin.Context)
	GetBalance(c *gin.Context)
	Withdraw(c *gin.Context)
	Deposit(c *gin.Context)
	GetTransactionHistory(c *gin.Context)
}

type WalletHandlerImpl struct {
	usecase   usecase.WalletUsecase
	logger    *logrus.Logger
	validator *validator.Validate
}

func NewWalletHandler(usecase usecase.WalletUsecase, logger *logrus.Logger, validator *validator.Validate) WalletHandler {
	return &WalletHandlerImpl{
		usecase:   usecase,
		logger:    logger,
		validator: validator,
	}
}
func (h *WalletHandlerImpl) getUserIDFromContext(c *gin.Context) (uuid.UUID, bool) {
	userIDVal, exists := c.Get("user_id")
	if !exists {
		h.logger.Error("user_id not found in context")
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  false,
			"message": "Unauthorized",
		})
		return uuid.Nil, false
	}

	userID, ok := userIDVal.(uuid.UUID)
	if !ok {
		h.logger.Error("user_id in context is not uuid.UUID")
		c.JSON(http.StatusUnauthorized, gin.H{
			"status":  false,
			"message": "Unauthorized",
		})
		return uuid.Nil, false
	}

	return userID, true
}

func (h *WalletHandlerImpl) CreateWallet(c *gin.Context) {
	userID, ok := h.getUserIDFromContext(c)
	if !ok {
		return
	}

	var req params.CreateWalletRequest
	req.UserID = userID
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request payload")
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid request payload",
		})
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		details := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			details[err.Field()] = getValidationErrorMessage(err)
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Validation failed",
			"errors":  details,
		})
		return
	}

	walletResp, err := h.usecase.CreateWallet(c.Request.Context(), &req)
	if err != nil {
		c.AbortWithStatusJSON(err.StatusCode, err)
		return
	}
	resp := response.CreatedSuccessWithPayload(walletResp)
	c.JSON(resp.StatusCode, resp)
}

func (h *WalletHandlerImpl) GetBalance(c *gin.Context) {
	userID, ok := h.getUserIDFromContext(c)
	if !ok {
		return
	}

	balanceResp, custErr := h.usecase.GetBalance(c.Request.Context(), userID)
	if custErr != nil {
		c.AbortWithStatusJSON(custErr.StatusCode, custErr)
		return
	}

	resp := response.GeneralSuccessCustomMessageAndPayload("Balance retrieved successfully", balanceResp)
	c.JSON(resp.StatusCode, resp)
}

func (h *WalletHandlerImpl) Withdraw(c *gin.Context) {
	userID, ok := h.getUserIDFromContext(c)
	if !ok {
		return
	}

	var req params.WithdrawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request payload")
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid request payload",
		})
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		details := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			details[err.Field()] = getValidationErrorMessage(err)
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Validation failed",
			"errors":  details,
		})
		return
	}

	withdrawResp, custErr := h.usecase.Withdraw(c.Request.Context(), userID, &req)
	if custErr != nil {
		c.JSON(custErr.StatusCode, custErr)
		return
	}

	resp := response.GeneralSuccessCustomMessageAndPayload("Withdrawal completed successfully", withdrawResp)
	c.JSON(resp.StatusCode, resp)
}

func (h *WalletHandlerImpl) Deposit(c *gin.Context) {
	userID, ok := h.getUserIDFromContext(c)
	if !ok {
		return
	}

	var req params.DepositRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Invalid request payload for deposit")
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid request payload",
		})
		return
	}

	if err := h.validator.Struct(&req); err != nil {
		details := make(map[string]string)
		for _, err := range err.(validator.ValidationErrors) {
			details[err.Field()] = "Validation error on this field"
		}

		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Validation failed",
			"errors":  details,
		})
		return
	}

	depositResp, custErr := h.usecase.Deposit(c.Request.Context(), userID, &req)
	if custErr != nil {
		c.JSON(custErr.StatusCode, custErr)
		return
	}

	resp := response.GeneralSuccessCustomMessageAndPayload("Deposit completed successfully", depositResp)
	c.JSON(resp.StatusCode, resp)
}

func (h *WalletHandlerImpl) GetTransactionHistory(c *gin.Context) {
	userID, ok := h.getUserIDFromContext(c)
	if !ok {
		return
	}

	// Parse pagination parameters
	limitStr := c.DefaultQuery("limit", "10")
	offsetStr := c.DefaultQuery("offset", "0")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	transactions, custErr := h.usecase.GetTransactionHistory(c.Request.Context(), userID, limit, offset)
	if custErr != nil {
		c.AbortWithStatusJSON(custErr.StatusCode, custErr)
		return
	}

	resp := response.GeneralSuccessCustomMessageAndPayload("Transaction history retrieved successfully", transactions)
	c.JSON(resp.StatusCode, resp)
}
