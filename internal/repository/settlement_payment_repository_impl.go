package repository

import (
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

type settlementPaymentRepositoryImpl struct {
	db *gorm.DB
}

func NewSettlementPaymentRepository(db *gorm.DB) SettlementPaymentRepository {
	return &settlementPaymentRepositoryImpl{db: db}
}

func (r *settlementPaymentRepositoryImpl) Create(payment *entity.SettlementPayment) error {
	return r.db.Create(payment).Error
}

func (r *settlementPaymentRepositoryImpl) Delete(payment *entity.SettlementPayment) error {
	return r.db.Delete(payment).Error
}

func (r *settlementPaymentRepositoryImpl) ListByBillID(billID int64) ([]entity.SettlementPayment, error) {
	var rows []entity.SettlementPayment
	err := r.db.Where("bill_id = ?", billID).
		Order("paid_at ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *settlementPaymentRepositoryImpl) FindByIDAndBillID(id, billID int64) (*entity.SettlementPayment, error) {
	var row entity.SettlementPayment
	if err := r.db.Where("id = ? AND bill_id = ?", id, billID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *settlementPaymentRepositoryImpl) CountByParticipantID(participantID int64) (int64, error) {
	var count int64
	err := r.db.Model(&entity.SettlementPayment{}).
		Where("from_participant_id = ? OR to_participant_id = ?", participantID, participantID).
		Count(&count).Error
	return count, err
}
