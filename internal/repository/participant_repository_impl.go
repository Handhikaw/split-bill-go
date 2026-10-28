package repository

import (
	"time"

	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type participantRepositoryImpl struct {
	db *gorm.DB
}

func NewParticipantRepository(db *gorm.DB) ParticipantRepository {
	return &participantRepositoryImpl{db: db}
}

func (p *participantRepositoryImpl) Upsert(participant *entity.Participant) (int64, error) {
	err := p.db.Save(participant).Error
	if err != nil {
		return 0, err
	}

	return participant.ID, nil
}

func (p *participantRepositoryImpl) FindByID(id int64) (*entity.Participant, error) {
	var participant entity.Participant
	if err := p.db.First(&participant, id).Error; err != nil {
		return nil, err
	}
	return &participant, nil
}

// SetUserID implements [ParticipantRepository].
func (p *participantRepositoryImpl) SetUserID(id int64, userID *int64) error {
	if userID == nil {
		return nil
	}
	return p.db.Model(&entity.Participant{}).Where("id = ?", id).Update("user_id", *userID).Error
}

func (p *participantRepositoryImpl) ExistsByIDAndBillID(participantID int64, billID int64) (bool, error) {
	var count int64
	err := p.db.Model(&entity.Participant{}).
		Where("id = ? AND bill_id = ?", participantID, billID).
		Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

func (p *participantRepositoryImpl) Delete(participant *entity.Participant) error {
	return p.db.Delete(participant).Error
}

func (p *participantRepositoryImpl) GetAffectedExpenseIDsByParticipantID(tx *gorm.DB, participantID int64) ([]int64, error) {
	var expenseIDs []int64
	err := tx.Raw(`
		SELECT DISTINCT expense_id
		FROM expense_participant
		WHERE participant_id = ?
		UNION
		SELECT DISTINCT ei.expense_id
		FROM expense_item_participant eip
		JOIN expense_item ei ON ei.id = eip.item_id
		WHERE eip.participant_id = ?
	`, participantID, participantID).Scan(&expenseIDs).Error
	if err != nil {
		return nil, err
	}
	return expenseIDs, nil
}

func (p *participantRepositoryImpl) DeleteExpenseItemParticipantsByParticipantID(tx *gorm.DB, participantID int64) error {
	return tx.Where("participant_id = ?", participantID).Delete(&entity.ExpenseItemParticipant{}).Error
}

func (p *participantRepositoryImpl) DeleteExpenseParticipantsByParticipantID(tx *gorm.DB, participantID int64) error {
	return tx.Where("participant_id = ?", participantID).Delete(&entity.ExpenseParticipant{}).Error
}

func (p *participantRepositoryImpl) DeletePaymentsByParticipantID(tx *gorm.DB, participantID int64) error {
	return tx.Where("paid_by = ?", participantID).Delete(&entity.Payment{}).Error
}

// UpsertParticipantShare implements [ParticipantRepository].
func (p *participantRepositoryImpl) UpsertParticipantShare(tx *gorm.DB, ep *entity.ExpenseParticipant) error {
	return tx.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "expense_id"}, {Name: "participant_id"}},
		DoUpdates: clause.Assignments(map[string]interface{}{
			"share_amount": gorm.Expr("expense_participant.share_amount + ?", ep.ShareAmount),
			"updated_at":   time.Now(), // Bagus juga untuk update manual timestamp-nya di sini
		}),
	}).Create(ep).Error
}

// GetParticipants implements [ParticipantRepository].
func (p *participantRepositoryImpl) GetParticipants(billId int64) ([]entity.Participant, error) {
	var participants []entity.Participant
	err := p.db.Where("bill_id = ?", billId).Find(&participants).Error
	return participants, err
}
