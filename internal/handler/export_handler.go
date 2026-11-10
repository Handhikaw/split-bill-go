package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/harkandavez/split-bill-go/internal/middleware"
	"github.com/harkandavez/split-bill-go/internal/service"
)

type ExportHandler struct {
	exportService service.ExportService
}

func NewExportHandler(exportService service.ExportService) *ExportHandler {
	return &ExportHandler{exportService: exportService}
}

func (h *ExportHandler) Export(c *gin.Context) {
	publicID := c.Param("public_id")
	viewerID, _ := middleware.AuthUserID(c)
	format := c.DefaultQuery("format", "csv")

	file, err := h.exportService.Export(publicID, viewerID, format)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, file.Filename))
	c.Data(http.StatusOK, file.ContentType, file.Data)
}
