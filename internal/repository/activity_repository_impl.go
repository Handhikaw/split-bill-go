package repository

import (
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

type activityRepository struct {
	db *gorm.DB
}

func NewActivityRepository(db *gorm.DB) ActivityRepository {
	return &activityRepository{db: db}
}

func (r *activityRepository) Save(log entity.ActivityLog) error {
	// GORM Insert
	return r.db.Create(&log).Error
}

// ListByBillID implements [ActivityRepository].
func (r *activityRepository) ListByBillID(billID int64, limit int) ([]entity.ActivityLog, error) {
	var logs []entity.ActivityLog
	q := r.db.Where("bill_id = ?", billID).Order("created_at DESC, id DESC")
	if limit > 0 {
		q = q.Limit(limit)
	}
	if err := q.Find(&logs).Error; err != nil {
		return nil, err
	}
	return logs, nil
}
