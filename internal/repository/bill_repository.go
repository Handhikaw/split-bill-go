package repository

import "github.com/harkandavez/split-bill-go/internal/model/entity"

type BillRepository interface {
	CreateSplitBill(group *entity.Bill, expense *entity.Expense) error
	CreateGroupBill(group *entity.Bill) error
	GetBillByPublicID(publicID string) (*entity.Bill, error)
	CreateParticipant(participant *entity.Participant) error
}
