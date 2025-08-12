package handler

import (
	"go-digital-wallet/internal/commons/response"
	"go-digital-wallet/internal/params"
	"go-digital-wallet/internal/usecase"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/sirupsen/logrus"
)

type AuthHandler interface {
	Register(c *gin.Context)
	Login(c *gin.Context)
}

type AuthHandlerImpl struct {
	authService usecase.AuthUsecase
	logger      *logrus.Logger
	validator   *validator.Validate
}

func NewAuthHandler(authService usecase.AuthUsecase, logger *logrus.Logger, validator *validator.Validate) AuthHandler {
	return &AuthHandlerImpl{
		authService: authService,
		logger:      logger,
		validator:   validator,
	}
}

func (h *AuthHandlerImpl) Register(c *gin.Context) {
	var req params.RegisterRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse register request")
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid JSON format",
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

	authResponse, custErr := h.authService.Register(&req)
	if custErr != nil {
		c.AbortWithStatusJSON(custErr.StatusCode, custErr)
		return
	}

	resp := response.CreatedSuccessWithPayload(authResponse)
	c.JSON(resp.StatusCode, resp)
}

func (h *AuthHandlerImpl) Login(c *gin.Context) {
	var req params.LoginRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.WithError(err).Error("Failed to parse login request")
		c.JSON(http.StatusBadRequest, gin.H{
			"status":  false,
			"message": "Invalid JSON format",
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

	authResponse, custErr := h.authService.Login(&req)
	if custErr != nil {
		c.AbortWithStatusJSON(custErr.StatusCode, custErr)
		return
	}

	resp := response.GeneralSuccessCustomMessageAndPayload("Success login user", authResponse)
	c.JSON(http.StatusOK, resp)
}

func getValidationErrorMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "This field is required"
	case "max":
		return "This field exceeds maximum length of " + err.Param()
	case "min":
		return "This field must be at least " + err.Param() + " characters"
	case "email":
		return "This field must be a valid email"
	case "oneof":
		return "This field must be one of: " + err.Param()
	default:
		return "This field is invalid"
	}
}
