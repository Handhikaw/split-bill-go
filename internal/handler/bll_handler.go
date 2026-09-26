package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/middleware"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/service"
)

type BillHandler struct {
	billService service.BillService
}

func NewBillHandler(svc service.BillService) *BillHandler {
	return &BillHandler{billService: svc}
}

func (h *BillHandler) CreateSplitBill(c *gin.Context) {
	var req dto.CreateSpiltBillRq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	// Identitas dari JWT (OptionalAuthMiddleware). Nil = guest.
	actorID, _ := middleware.AuthUserID(c)

	res, err := h.billService.CreateSplitBill(req, actorID)
	if err != nil {
		// Error dikirim ke Middleware
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, model.Success(res))
}

func (h *BillHandler) CreateGroupBill(c *gin.Context) {
	var req dto.CreateGroupBillRq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	// Identitas dari JWT (OptionalAuthMiddleware). Nil = guest.
	actorID, _ := middleware.AuthUserID(c)

	res, err := h.billService.CreateGroupBill(req, actorID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, model.Success(res))
}

func (h *BillHandler) GetSplitBill(c *gin.Context) {
	publicID := c.Param("public_id")

	res, err := h.billService.GetSplitBill(publicID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}

func (h *BillHandler) GetGroupBill(c *gin.Context) {
	publicID := c.Param("public_id")

	res, err := h.billService.GetGroupBill(publicID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}

func (h *BillHandler) JoinSplitBill(c *gin.Context) {
	publicID := c.Param("public_id")

	var req dto.JoinSplitBillRq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	err := h.billService.JoinSplitBill(publicID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}
