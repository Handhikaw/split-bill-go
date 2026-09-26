package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/model"
	"github.com/harkandavez/split-bill-go/internal/service"
)

type ActivityHandler struct {
	activityService service.ActivityService
}

func NewActivityHandler(svc service.ActivityService) *ActivityHandler {
	return &ActivityHandler{activityService: svc}
}

// GetActivities mengembalikan log aktivitas bill, terbaru dulu.
// Publik seperti GetGroupBill (model share-link). ?limit= (default 20, maks 50).
func (h *ActivityHandler) GetActivities(c *gin.Context) {
	publicID := c.Param("public_id")

	limit := 20
	if raw := c.Query("limit"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && n > 0 {
			if n > 50 {
				n = 50
			}
			limit = n
		}
	}

	res, err := h.activityService.List(publicID, limit)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, model.Success(res))
}
