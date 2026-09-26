package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/model"
)

func ErrorMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		var apiErr *model.ApiError
		if errors.As(err, &apiErr) {
			c.JSON(apiErr.HttpStatus, model.ApiResponse{
				Code:    apiErr.Code,
				Message: apiErr.Message,
			})
			return
		}

		c.JSON(http.StatusInternalServerError, model.ApiError{
			Code:    "INTERNAL_SERVER_ERROR",
			Message: err.Error(),
		})
	}
}
