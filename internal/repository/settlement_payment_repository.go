package repository

import "github.com/harkandavez/split-bill-go/internal/model/entity"

type SettlementPaymentRepository interface {
	Create(payment *entity.SettlementPayment) error
	Delete(payment *entity.SettlementPayment) error
	ListByBillID(billID int64) ([]entity.SettlementPayment, error)
	FindByIDAndBillID(id, billID int64) (*entity.SettlementPayment, error)
	CountByParticipantID(participantID int64) (int64, error)
}
