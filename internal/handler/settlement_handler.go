package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/service"
)

type SettlementHandler struct {
	settlementService service.SettlementService
}

func NewSettlementHandler(settlementService service.SettlementService) *SettlementHandler {
	return &SettlementHandler{settlementService: settlementService}
}

func (h *SettlementHandler) Create(c *gin.Context) {
	publicID := c.Param("public_id")
	actorID := participantIDFromContext(c)

	var req dto.SettlementPaymentRq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	res, err := h.settlementService.Create(publicID, actorID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, model.Success(res))
}

func (h *SettlementHandler) Delete(c *gin.Context) {
	publicID := c.Param("public_id")
	actorID := participantIDFromContext(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid settlement id"))
		return
	}

	if err := h.settlementService.Delete(publicID, id, actorID); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}
