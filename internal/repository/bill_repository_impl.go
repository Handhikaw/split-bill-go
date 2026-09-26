package repository

import (
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

type billRepositoryImpl struct {
	db *gorm.DB
}

func NewBillRepository(db *gorm.DB) BillRepository {
	return &billRepositoryImpl{db: db}
}

func (r *billRepositoryImpl) CreateSplitBill(group *entity.Bill, expense *entity.Expense) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(group).Error; err != nil {
			return err
		}
		expense.BillID = group.ID
		return tx.Create(expense).Error
	})
}

func (r *billRepositoryImpl) CreateGroupBill(group *entity.Bill) error {
	return r.db.Create(group).Error
}

func (r *billRepositoryImpl) GetBillByPublicID(publicID string) (*entity.Bill, error) {
	var bill entity.Bill
	// Mencari berdasarkan public_id
	err := r.db.Where("public_id = ?", publicID).First(&bill).Error
	if err != nil {
		return nil, err
	}
	return &bill, nil
}

func (r *billRepositoryImpl) CreateParticipant(participant *entity.Participant) error {
	return r.db.Create(participant).Error
}
