package repository

import (
	"github.com/harkandavez/split-bill-go/internal/model/entity"
	"gorm.io/gorm"
)

type ParticipantRepository interface {
	Upsert(participant *entity.Participant) (int64, error)
	FindByID(id int64) (*entity.Participant, error)
	ExistsByIDAndBillID(participantID int64, billID int64) (bool, error)
	Delete(participant *entity.Participant) error
	GetAffectedExpenseIDsByParticipantID(tx *gorm.DB, participantID int64) ([]int64, error)
	DeleteExpenseItemParticipantsByParticipantID(tx *gorm.DB, participantID int64) error
	DeleteExpenseParticipantsByParticipantID(tx *gorm.DB, participantID int64) error
	DeletePaymentsByParticipantID(tx *gorm.DB, participantID int64) error
	UpsertParticipantShare(tx *gorm.DB, ep *entity.ExpenseParticipant) error
	GetParticipants(billId int64) ([]entity.Participant, error)
}
