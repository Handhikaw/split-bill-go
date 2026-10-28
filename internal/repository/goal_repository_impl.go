package repository

import (
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

type goalRepositoryImpl struct {
	db *gorm.DB
}

func NewGoalRepository(db *gorm.DB) GoalRepository {
	return &goalRepositoryImpl{db: db}
}

func (r *goalRepositoryImpl) GetGoalByBillID(billID int64) (*entity.GoalBill, error) {
	var goal entity.GoalBill
	if err := r.db.Where("bill_id = ?", billID).First(&goal).Error; err != nil {
		return nil, err
	}
	return &goal, nil
}

func (r *goalRepositoryImpl) ListContributions(billID int64) ([]entity.Contribution, error) {
	var rows []entity.Contribution
	err := r.db.Where("bill_id = ?", billID).
		Order("paid_at ASC, id ASC").
		Find(&rows).Error
	return rows, err
}

func (r *goalRepositoryImpl) SumContributions(billID int64) (int64, error) {
	var total int64
	err := r.db.Model(&entity.Contribution{}).
		Select("COALESCE(SUM(amount), 0)").
		Where("bill_id = ?", billID).
		Scan(&total).Error
	return total, err
}

func (r *goalRepositoryImpl) CreateContribution(contribution *entity.Contribution) error {
	return r.db.Create(contribution).Error
}

func (r *goalRepositoryImpl) FindContributionByIDAndBillID(id, billID int64) (*entity.Contribution, error) {
	var row entity.Contribution
	if err := r.db.Where("id = ? AND bill_id = ?", id, billID).First(&row).Error; err != nil {
		return nil, err
	}
	return &row, nil
}

func (r *goalRepositoryImpl) DeleteContribution(contribution *entity.Contribution) error {
	return r.db.Delete(contribution).Error
}

func (r *goalRepositoryImpl) CountContributionsByParticipantID(participantID int64) (int64, error) {
	var count int64
	err := r.db.Model(&entity.Contribution{}).
		Where("participant_id = ?", participantID).
		Count(&count).Error
	return count, err
}
