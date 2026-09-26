package repository

import "github.com/harkandavez/split-bill-go/internal/model/entity"

type ActivityRepository interface {
	Save(log entity.ActivityLog) error
	// ListByBillID mengembalikan log terbaru dulu. limit <= 0 = tanpa batas.
	ListByBillID(billID int64, limit int) ([]entity.ActivityLog, error)
}
