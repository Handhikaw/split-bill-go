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

type ParticipantHandler struct {
	participantService service.ParticipantService
}

func NewParticipantHandler(svc service.ParticipantService) *ParticipantHandler {
	return &ParticipantHandler{participantService: svc}
}

func (h *ParticipantHandler) Participant(c *gin.Context) {
	publicID := c.Param("public_id")
	participantID := participantIDFromContext(c)
	var req dto.ParticipantRq

	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(model.NewBadRequest("invalid request body: " + err.Error()))
		return
	}

	authUserID, _ := middleware.AuthUserID(c)
	res, err := h.participantService.UpsertParticipant(publicID, participantID, authUserID, req)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, model.Success(res))
}

func (h *ParticipantHandler) RemoveParticipant(c *gin.Context) {
	publicID := c.Param("public_id")
	idStr := c.Param("id")
	participantID := participantIDFromContext(c)

	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		_ = c.Error(model.NewBadRequest("invalid participant id"))
		return
	}

	err = h.participantService.RemoveParticipant(publicID, id, participantID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(nil))
}
