package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/service"
)

type UserHandler struct {
	service service.UserService
}

func NewUserHandler(service service.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) AuthAnon(c *gin.Context) {
	res, err := h.service.AuthAnon(c.GetHeader("X-Anonymous-Id"))
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}

func (h *UserHandler) AuthGoogle(c *gin.Context) {
	c.Error(model.NewNotImplemented("Google auth is not implemented yet"))
}
