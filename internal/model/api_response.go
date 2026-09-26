package model

import "net/http"

type ApiResponse struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func Success(data any) ApiResponse {
	return ApiResponse{
		Code:    http.StatusText(http.StatusOK),
		Message: "success",
		Data:    data,
	}
}

func Created(data any) ApiResponse {
	return ApiResponse{
		Code:    http.StatusText(http.StatusCreated),
		Message: "success",
		Data:    data,
	}
}
