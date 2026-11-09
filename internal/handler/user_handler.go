package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/middleware"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
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
	var req dto.GoogleAuthRq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	currentUserID, _ := middleware.AuthUserID(c)
	res, err := h.service.AuthGoogle(c.Request.Context(), req.IDToken, currentUserID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}

func (h *UserHandler) Me(c *gin.Context) {
	userID, ok := middleware.AuthUserID(c)
	if !ok {
		_ = c.Error(model.NewUnauthorized("login dulu"))
		return
	}

	res, err := h.service.Me(*userID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}
