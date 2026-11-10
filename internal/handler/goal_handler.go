package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/middleware"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/service"
)

type GoalHandler struct {
	goalService service.GoalService
}

func NewGoalHandler(goalService service.GoalService) *GoalHandler {
	return &GoalHandler{goalService: goalService}
}

func (h *GoalHandler) CreateGoalBill(c *gin.Context) {
	var req dto.CreateGoalBillRq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	actorID, _ := middleware.AuthUserID(c)

	res, err := h.goalService.CreateGoalBill(req, actorID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, model.Success(res))
}

func (h *GoalHandler) GetGoalBill(c *gin.Context) {
	publicID := c.Param("public_id")
	viewerID, _ := middleware.AuthUserID(c)

	res, err := h.goalService.GetGoalBill(publicID, viewerID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}

func (h *GoalHandler) AddContribution(c *gin.Context) {
	publicID := c.Param("public_id")
	actorID := participantIDFromContext(c)

	var req dto.GoalContributionRq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	res, err := h.goalService.AddContribution(publicID, actorID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, model.Success(res))
}

func (h *GoalHandler) DeleteContribution(c *gin.Context) {
	publicID := c.Param("public_id")
	actorID := participantIDFromContext(c)

	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid contribution id"))
		return
	}

	if err := h.goalService.DeleteContribution(publicID, id, actorID); err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}

func (h *GoalHandler) UpdateGoalBill(c *gin.Context) {
	publicID := c.Param("public_id")
	actorID := participantIDFromContext(c)

	var req dto.UpdateGoalBillRq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	res, err := h.goalService.UpdateGoalBill(publicID, actorID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}
