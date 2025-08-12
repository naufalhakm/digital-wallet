package response

import (
	"net/http"
)

type CustomError struct {
	Code           string      `json:"code"`
	StatusCode     int         `json:"status_code"`
	Status         bool        `json:"status"`
	Message        string      `json:"message"`
	AdditionalInfo interface{} `json:"additional_info,omitempty"`
}

var (
	generalError = CustomError{
		Code:       "ERR0001",
		StatusCode: http.StatusInternalServerError,
		Status:     false,
		Message:    "INTERNAL SERVER ERROR",
	}
	repositoryError = CustomError{
		Code:       "ERR0002",
		StatusCode: http.StatusInternalServerError,
		Status:     false,
		Message:    "REPOSITORY ERROR",
	}
	notFoundError = CustomError{
		Code:       "ERR0003",
		StatusCode: http.StatusBadRequest,
		Status:     false,
		Message:    "NOT FOUND ERROR",
	}
	unauthorizedError = CustomError{
		Code:       "ERR0004",
		StatusCode: http.StatusUnauthorized,
		Status:     false,
		Message:    "UNAUTHORIZED",
	}
	badRequestError = CustomError{
		Code:       "ERR0005",
		StatusCode: http.StatusBadRequest,
		Status:     false,
		Message:    "BAD REQUEST ERROR",
	}
)

func GeneralError(message ...string) *CustomError {
	err := generalError
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}
func GeneralErrorWithAdditionalInfo(info interface{}, message ...string) *CustomError {
	err := generalError
	err.AdditionalInfo = info
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}

func RepositoryError(message ...string) *CustomError {
	err := repositoryError
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}

func RepositoryErrorWithAdditionalInfo(info interface{}, message ...string) *CustomError {
	err := repositoryError
	err.AdditionalInfo = info
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}

func NotFoundError(message ...string) *CustomError {
	err := notFoundError
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}

func NotFoundErrorWithAdditionalInfo(info interface{}, message ...string) *CustomError {
	err := repositoryError
	err.AdditionalInfo = info
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}
func UnauthorizedError(message ...string) *CustomError {
	err := unauthorizedError
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}

func UnauthorizedErrorWithAdditionalInfo(info interface{}, message ...string) *CustomError {
	err := unauthorizedError
	err.AdditionalInfo = info
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}

func BadRequestError(message ...string) *CustomError {
	err := badRequestError
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}

func BadRequestErrorWithAdditionalInfo(info interface{}, message ...string) *CustomError {
	err := badRequestError
	err.AdditionalInfo = info
	if len(message) != 0 {
		err.Message = message[0]
	}
	return &err
}
