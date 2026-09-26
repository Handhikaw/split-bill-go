package model

import "net/http"

type ApiError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	HttpStatus int    `json:"-"`
}

func (e *ApiError) Error() string {
	return e.Message
}

func NewBadRequest(msg string) *ApiError {
	return &ApiError{
		Code:       "BAD_REQUEST",
		Message:    msg,
		HttpStatus: http.StatusBadRequest,
	}
}

func NewNotFound(msg string) *ApiError {
	return &ApiError{
		Code:       "NOT_FOUND",
		Message:    msg,
		HttpStatus: http.StatusNotFound,
	}
}

func NewNotImplemented(msg string) *ApiError {
	return &ApiError{
		Code:       "NOT_IMPLEMENTED",
		Message:    msg,
		HttpStatus: http.StatusNotImplemented,
	}
}

func NewInternalError(msg string) *ApiError {
	return &ApiError{
		Code:       "INTERNAL_SERVER_ERROR",
		Message:    msg,
		HttpStatus: http.StatusInternalServerError,
	}
}
