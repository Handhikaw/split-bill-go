package service

import (
	"github.com/harkandavez/split-bill-go/internal/model/dto"
	"github.com/harkandavez/split-bill-go/internal/model/entity"
)

type ActivityService interface {
	Log(log entity.ActivityLog)
	// List mengembalikan log aktivitas bill (terbaru dulu) beserta nama aktor.
	List(publicID string, limit int) ([]dto.ActivityItem, error)
}
