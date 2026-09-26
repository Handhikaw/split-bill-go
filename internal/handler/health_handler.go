package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/model"
)

type HealthHandler struct{}

func NewHealthHandler() *HealthHandler {
	return &HealthHandler{}
}

func (h *HealthHandler) Health(c *gin.Context) {
	if false {
		c.Error(model.NewBadRequest("invalid request"))
		return
	}

	c.JSON(http.StatusOK, model.Success("OK"))
}
