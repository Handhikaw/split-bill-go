package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/service"
)

type PaymentHandler struct {
	service service.PaymentService
}

func NewPaymentHandler(service service.PaymentService) *PaymentHandler {
	return &PaymentHandler{service: service}
}

func (h *PaymentHandler) ProcessPayment(c *gin.Context) {
	publicID := c.Param("public_id")
	var req dto.PaymentRq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	err := h.service.ProcessPayment(publicID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

func (h *PaymentHandler) ReplaceExpensePayments(c *gin.Context) {
	publicID := c.Param("public_id")
	var req dto.ReplacePaymentsRq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	if err := h.service.ReplaceExpensePayments(publicID, req); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}
